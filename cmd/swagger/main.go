package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/mithril-framework/mithril/internal/timezone"
)

const (
	openAIURL     = "https://api.openai.com/v1/chat/completions"
	defaultModel  = "gpt-4o-mini"
	routesDir     = "routes"
	swaggerPath   = "docs/swagger.json"
	systemPrompt  = "You are an OpenAPI 3.0 schema generator. This app is a Go Fiber app: routes live in routes/*.go and are registered via RegisterAll (e.g. SetupCrudBlogRoutes, SetupAuthRoutes). API routes are under app.Group(\"/api\") (e.g. /api/blogs, /api/users, /api/auth/...). CRUD uses GET list, GET :id, POST, PUT :id, DELETE :id. Given the route files and current OpenAPI schema (and any git diff), produce a single complete, valid OpenAPI 3.0 JSON document. Preserve existing paths and components unless the route files or diff indicate changes. Your response MUST be strict JSON only: a single JSON object, every key in double quotes (e.g. \"description\": not description:), no markdown code fences, no trailing commas, no comments, no explanation."
)

func main() {
	loadEnvFile(".env")
	if err := timezone.InitFromEnv(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "OPENAI_API_KEY is not set. Add it to .env for make swagger.")
		os.Exit(1)
	}

	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = defaultModel
	}

	routesContent, err := readRoutesContent()
	if err != nil {
		fmt.Fprintf(os.Stderr, "routes: %v\n", err)
		os.Exit(1)
	}

	swaggerContent, err := os.ReadFile(swaggerPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "swagger read: %v\n", err)
		os.Exit(1)
	}

	diffRoutes := runGitDiff(routesDir)
	diffSwagger := runGitDiff(swaggerPath)

	routeSummary := buildRouteSummary(routesContent)
	userParts := []string{}
	if routeSummary != "" {
		userParts = append(userParts, "=== Paths to document ===\n"+routeSummary+"\n\n")
	}
	userParts = append(userParts,
		"=== Route files (routes/*.go) ===\n"+routesContent,
		"\n=== Current docs/swagger.json ===\n"+string(swaggerContent),
	)
	if diffRoutes != "" {
		userParts = append(userParts, "\n=== Git diff of routes/ ===\n"+diffRoutes)
	}
	if diffSwagger != "" {
		userParts = append(userParts, "\n=== Git diff of docs/swagger.json ===\n"+diffSwagger)
	}
	userContent := strings.Join(userParts, "")

	body := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userContent},
		},
	}
	bodyJSON, _ := json.Marshal(body)

	req, err := http.NewRequest(http.MethodPost, openAIURL, bytes.NewReader(bodyJSON))
	if err != nil {
		fmt.Fprintf(os.Stderr, "request: %v\n", err)
		os.Exit(1)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "openai request: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read response: %v\n", err)
		os.Exit(1)
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "openai error %d: %s\n", resp.StatusCode, string(respBody))
		os.Exit(1)
	}

	var openAIResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &openAIResp); err != nil {
		fmt.Fprintf(os.Stderr, "parse openai response: %v\n", err)
		os.Exit(1)
	}
	if len(openAIResp.Choices) == 0 {
		fmt.Fprintln(os.Stderr, "openai returned no choices")
		os.Exit(1)
	}

	raw := strings.TrimSpace(openAIResp.Choices[0].Message.Content)
	raw = stripMarkdownCodeFence(raw)
	toParse := fixUnquotedKeys(raw)

	var doc map[string]any
	if err := json.Unmarshal([]byte(toParse), &doc); err != nil {
		snippet := toParse
		if syntaxErr, ok := err.(*json.SyntaxError); ok {
			offset := int(syntaxErr.Offset)
			if offset >= 0 && offset <= len(toParse) {
				const window = 80
				start := offset - window
				if start < 0 {
					start = 0
				}
				end := offset + window
				if end > len(toParse) {
					end = len(toParse)
				}
				snippet = toParse[start:end]
				if start > 0 {
					snippet = "..." + snippet
				}
				if end < len(toParse) {
					snippet = snippet + "..."
				}
				fmt.Fprintf(os.Stderr, "invalid json from model: %v (offset %d)\nsnippet around error: %s\n", err, offset, snippet)
			} else {
				fallback := toParse
				if len(fallback) > 200 {
					fallback = fallback[:200] + "..."
				}
				fmt.Fprintf(os.Stderr, "invalid json from model: %v\nraw snippet: %s\n", err, fallback)
			}
		} else {
			if len(snippet) > 200 {
				snippet = snippet[:200] + "..."
			}
			fmt.Fprintf(os.Stderr, "invalid json from model: %v\nraw snippet: %s\n", err, snippet)
		}
		os.Exit(1)
	}
	if _, ok := doc["openapi"]; !ok {
		fmt.Fprintln(os.Stderr, "model output missing openapi field")
		os.Exit(1)
	}
	if _, ok := doc["info"]; !ok {
		fmt.Fprintln(os.Stderr, "model output missing info field")
		os.Exit(1)
	}
	if _, ok := doc["paths"]; !ok {
		fmt.Fprintln(os.Stderr, "model output missing paths field")
		os.Exit(1)
	}

	indented, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal: %v\n", err)
		os.Exit(1)
	}

	dir := filepath.Dir(swaggerPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir: %v\n", err)
		os.Exit(1)
	}

	tmp, err := os.CreateTemp(dir, "swagger.*.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "temp file: %v\n", err)
		os.Exit(1)
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(indented); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		fmt.Fprintf(os.Stderr, "write temp: %v\n", err)
		os.Exit(1)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		fmt.Fprintf(os.Stderr, "close temp: %v\n", err)
		os.Exit(1)
	}
	if err := os.Rename(tmpPath, swaggerPath); err != nil {
		os.Remove(tmpPath)
		fmt.Fprintf(os.Stderr, "rename: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("docs/swagger.json updated.")
}

func loadEnvFile(filename string) {
	f, err := os.Open(filename)
	if err != nil {
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, "=")
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		if key == "" || os.Getenv(key) != "" {
			continue
		}
		value := strings.TrimSpace(line[idx+1:])
		value = strings.Trim(value, `"`)
		os.Setenv(key, value)
	}
}

