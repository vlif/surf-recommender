package anthropic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const (
	apiURL  = "https://api.anthropic.com/v1/messages"
	version = "2023-06-01"

	DefaultModel     = "claude-sonnet-4-6"
	DefaultMaxTokens = 2048

	maxRetries = 3
	retryDelay = 10 * time.Second
)

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type request struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	System    string    `json:"system"`
	Messages  []message `json:"messages"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type response struct {
	Content []contentBlock `json:"content"`
	Error   *apiError      `json:"error,omitempty"`
}

type apiError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type Client struct {
	apiKey     string
	model      string
	maxTokens  int
	httpClient *http.Client
}

func NewClient(apiKey, model string) *Client {
	if model == "" {
		model = DefaultModel
	}
	return &Client{
		apiKey:     apiKey,
		model:      model,
		maxTokens:  DefaultMaxTokens,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

// Send отправляет сообщение в Anthropic Messages API и возвращает текст ответа.
// При ошибках 529 (overloaded) и 529 делает до maxRetries повторных попыток с паузой.
func (c *Client) Send(systemPrompt, userMessage string) (string, error) {
	body, err := json.Marshal(request{
		Model:     c.model,
		MaxTokens: c.maxTokens,
		System:    systemPrompt,
		Messages:  []message{{Role: "user", Content: userMessage}},
	})
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	var lastErr error
	for attempt := range maxRetries {
		if attempt > 0 {
			log.Printf("anthropic: попытка %d/%d через %v...", attempt+1, maxRetries, retryDelay)
			time.Sleep(retryDelay)
		}

		result, retry, err := c.do(body)
		if err == nil {
			return result, nil
		}
		lastErr = err
		if !retry {
			return "", err
		}
	}

	return "", fmt.Errorf("anthropic: все %d попытки исчерпаны: %w", maxRetries, lastErr)
}

// do выполняет один HTTP-запрос. Возвращает (результат, нужен_retry, ошибка).
func (c *Client) do(body []byte) (string, bool, error) {
	req, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return "", false, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", version)
	req.Header.Set("content-type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", true, fmt.Errorf("anthropic request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", false, fmt.Errorf("read response: %w", err)
	}

	// 529 — перегрузка, 500/502/503 — временные серверные ошибки: retry
	if resp.StatusCode == 529 || resp.StatusCode >= 500 {
		return "", true, fmt.Errorf("anthropic API %d: %s", resp.StatusCode, respBody)
	}

	if resp.StatusCode != http.StatusOK {
		return "", false, fmt.Errorf("anthropic API %d: %s", resp.StatusCode, respBody)
	}

	var result response
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", false, fmt.Errorf("decode response: %w", err)
	}

	if result.Error != nil {
		return "", false, fmt.Errorf("anthropic error [%s]: %s", result.Error.Type, result.Error.Message)
	}

	if len(result.Content) == 0 {
		return "", false, fmt.Errorf("empty response from anthropic")
	}

	return result.Content[0].Text, false, nil
}
