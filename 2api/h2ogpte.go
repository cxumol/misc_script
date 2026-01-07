package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// --- Configuration ---

type Config struct {
	BaseURL          string
	IsGuest          bool
	Session          string
	CsrfToken        string
	WorkspaceID      string
	PromptTemplateID string
	Host             string
	Port             int
	APIKey           string
	ReasoningEffort  int
}

var cfg Config

func loadConfig() {
	getEnv := func(key, def string) string {
		if v := os.Getenv(key); v != "" {
			return v
		}
		return def
	}

	isGuest := strings.ToLower(getEnv("IS_GUEST", "true"))
	cfg = Config{
		BaseURL:          getEnv("H2OGPTE_BASE_URL", "https://h2ogpte.genai.h2o.ai"),
		IsGuest:          isGuest == "true" || isGuest == "1" || isGuest == "yes",
		Session:          getEnv("H2OGPTE_SESSION", ""),
		CsrfToken:        getEnv("H2OGPTE_CSRF_TOKEN", ""),
		WorkspaceID:      getEnv("H2OGPTE_WORKSPACE_ID", "workspaces/h2ogpte-guest"),
		PromptTemplateID: getEnv("H2OGPTE_PROMPT_TEMPLATE_ID", ""),
		Host:             getEnv("HOST", "127.0.0.1"),
		APIKey:           getEnv("API_KEY", ""),
	}

	port, _ := strconv.Atoi(getEnv("PORT", "2156"))
	cfg.Port = port

	re, _ := strconv.Atoi(getEnv("REASONING_EFFORT", "65000"))
	cfg.ReasoningEffort = re
}

// --- UUID Helper ---

func genUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func genHexID(prefix string) string {
	b := make([]byte, 12)
	rand.Read(b)
	return prefix + hex.EncodeToString(b)
}

// --- Credential Store ---

type StoredCredential struct {
	Session    string `json:"session"`
	CsrfToken  string `json:"csrf_token"`
	UserID     string `json:"user_id"`
	Username   string `json:"username"`
	CreatedAt  string `json:"created_at"`
	LastUsedAt string `json:"last_used_at"`
}

type CredentialStore struct {
	mu      sync.Mutex
	path    string
	current *StoredCredential
}

var credStore *CredentialStore

func initCredStore() {
	path := filepath.Join(".", "guest_credentials.json")
	credStore = &CredentialStore{path: path}
	credStore.load()
}

func (cs *CredentialStore) load() {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	data, err := os.ReadFile(cs.path)
	if err == nil {
		var sc StoredCredential
		if json.Unmarshal(data, &sc) == nil {
			cs.current = &sc
		}
	}
}

func (cs *CredentialStore) save(session, csrf, uid, uname string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	now := time.Now().Format(time.RFC3339)
	sc := StoredCredential{
		Session: session, CsrfToken: csrf, UserID: uid, Username: uname,
		CreatedAt: now, LastUsedAt: now,
	}
	cs.current = &sc
	data, _ := json.MarshalIndent(sc, "", "  ")
	os.WriteFile(cs.path, data, 0644)
}

func (cs *CredentialStore) get() (string, string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	if cfg.IsGuest {
		if cs.current != nil {
			cs.current.LastUsedAt = time.Now().Format(time.RFC3339)
			return cs.current.Session, cs.current.CsrfToken
		}
		return "", ""
	}
	return cfg.Session, cfg.CsrfToken
}

// --- H2OGPTE Client ---

type H2Client struct {
	client     *http.Client
	refreshMu  sync.Mutex
	refreshing bool
}

var h2Client *H2Client

func initH2Client() {
	// Main client for RPC calls (Manual cookie handling)
	h2Client = &H2Client{
		client: &http.Client{
			Timeout: 60 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				IdleConnTimeout:     30 * time.Second,
				DisableKeepAlives:   false,
			},
		},
	}
}

func (c *H2Client) getHeaders(csrf string) http.Header {
	h := http.Header{}
	h.Set("Accept", "*/*")
	h.Set("Content-Type", "application/json")
	h.Set("Origin", cfg.BaseURL)
	h.Set("X-Csrf-Token", csrf)
	h.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	return h
}

func (c *H2Client) ensureCreds() error {
	sess, csrf := credStore.get()
	if sess != "" && csrf != "" {
		return nil
	}
	if cfg.IsGuest {
		return c.refreshCredentials(false)
	}
	return fmt.Errorf("missing static credentials")
}