func readRoutesContent() (string, error) {
	entries, err := os.ReadDir(routesDir)
	if err != nil {
		return "", err
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(routesDir, e.Name()))
		if err != nil {
			return "", err
		}
		out = append(out, "--- "+e.Name()+" ---\n"+string(b))
	}
	return strings.Join(out, "\n"), nil
}

func runGitDiff(path string) string {
	cmd := exec.Command("git", "diff", path)
	cmd.Dir = "."
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(out)
}

// pathMethodRe matches .Get("/path", .Post("/path", etc. Captures method and path.
var pathMethodRe = regexp.MustCompile(`\.(Get|Post|Put|Delete|Patch)\(\s*"([^"]+)"`)

func buildRouteSummary(routesContent string) string {
	useAPIBase := strings.Contains(routesContent, `Group("/api")`)
	matches := pathMethodRe.FindAllStringSubmatch(routesContent, -1)
	if len(matches) == 0 {
		return ""
	}
	seen := make(map[string]bool)
	var lines []string
	for _, m := range matches {
		if len(m) != 3 {
			continue
		}
		method := strings.ToUpper(m[1])
		path := m[2]
		if useAPIBase && !strings.HasPrefix(path, "/api") {
			path = "/api" + path
		}
		key := method + " " + path
		if seen[key] {
			continue
		}
		seen[key] = true
		lines = append(lines, method+" "+path)
	}
	if len(lines) == 0 {
		return ""
	}
	return "Paths to document: " + strings.Join(lines, ", ")
}

var markdownFence = regexp.MustCompile("(?s)^\\s*```(?:json)?\\s*\\n(.*?)\\n\\s*```\\s*$")

func stripMarkdownCodeFence(s string) string {
	s = strings.TrimSpace(s)
	if m := markdownFence.FindStringSubmatch(s); len(m) == 2 {
		s = strings.TrimSpace(m[1])
	} else {
		if strings.HasPrefix(s, "```json") {
			s = strings.TrimPrefix(s, "```json")
		} else if strings.HasPrefix(s, "```") {
			s = strings.TrimPrefix(s, "```")
		}
		s = strings.TrimSuffix(s, "```")
		s = strings.TrimSpace(s)
	}
	// Extract from first { to matching }; ignore trailing text or extra braces in prose
	if idx := strings.Index(s, "{"); idx >= 0 {
		s = s[idx:]
		s = extractJSONObject(s)
	}
	return s
}

// skipUnicodeSpace advances i past any runes where unicode.IsSpace is true, writing them to b. Returns the new index.
func skipUnicodeSpace(s string, i int, b *strings.Builder) int {
	for i < len(s) {
		r, size := utf8.DecodeRuneInString(s[i:])
		if size == 0 || !unicode.IsSpace(r) {
			return i
		}
		b.WriteString(s[i : i+size])
		i += size
	}
	return i
}

// fixUnquotedKeys returns a copy of s with unquoted object keys (e.g. description:) turned into quoted keys ("description":).
func fixUnquotedKeys(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 512)
	inString := false
	escape := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if escape {
			b.WriteByte(c)
			escape = false
			continue
		}
		if inString {
			b.WriteByte(c)
			if c == '\\' {
				escape = true
			} else if c == '"' {
				inString = false
			}
			continue
		}
		if c == '"' {
			inString = true
			b.WriteByte(c)
			continue
		}
		if c == ',' || c == '{' {
			b.WriteByte(c)
			i++
			i = skipUnicodeSpace(s, i, &b)
			if i >= len(s) {
				break
			}
			c = s[i]
			if c == '"' {
				i--
				continue
			}
			if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' {
				start := i
				for i < len(s) {
					cc := s[i]
					if (cc >= 'a' && cc <= 'z') || (cc >= 'A' && cc <= 'Z') || (cc >= '0' && cc <= '9') || cc == '_' {
						i++
					} else {
						break
					}
				}
				ident := s[start:i]
				i = skipUnicodeSpace(s, i, &b)
				if i < len(s) && s[i] == ':' {
					b.WriteByte('"')
					b.WriteString(ident)
					b.WriteByte('"')
					b.WriteByte(':')
					i++
				} else if i < len(s) && s[i] == '"' {
					b.WriteByte('"')
					b.WriteString(ident)
					b.WriteByte('"')
					b.WriteByte(':')
					i++
					i = skipUnicodeSpace(s, i, &b)
					if i < len(s) && s[i] == ':' {
						i++
					}
				} else {
					b.WriteString(ident)
				}
				i-- // for-loop will i++; we advanced i in this block so next char is at current i
				continue
			}
			i-- // re-process this character (e.g. digit, true/false/null)
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

// extractJSONObject returns the first complete {...} object by brace matching.
func extractJSONObject(s string) string {
	depth := 0
	inString := false
	escape := false
	quote := byte(0)
	for i := 0; i < len(s); i++ {
		c := s[i]
		if escape {
			escape = false
			continue
		}
		if inString {
			if c == '\\' && quote == '"' {
				escape = true
				continue
			}
			if c == quote {
				inString = false
			}
			continue
		}
		switch c {
		case '"':
			inString = true
			quote = c
		case '{', '[':
			depth++
		case '}', ']':
			depth--
			if depth == 0 {
				return s[:i+1]
			}
		}
	}
	return s
}
