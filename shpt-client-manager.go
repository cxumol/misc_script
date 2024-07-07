package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	// "sync"
	"syscall"
	"time"
)

const (
	LogFormat = "Jan 02 15:04:05"
	TryFormat = "15:04"
)

type Command struct {
	Line    int
	Text    string
	Cmd     *exec.Cmd
	Log     string
	Started bool
}

var isLocal bool

func getTime() time.Time {
	if isLocal {
		return time.Now()
	} else {
		return time.Now().UTC()
	}
}

func getLoc() *time.Location {
	if isLocal {
		return time.Local
	} else {
		return time.UTC
	}
}

// readCommands 从文件中读取命令
func readCommands(filename string) ([]*Command, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var commands []*Command
	scanner := bufio.NewScanner(file)
	for i := 0; scanner.Scan(); i++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		commands = append(commands, &Command{
			Line: i,
			Text: line,
		})
	}
	return commands, scanner.Err()
}

// startCommand 启动命令，并返回一个输出通道
func startCommand(cmd *Command) (<-chan string, error) {
	cmd.Cmd = exec.Command("bash", "-c", cmd.Text)
	// parts := strings.Split(cmd.Text, " ")
	// cmd.Cmd = exec.Command(parts[0], parts[1:]...)
	stdout, err := cmd.Cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	err = cmd.Cmd.Start()
	if err != nil {
		return nil, err
	}
	cmd.Started = true

	ch := make(chan string)
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			ch <- scanner.Text()
		}
		close(ch)
	}()

	return ch, nil
}

// parseLog 解析日志行，返回时间和尾部时间
func parseLog(log string) (time.Time, time.Time, error) {
	// 如果 "No job available" 不在日志中，返回一个错误
	if !strings.Contains(log, "No job available") {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid log format")
	}

	log = strings.TrimSpace(log)
	log = strings.ReplaceAll(log, "\r", "")

	parts := strings.Split(log, " ")
	if len(parts) < 9 {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid log format")
	}

	logTime, err := time.ParseInLocation(LogFormat, strings.Join(parts[:3], " "), getLoc())
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	// 因为 logTime 缺失了年份数据, 所以用当前的年份数据显示
	logTime = time.Date(getTime().Year(), logTime.Month(), logTime.Day(), logTime.Hour(), logTime.Minute(), logTime.Second(), 0, getLoc())

	tryTime, err := time.ParseInLocation(TryFormat, parts[len(parts)-1], getLoc())
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	// 设置 tryTime 的年份、月份和日份与 logTime 相同
	tryTime = time.Date(getTime().Year(), logTime.Month(), logTime.Day(), tryTime.Hour(), tryTime.Minute(), logTime.Second(), 0, getLoc())

	// 如果 tryTime 在 logTime 之前，说明它应该是明天的时间
	if tryTime.Before(logTime) {
		tryTime = tryTime.Add(24 * time.Hour)
	}

	return logTime, tryTime, nil
}

// isProcessAlive 检查命令关联的进程是否仍在运行
func isProcessAlive(cmd *exec.Cmd) bool {
	if cmd.Process == nil {
		return false
	}

	// 尝试向进程发送一个信号0来检查进程是否存在
	// 如果没有错误返回，说明进程仍在运行
	err := cmd.Process.Signal(syscall.Signal(0))
	return err == nil
}

// checkConditionA 检查所有已在运行的命令的最后一行是否符合特定格式
func checkConditionA(commands []*Command) bool {
	for _, cmd := range commands {
		if !cmd.Started {
			fmt.Println("[debug] cmd not started ", cmd.Line)
			continue
		}
		if !isProcessAlive(cmd.Cmd) {
			fmt.Println("[debug] process not alive ", cmd.Line)
			cmd.Started = false // 更新 cmd.Started 状态为 false，因为进程已经不在运行
			continue
		}
		_, _, err := parseLog(cmd.Log)
		if err != nil {
			fmt.Println("[warn] parseLog ", err)
			return false
		}
	}
	return true
}

// findStoppedCommand 查找已停止的命令
func findStoppedCommand(commands []*Command) *Command {
	for _, cmd := range commands {
		isProcessErr := strings.Contains(cmd.Log, "(error) Error UNKNOWN")
		if !isProcessAlive(cmd.Cmd) || isProcessErr {
			return cmd
		}
	}

	// 检查命令是否已经执行结束（无论是正常结束还是异常退出）
	// if cmd.Cmd.ProcessState != nil && cmd.Cmd.ProcessState.Exited() {
	// 	return cmd
	// }
	return nil
}