func (c *H2Client) refreshCredentials(forceNew bool) error {
	c.refreshMu.Lock()
	defer c.refreshMu.Unlock()

	// Double check inside lock
	sess, csrf := credStore.get()
	if !forceNew && sess != "" && csrf != "" && !c.refreshing {
		return nil
	}

	c.refreshing = true
	defer func() { c.refreshing = false }()

	// Use a FRESH client with a CookieJar for the handshake to handle redirects
	jar, _ := cookiejar.New(nil)
	handshakeClient := &http.Client{
		Timeout: 30 * time.Second,
		Jar:     jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Mimic browser behavior, ensure headers persist on redirect
			req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
			return nil
		},
	}

	targetURL := cfg.BaseURL + "/chats"
	req, _ := http.NewRequest("GET", targetURL, nil)
	
	// Headers exactly as Python script
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	// If renewing, attach existing cookie
	if !forceNew && sess != "" {
		// We add it to the jar so it handles it properly across redirects
		if u, err := req.URL.Parse(cfg.BaseURL); err == nil {
			jar.SetCookies(u, []*http.Cookie{{Name: "h2ogpte.session", Value: sess}})
		}
	}

	resp, err := handshakeClient.Do(req)
	if err != nil {
		return fmt.Errorf("handshake failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("handshake status: %d", resp.StatusCode)
	}

	// Extract new session from the jar (handling redirects)
	var newSess string
	u, _ := req.URL.Parse(cfg.BaseURL)
	for _, cookie := range jar.Cookies(u) {
		if cookie.Name == "h2ogpte.session" {
			newSess = cookie.Value
			break
		}
	}
	
	// Fallback: check response cookies directly if jar didn't catch it for some reason
	if newSess == "" {
		for _, cookie := range resp.Cookies() {
			if cookie.Name == "h2ogpte.session" {
				newSess = cookie.Value
				break
			}
		}
	}

	if newSess == "" && !forceNew {
		newSess = sess
	}
	if newSess == "" {
		return fmt.Errorf("failed to retrieve session cookie")
	}

	// Extract CSRF & User info from HTML
	bodyBytes, _ := io.ReadAll(resp.Body)
	html := string(bodyBytes)
	marker := "data-conf='"
	idx := strings.Index(html, marker)
	if idx == -1 {
		return fmt.Errorf("failed to parse config from html (marker not found)")
	}
	start := idx + len(marker)
	end := strings.Index(html[start:], "'")
	if end == -1 {
		return fmt.Errorf("failed to parse config end")
	}
	jsonConf := html[start : start+end]

	var confData struct {
		CsrfToken string `json:"csrf_token"`
		UserID    string `json:"user_id"`
		Username  string `json:"username"`
	}
	if err := json.Unmarshal([]byte(jsonConf), &confData); err != nil {
		return fmt.Errorf("json parse error: %v", err)
	}

	credStore.save(newSess, confData.CsrfToken, confData.UserID, confData.Username)
	log.Printf("Refreshed credentials for %s", confData.Username)
	return nil
}

