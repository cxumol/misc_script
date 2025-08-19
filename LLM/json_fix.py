import json,re

def repair_json(s: str) -> str:
    """
    Repairs a broken or malformed JSON string into a valid, compact version.
    Inspired by <https://github.com/RealAlexandreAI/json-repair/blob/main/jsonrepair.go>
    A combination of regexp and manual parsing to achieve high performance and robustness.

    Repair capabilities:
    - Removes Markdown code blocks (```json) and whitespace.
    - Converts Python/JavaScript literals (True, False, None) to JSON (true, false, null).
    - Adds double quotes to unquoted object keys.
    - Replaces single quotes with double quotes.
    - Handles trailing commas in objects and arrays.
    - Attempts to close unclosed objects and arrays.
    """
    
    # 1. Initial cleanup: remove whitespace, code blocks, and normalize literals
    s = s.strip()
    if s.startswith("```json"):
        s = s[7:]
    if s.endswith("```"):
        s = s[:-3]
    s = s.strip()

    # Normalize literals (None -> null, True -> true, False -> false)
    s = re.sub(r'\bNone\b', 'null', s)
    s = re.sub(r'\bTrue\b', 'true', s)
    s = re.sub(r'\bFalse\b', 'false', s)
    # Replace single-quoted strings with double-quoted ones
    s = re.sub(r"(?<!\\)'(.*?)(?<!\\)'", r'"\1"', s)
    # Add quotes to unquoted object keys
    # (an identifier followed by optional whitespace and a ':')
    s = re.sub(r'([_a-zA-Z][_a-zA-Z0-9]*)\s*:', r'"\1":', s)

    # 2. Fast path: If JSON is already valid, format and return
    try:
        return json.dumps(json.loads(s), separators=(',', ':'))
    except json.JSONDecodeError:
        pass # If not valid, proceed to manual repair

    # 3. Manual parsing and reconstruction
    res = []
    i = 0
    n = len(s)
    stack = []  # Stack to track whether we're in an object or array

    while i < n:
        char = s[i]

        if char.isspace():
            i += 1
            continue

        if char == '{':
            stack.append('}')
            res.append(char)
        elif char == '[':
            stack.append(']')
            res.append(char)
        elif char in '}]' and stack and char == stack[-1]:
            # Remove trailing comma if it exists
            if res and res[-1] == ',':
                res.pop()
            res.append(char)
            stack.pop()
        elif char == '"':  # Parse a string
            start = i
            i += 1
            while i < n:
                if s[i] == '"' and s[i-1] != '\\':
                    break
                i += 1
            res.append(s[start:i+1])
        elif char.isdigit() or char == '-':  # Parse a number
            match = re.match(r'-?\d+(\.\d+)?([eE][+-]?\d+)?', s[i:])
            if match:
                res.append(match.group(0))
                i += len(match.group(0)) - 1
        elif s[i:i+4] in ('true', 'null'):
            res.append(s[i:i+4])
            i += 3
        elif s[i:i+5] == 'false':
            res.append(s[i:i+5])
            i += 4
        elif char in ':,':
            # Avoid duplicate or misplaced commas
            if res and res[-1] not in '[{,':
                res.append(char)
        
        i += 1

    # 4. Close any unclosed structures
    if res and res[-1] == ',':
        res.pop()
    while stack:
        res.append(stack.pop())

    repaired_s = "".join(res)

    # 5. Final attempt to validate and format
    try:
        # Use loads and dumps for a final cleanup and format pass
        return json.dumps(json.loads(repaired_s), separators=(',', ':'))
    except json.JSONDecodeError:
        # If all else fails, return the best-effort repaired string
        return repaired_s