// findLatestTryAgain 找到最晚尝试再次运行的命令
func findLatestTryAgain(commands []*Command, tryDelay time.Duration) *Command {
	var latestCmd *Command
	var latestTry time.Time
	for _, cmd := range commands {
		if cmd.Log == "" {
			continue
		}

		_, tryTime, err := parseLog(cmd.Log)
		if err != nil {
			continue
		}

		// 如果 tryTime 在当前时间 5 分钟以内，跳过这个命令
		if getTime().Add(tryDelay).After(tryTime) {
			// fmt.Printf("[pid %d] delayedTime= %s |AFTER=SKIP| tryTime= %s\t", cmd.Line, time.Now().UTC().Add(tryDelay), tryTime)
			// time.Sleep(1 * time.Minute)
			continue
		}
		// fmt.Printf("[pid %d] delayedTime= %s |BEFORE=RESTART| tryTime= %s\t", cmd.Line, time.Now().UTC().Add(tryDelay), tryTime)

		fmt.Printf("tryTime [", cmd.Line ,"]:", tryTime)
		if latestCmd == nil || tryTime.After(latestTry) {
			latestCmd = cmd
			latestTry = tryTime
			fmt.Println("latestTryTime update:", cmd.Line, ":", tryTime)
		}
	}
	return latestCmd
}

// findEarliestLog 在具有相同尾部时间的命令中找到最早的日志
func findEarliestLog(commands []*Command, latestTry time.Time) *Command {
	var earliestCmd *Command
	var earliestLog time.Time
	for _, cmd := range commands {
		if cmd.Log == "" {
			continue
		}

		logTime, tryTime, err := parseLog(cmd.Log)
		if err != nil || !tryTime.Equal(latestTry) {
			continue
		}

		if earliestCmd == nil || logTime.Before(earliestLog) {
			earliestCmd = cmd
			earliestLog = logTime
		}
	}
	return earliestCmd
}

// restartCommand 重新启动命令
func restartCommand(cmd *Command) error {
	fmt.Printf("[Restart %d] ...\n", cmd.Line)
	if isProcessAlive(cmd.Cmd) || !cmd.Started {
		cmd.Cmd.Process.Kill()
		cmd.Cmd.Wait()
	}
	cmd.Started = false
	cmd.Log = ""

	ch, err := startCommand(cmd)
	if err != nil {
		return err
	}
	go func() {
		for log := range ch {
			fmt.Printf("[process %d] %s\n", cmd.Line, log)
			cmd.Log = log
		}
	}()

	return nil
}

func main() {
	// 读取命令
	cmdFile := flag.String("c", "commands.txt", "file containing the commands to run")
	tryDelay := flag.Duration("t", 5*time.Minute, "the delay before a command is tried again")
	flag.BoolVar(&isLocal, "local", false, "set isLocal to true")
	flag.Parse()
	// commands, err := readCommands("commands.txt")
	commands, err := readCommands(*cmdFile)
	if err != nil {
		fmt.Println("Error reading commands:", err)
		return
	}

	// 启动命令
	for _, cmd := range commands {
		for !checkConditionA(commands) {
			time.Sleep(1 * time.Minute)
		}
		go func(cmd *Command) {
			ch, err := startCommand(cmd)
			if err != nil {
				fmt.Println("Error starting command:", err)
				return
			}
			for log := range ch {
				fmt.Printf("[process %d] %s\n", cmd.Line, log)
				cmd.Log = log
			}
		}(cmd)
		time.Sleep(1 * time.Second)
	}

	// 等待所有命令启动
	// wg.Wait()

	fmt.Println("=========进入主循环")
	// 主循环
	for {
		for !checkConditionA(commands) {
			time.Sleep(1 * time.Minute)
		}

		stoppedCmd := findStoppedCommand(commands)
		if stoppedCmd != nil {
			err := restartCommand(stoppedCmd)
			if err != nil {
				fmt.Println("Error restarting command:", err)
				return
			}
			time.Sleep(1 * time.Second)
			continue
		}

		latestCmd := findLatestTryAgain(commands, *tryDelay)
		if latestCmd == nil {
			time.Sleep(1 * time.Second)
			continue
		}

		_, latestTry, _ := parseLog(latestCmd.Log)
		earliestCmd := findEarliestLog(commands, latestTry)
		fmt.Println("latestCmd",latestCmd.Line)
		fmt.Println("earliestCmd",earliestCmd.Line)

		err := restartCommand(latestCmd)
		if err != nil {
			fmt.Println("Error restarting command:", err)
			return
		}
		time.Sleep(5 * time.Second)
	}
	fmt.Println("====================退出主循环")
}