func (c *H2Client) rpcDB(method string, args ...interface{}) (interface{}, error) {
	if err := c.ensureCreds(); err != nil {
		return nil, err
	}

	doReq := func() (*http.Response, error) {
		sess, csrf := credStore.get()
		
		payloadData := make([]interface{}, 0, 1+len(args))
		payloadData = append(payloadData, method)
		payloadData = append(payloadData, args...)
		
		payload, _ := json.Marshal(payloadData)
		req, _ := http.NewRequest("POST", cfg.BaseURL+"/rpc/db", bytes.NewBuffer(payload))
		req.Header = c.getHeaders(csrf)
		req.AddCookie(&http.Cookie{Name: "h2ogpte.session", Value: sess})
		return c.client.Do(req)
	}

	resp, err := doReq()
	if err != nil {
		return nil, err
	}

	// Handle 401/429
	if resp.StatusCode == 401 || (resp.StatusCode == 429 && cfg.IsGuest) {
		resp.Body.Close()
		log.Println("Unauthorized or Quota exceeded, refreshing...")
		if err := c.refreshCredentials(resp.StatusCode == 429); err != nil {
			return nil, err
		}
		resp, err = doReq()
		if err != nil {
			return nil, err
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("RPC failed: %d", resp.StatusCode)
	}

	var result interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *H2Client) createChatSession() (string, error) {
	ws := cfg.WorkspaceID
	res, err := c.rpcDB("create_chat_session", nil, ws)
	if err != nil {
		log.Printf("Error creating chat session (RPC): %v", err)
		return genUUID(), err 
	}
	if m, ok := res.(map[string]interface{}); ok {
		if id, ok := m["id"].(string); ok {
			return id, nil
		}
	}
	if s, ok := res.(string); ok {
		return s, nil
	}
	return genUUID(), nil
}

func (c *H2Client) deleteChatSession(id string) {
	sess, csrf := credStore.get()
	payload, _ := json.Marshal([]interface{}{
		"q:crawl_quick.DeleteChatSessionsJob",
		map[string]interface{}{
			"name":             "Deleting Chat Sessions",
			"chat_session_ids": []string{id},
		},
	})
	req, _ := http.NewRequest("POST", cfg.BaseURL+"/rpc/job", bytes.NewBuffer(payload))
	req.Header = c.getHeaders(csrf)
	req.AddCookie(&http.Cookie{Name: "h2ogpte.session", Value: sess})
	if resp, err := c.client.Do(req); err == nil {
		resp.Body.Close()
	}
}

// --- Session Manager ---

type SessionManager struct {
	pool    chan string
	cleanup chan string
	target  int
}

var sessMgr *SessionManager

func initSessionManager() {
	sessMgr = &SessionManager{
		pool:    make(chan string, 20),
		cleanup: make(chan string, 100),
		target:  5,
	}

	// Maintainer
	go func() {
		for {
			if len(sessMgr.pool) < sessMgr.target {
				if id, err := h2Client.createChatSession(); err == nil {
					sessMgr.pool <- id
				} else {
					// Don't spam logs if network is down
					time.Sleep(5 * time.Second)
				}
			} else {
				time.Sleep(2 * time.Second)
			}
		}
	}()

	// Cleanup
	go func() {
		for id := range sessMgr.cleanup {
			h2Client.deleteChatSession(id)
		}
	}()
}

func (sm *SessionManager) Get() string {
	select {
	case id := <-sm.pool:
		return id
	default:
		id, _ := h2Client.createChatSession()
		return id
	}
}

func (sm *SessionManager) Recycle(id string) {
	sm.cleanup <- id
}

// --- WebSocket Chat ---

func runChat(chatID, msg, model, sysPrompt string, temp float64, maxTokens int, callback func(string, bool) error) error {
	sess, _ := credStore.get()
	if sess == "" {
		return fmt.Errorf("no valid session available for websocket")
	}

	wsURL := strings.Replace(cfg.BaseURL, "https://", "wss://", 1) + "/ws?currentSessionID=" + chatID

	header := http.Header{}
	header.Set("Cookie", "h2ogpte.session="+sess)
	header.Set("Origin", cfg.BaseURL)
	header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = 15 * time.Second

	conn, _, err := dialer.Dial(wsURL, header)
	if err != nil {
		return fmt.Errorf("websocket dial failed: %w", err)
	}
	defer conn.Close()

	if maxTokens == 0 {
		maxTokens = 32768
	}
	if model == "" {
		model = "auto"
	}

	llmArgs := map[string]interface{}{
		"max_new_tokens":        maxTokens,
		"enable_vision":         "auto",
		"visible_vision_models": []string{"auto"},
		"use_agent":             false,
		"reasoning_effort":      cfg.ReasoningEffort,
		"cost_controls":         map[string]interface{}{"max_cost": 0.05},
		"remove_non_private":    false,
		"temperature":           temp,
	}
	llmArgsJSON, _ := json.Marshal(llmArgs)
	ragConfig := map[string]interface{}{"rag_type": "auto", "num_neighbor_chunks_to_include": 1}
	ragJSON, _ := json.Marshal(ragConfig)

	req := map[string]interface{}{
		"t":                      "cq",
		"mode":                   "s",
		"session_id":             chatID,
		"correlation_id":         genUUID(),
		"body":                   msg,
		"llm":                    model,
		"llm_args":               string(llmArgsJSON),
		"self_reflection_config": "null",
		"rag_config":             string(ragJSON),
		"include_chat_history":   "auto",
		"tags":                   []string{},
		"prompt_template_id":     nil,
	}
	if cfg.PromptTemplateID != "" {
		req["prompt_template_id"] = cfg.PromptTemplateID
	}
	if sysPrompt != "" {
		req["system_prompt"] = sysPrompt
	}

	if err := conn.WriteJSON(req); err != nil {
		return err
	}

	collected := ""
	for {
		conn.SetReadDeadline(time.Now().Add(120 * time.Second))
		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var data map[string]interface{}
		if err := json.Unmarshal(msgBytes, &data); err != nil {
			continue
		}

		t, _ := data["t"].(string)
		if t == "cp" { // partial
			if body, ok := data["body"].(string); ok {
				if err := callback(body, false); err != nil {
					return err
				}
				collected += body
			}
		} else if t == "cr" { // accumulated response
			if collected == "" {
				if body, ok := data["body"].(string); ok {
					callback(body, false)
				}
			}
		} else if t == "ca" || t == "cd" { // answer metadata or done
			break
		} else if t == "ce" {
			errMsg := "unknown error"
			if e, ok := data["error"]; ok {
				errMsg = fmt.Sprintf("%v", e)
			} else if b, ok := data["body"]; ok {
				errMsg = fmt.Sprintf("%v", b)
			}
			return fmt.Errorf("chat error: %s", errMsg)
		}
	}
	return nil
}

// --- API Models ---

type OpenAIModel struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature"`
	Stream      bool      `json:"stream"`
	MaxTokens   int       `json:"max_tokens"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type Chunk struct {
	ID      string        `json:"id"`
	Object  string        `json:"object"`
	Created int64         `json:"created"`
	Model   string        `json:"model"`
	Choices []ChunkChoice `json:"choices"`
}

type ChunkChoice struct {
	Index  int         `json:"index"`
	Delta  Delta       `json:"delta"`
	Finish interface{} `json:"finish_reason"` // can be null
}

type Delta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// --- HTTP Handlers ---

func cors(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "*")
}

func verifyKey(r *http.Request) bool {
	if cfg.APIKey == "" {
		return true
	}
	auth := r.Header.Get("Authorization")
	token := strings.TrimPrefix(auth, "Bearer ")
	return token == cfg.APIKey
}

func handleModels(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method == "OPTIONS" {
		return
	}
	if !verifyKey(r) {
		http.Error(w, "Unauthorized", 401)
		return
	}

	list := []string{
		"auto", "gpt-4o", "gpt-4o-mini", "gpt-4.1", "gpt-4.1-mini", "gpt-4.1-nano", "gpt-5", "o4-mini", "o3",
        "gemini-2.5-pro", "gemini-2.5-flash",
        "claude-sonnet-4-5-20250929", "claude-sonnet-4-20250514", "claude-opus-4-20250514", "claude-opus-4-1-20250805",
        "claude-3-7-sonnet-20250219", "claude-3-5-sonnet-20241022", "claude-3-5-haiku-20241022",
        "deepseek-ai/DeepSeek-R1", "deepseek-ai/DeepSeek-V3",
        "meta-llama/Meta-Llama-3.1-8B-Instruct", "meta-llama/Meta-Llama-3.1-405B-Instruct", "meta-llama/Meta-Llama-3.1-70B-Instruct",
        "meta-llama/Llama-3.3-70B-Instruct", "meta-llama/Llama-3.2-11B-Vision-Instruct", "meta-llama/Llama-3.2-90B-Vision-Instruct",
        "meta-llama/Llama-4-Maverick-17B-128E-Instruct", "meta-llama/Llama-4-Scout-17B-16E-Instruct",
        "mistralai/Mixtral-8x7B-Instruct-v0.1", "mistralai/Mistral-7B-Instruct-v0.2", "pixtral-large-2502",
        "openai/gpt-oss-20b", "openai/gpt-oss-120b", "Llama-3_3-Nemotron-Super-49B-v1_5",
	}

	models := make([]OpenAIModel, len(list))
	for i, id := range list {
		models[i] = OpenAIModel{ID: id, Object: "model", Created: time.Now().Unix(), OwnedBy: "h2ogpte"}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"object": "list", "data": models})
}

func handleChat(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method == "OPTIONS" {
		return
	}
	if !verifyKey(r) {
		http.Error(w, "Unauthorized", 401)
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	// Build context
	var sysPrompt string
	var parts []string
	for _, m := range req.Messages {
		if m.Role == "system" {
			sysPrompt = m.Content
		}
		if m.Role == "user" {
			parts = append(parts, "User: "+m.Content)
		}
		if m.Role == "assistant" {
			parts = append(parts, "Assistant: "+m.Content)
		}
	}

	fullMsg := ""
	if len(parts) == 1 {
		// Just last user message content if only one message
		if len(req.Messages) > 0 {
			fullMsg = req.Messages[len(req.Messages)-1].Content
		}
	} else {
		fullMsg = strings.Join(parts, "\n")
		if !strings.HasSuffix(fullMsg, "Assistant: ") {
			fullMsg += "\nAssistant:"
		}
	}
	if fullMsg == "" && len(req.Messages) > 0 {
		fullMsg = req.Messages[len(req.Messages)-1].Content
	}

	chatID := sessMgr.Get()
	defer sessMgr.Recycle(chatID)

	complID := genHexID("chatcmpl-")
	created := time.Now().Unix()

	if req.Stream {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming not supported", 500)
			return
		}

		// Send initial role
		startChunk := Chunk{
			ID: complID, Object: "chat.completion.chunk", Created: created, Model: req.Model,
			Choices: []ChunkChoice{{Index: 0, Delta: Delta{Role: "assistant"}, Finish: nil}},
		}
		b, _ := json.Marshal(startChunk)
		fmt.Fprintf(w, "data: %s\n\n", b)
		flusher.Flush()

		err := runChat(chatID, fullMsg, req.Model, sysPrompt, req.Temperature, req.MaxTokens, func(text string, isErr bool) error {
			chunk := Chunk{
				ID: complID, Object: "chat.completion.chunk", Created: created, Model: req.Model,
				Choices: []ChunkChoice{{Index: 0, Delta: Delta{Content: text}, Finish: nil}},
			}
			b, _ := json.Marshal(chunk)
			fmt.Fprintf(w, "data: %s\n\n", b)
			flusher.Flush()
			return nil
		})

		if err != nil {
			log.Printf("Stream Error: %v", err)
			errChunk := Chunk{
				ID:      complID,
				Object:  "chat.completion.chunk",
				Created: created,
				Model:   req.Model,
				Choices: []ChunkChoice{{Delta: Delta{Content: fmt.Sprintf("[Error: %v]", err)}}},
			}
			b, _ := json.Marshal(errChunk)
			fmt.Fprintf(w, "data: %s\n\n", b)
		}

		endChunk := Chunk{
			ID: complID, Object: "chat.completion.chunk", Created: created, Model: req.Model,
			Choices: []ChunkChoice{{Index: 0, Delta: Delta{}, Finish: "stop"}},
		}
		b, _ = json.Marshal(endChunk)
		fmt.Fprintf(w, "data: %s\n\n", b)
		fmt.Fprintf(w, "data: [DONE]\n\n")
		flusher.Flush()

	} else {
		var fullResp strings.Builder
		err := runChat(chatID, fullMsg, req.Model, sysPrompt, req.Temperature, req.MaxTokens, func(text string, isErr bool) error {
			fullResp.WriteString(text)
			return nil
		})

		if err != nil {
			log.Printf("Chat Error: %v", err)
			http.Error(w, err.Error(), 500)
			return
		}

		resp := ChatResponse{
			ID: complID, Object: "chat.completion", Created: created, Model: req.Model,
			Choices: []Choice{{Index: 0, Message: Message{Role: "assistant", Content: fullResp.String()}, FinishReason: "stop"}},
			Usage:   Usage{PromptTokens: len(fullMsg) / 4, CompletionTokens: fullResp.Len() / 4, TotalTokens: (len(fullMsg) + fullResp.Len()) / 4},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

// --- Main ---

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	loadConfig()
	initCredStore()
	initH2Client()

	log.Printf("🚀 H2OGPTE to OpenAI API Service")
	log.Printf("📡 Listening on %s:%d", cfg.Host, cfg.Port)
	log.Printf("🔗 Target: %s", cfg.BaseURL)

	if cfg.IsGuest {
		if err := h2Client.ensureCreds(); err != nil {
			log.Printf("⚠ Initial credential check failed (will retry on request): %v", err)
		} else {
			log.Println("✓ Guest credentials initialized")
		}
	}

	initSessionManager()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{"message": "H2OGPTE Proxy", "endpoints": []string{"/v1/models", "/v1/chat/completions"}})
	})
	mux.HandleFunc("/v1/models", handleModels)
	mux.HandleFunc("/v1/models/", func(w http.ResponseWriter, r *http.Request) { // individual model
		if !verifyKey(r) {
			http.Error(w, "Unauthorized", 401)
			return
		}
		id := strings.TrimPrefix(r.URL.Path, "/v1/models/")
		json.NewEncoder(w).Encode(OpenAIModel{ID: id, Object: "model", Created: time.Now().Unix(), OwnedBy: "h2ogpte"})
	})
	mux.HandleFunc("/v1/chat/completions", handleChat)

	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Handler: mux,
	}
	log.Fatal(srv.ListenAndServe())
}
