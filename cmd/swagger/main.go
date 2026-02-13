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
)

const (
	openAIURL     = "https://api.openai.com/v1/chat/completions"
	defaultModel  = "gpt-4o-mini"
	routesDir     = "routes"
	swaggerPath   = "docs/swagger.json"
	systemPrompt  = "You are an OpenAPI 3.0 schema generator. Given Go Fiber route files and the current OpenAPI schema (and any git diff), produce a single complete, valid OpenAPI 3.0 JSON document. Preserve existing paths and components unless the route files or diff indicate changes. Output only the raw JSON document, no markdown code fences, no explanation, no commentary."
)

func main() {
	loadEnvFile(".env")

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

	userParts := []string{
		"=== Route files (routes/*.go) ===\n" + routesContent,
		"\n=== Current docs/swagger.json ===\n" + string(swaggerContent),
	}
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

	var doc map[string]any
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		fmt.Fprintf(os.Stderr, "invalid json from model: %v\n", err)
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

var markdownFence = regexp.MustCompile("(?s)^\\s*```(?:json)?\\s*\\n(.*?)\\n\\s*```\\s*$")

func stripMarkdownCodeFence(s string) string {
	if m := markdownFence.FindStringSubmatch(s); len(m) == 2 {
		return strings.TrimSpace(m[1])
	}
	// Try stripping a leading ```json or ``` and trailing ```
	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
	} else if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
	}
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}
