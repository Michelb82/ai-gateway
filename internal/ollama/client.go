package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

type chatRequest struct {
	Model              string          `json:"model"`
	Messages           []chatMessage   `json:"messages"`
	Temperature        float64         `json:"temperature"`
	Think              bool            `json:"think"`
	ChatTemplateKwargs map[string]bool `json:"chat_template_kwargs"`
	KeepAlive          string          `json:"keep_alive,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type tagsResponse struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
}

type pullRequest struct {
	Name   string `json:"name"`
	Stream bool   `json:"stream"`
}

type pullResponse struct {
	Status string `json:"status"`
	Error  string `json:"error"`
}

func (c *Client) Complete(ctx context.Context, systemPrompt, prompt, model, keepAlive string) (string, error) {
	systemPrompt = strings.TrimSpace(systemPrompt)
	prompt = strings.TrimSpace(prompt)
	model = strings.TrimSpace(model)
	keepAlive = strings.TrimSpace(keepAlive)
	if prompt == "" {
		return "", fmt.Errorf("prompt must not be empty")
	}
	if systemPrompt == "" {
		return "", fmt.Errorf("system prompt must not be empty")
	}
	if model == "" {
		return "", fmt.Errorf("model must not be empty")
	}

	body := chatRequest{
		Model: model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.1,
		Think:       false,
		ChatTemplateKwargs: map[string]bool{
			"enable_thinking": false,
		},
		KeepAlive: keepAlive,
	}

	encoded, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("encode chat request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/chat/completions", bytes.NewReader(encoded))
	if err != nil {
		return "", fmt.Errorf("create chat request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("chat request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read chat response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("chat request returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var decoded chatResponse
	if err := json.Unmarshal(respBody, &decoded); err != nil {
		return "", fmt.Errorf("decode chat response: %w", err)
	}
	if len(decoded.Choices) == 0 {
		return "", fmt.Errorf("chat response contained no choices")
	}

	content := strings.TrimSpace(decoded.Choices[0].Message.Content)
	if content == "" {
		return "", fmt.Errorf("chat response contained empty content")
	}

	return content, nil
}

func (c *Client) ListModels(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/tags", nil)
	if err != nil {
		return nil, fmt.Errorf("create tags request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama unreachable: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read tags response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("tags request returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var decoded tagsResponse
	if err := json.Unmarshal(respBody, &decoded); err != nil {
		return nil, fmt.Errorf("decode tags response: %w", err)
	}

	names := make([]string, 0, len(decoded.Models))
	for _, model := range decoded.Models {
		name := strings.TrimSpace(model.Name)
		if name != "" {
			names = append(names, name)
		}
	}
	return names, nil
}

func (c *Client) ModelAvailable(ctx context.Context, name string) (bool, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return false, fmt.Errorf("model name must not be empty")
	}

	models, err := c.ListModels(ctx)
	if err != nil {
		return false, err
	}

	for _, model := range models {
		if model == name {
			return true, nil
		}
	}
	return false, nil
}

// Pull downloads a model via Ollama's /api/pull (non-streaming).
func (c *Client) Pull(ctx context.Context, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("model name must not be empty")
	}

	encoded, err := json.Marshal(pullRequest{Name: name, Stream: false})
	if err != nil {
		return fmt.Errorf("encode pull request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/pull", bytes.NewReader(encoded))
	if err != nil {
		return fmt.Errorf("create pull request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Large pulls can exceed the default chat timeout; use a dedicated client.
	pullClient := &http.Client{Timeout: 0}
	resp, err := pullClient.Do(req)
	if err != nil {
		return fmt.Errorf("pull request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read pull response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("pull request returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var decoded pullResponse
	if err := json.Unmarshal(respBody, &decoded); err != nil {
		return fmt.Errorf("decode pull response: %w", err)
	}
	if decoded.Error != "" {
		return fmt.Errorf("pull failed: %s", decoded.Error)
	}
	status := strings.ToLower(strings.TrimSpace(decoded.Status))
	if status != "" && status != "success" {
		return fmt.Errorf("pull finished with status %q", decoded.Status)
	}
	return nil
}

// EnsureModels checks Ollama and pulls any configured models missing from /api/tags.
// Individual pull failures are skipped. It returns an error only when none of the
// configured models are available afterward (including when the endpoint is down).
func (c *Client) EnsureModels(ctx context.Context, names []string) (available []string, unavailable []string, err error) {
	unique := uniqueNonEmpty(names)
	if len(unique) == 0 {
		return nil, nil, fmt.Errorf("no models configured")
	}

	installed, err := c.ListModels(ctx)
	if err != nil {
		return nil, unique, fmt.Errorf("ollama health check failed: %w", err)
	}
	present := make(map[string]struct{}, len(installed))
	for _, model := range installed {
		present[model] = struct{}{}
	}

	for _, name := range unique {
		if _, ok := present[name]; ok {
			available = append(available, name)
			continue
		}
		if pullErr := c.Pull(ctx, name); pullErr != nil {
			unavailable = append(unavailable, name)
			continue
		}
		present[name] = struct{}{}
		available = append(available, name)
	}

	if len(available) == 0 {
		return nil, unavailable, fmt.Errorf("no configured models available")
	}
	return available, unavailable, nil
}

func uniqueNonEmpty(names []string) []string {
	seen := make(map[string]struct{}, len(names))
	out := make([]string, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out
}
