package ai

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	endpoint     string
	defaultModel string
	httpClient   *http.Client
}

func NewClient(endpoint, defaultModel string) *Client {
	return &Client{
		endpoint:     endpoint,
		defaultModel: defaultModel,
		httpClient: &http.Client{
			Timeout: 0, // No timeout for streaming
		},
	}
}

type GenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type GenerateResponse struct {
	Model     string `json:"model"`
	Response  string `json:"response"`
	Done      bool   `json:"done"`
	CreatedAt string `json:"created_at"`
}

type ModelInfo struct {
	Name       string `json:"name"`
	ModifiedAt string `json:"modified_at"`
	Size       int64  `json:"size"`
}

type ModelsResponse struct {
	Models []ModelInfo `json:"models"`
}

func (c *Client) GetStatus() (bool, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(c.endpoint + "/api/tags")
	if err != nil {
		return false, err
	}
	defer func() { _ = resp.Body.Close() }()
	return resp.StatusCode == http.StatusOK, nil
}

func (c *Client) ListModels() ([]ModelInfo, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(c.endpoint + "/api/tags")
	if err != nil {
		return nil, fmt.Errorf("connecting to ollama: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var models ModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&models); err != nil {
		return nil, fmt.Errorf("decoding models response: %w", err)
	}
	return models.Models, nil
}

func (c *Client) DefaultModel() string {
	return c.defaultModel
}

func (c *Client) GenerateStream(model, prompt string) (io.ReadCloser, error) {
	if model == "" {
		model = c.defaultModel
	}

	reqBody := GenerateRequest{
		Model:  model,
		Prompt: prompt,
		Stream: true,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	resp, err := c.httpClient.Post(c.endpoint+"/api/generate", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("calling ollama: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}

	return resp.Body, nil
}

// StreamTokens reads streaming responses and calls the callback for each token.
func StreamTokens(reader io.ReadCloser, onToken func(token string, done bool) error) error {
	defer func() { _ = reader.Close() }()
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		var resp GenerateResponse
		if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
			continue
		}
		if err := onToken(resp.Response, resp.Done); err != nil {
			return err
		}
		if resp.Done {
			return nil
		}
	}
	return scanner.Err()
}
