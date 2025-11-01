package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultTimeout = 30 * time.Second

// Message represents a single chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Client handles HTTP communication with OpenAI-compatible APIs.
type Client struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

// NewClient creates a new API client.
func NewClient(apiKey, baseURL string) (*Client, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, errors.New("api key cannot be empty")
	}
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	return &Client{
		apiKey:  apiKey,
		baseURL: strings.TrimSuffix(baseURL, "/"),
		http: &http.Client{
			Timeout: defaultTimeout,
		},
	}, nil
}

// Chat sends a chat completion request and returns the assistant's response.
func (c *Client) Chat(ctx context.Context, messages []Message, model string, temperature float64) (string, error) {
	if c == nil {
		return "", errors.New("client is nil")
	}

	reqBody := map[string]interface{}{
		"model":       model,
		"messages":    messages,
		"temperature": temperature,
		"stream":      false,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", c.decodeError(resp.Body, resp.StatusCode)
	}

	return c.decodeSuccess(resp.Body)
}

func (c *Client) decodeSuccess(r io.Reader) (string, error) {
	var response struct {
		Choices []struct {
			Message Message `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(r).Decode(&response); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if len(response.Choices) == 0 {
		return "", errors.New("no choices in response")
	}

	return response.Choices[0].Message.Content, nil
}

func (c *Client) decodeError(r io.Reader, status int) error {
	var apiErr struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.NewDecoder(r).Decode(&apiErr); err != nil {
		return fmt.Errorf("api error (status %d): failed to decode body: %w", status, err)
	}

	if apiErr.Error.Message != "" {
		return fmt.Errorf("api error (status %d): %s", status, apiErr.Error.Message)
	}

	return fmt.Errorf("api error (status %d)", status)
}
