package ai

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sebrandon1/go-toolbox/httpx"
)

type Client struct {
	endpoint     string
	defaultModel string
	// streamClient deliberately has no request timeout: streamed generation
	// responses can last far longer than a fixed deadline.
	streamClient *httpx.Client
}

func NewClient(endpoint, defaultModel string) *Client {
	streamClient := httpx.NewClient(httpx.Options{BaseURL: endpoint})
	// NewClient installs DefaultTimeout; streaming must not be cut short.
	streamClient.HTTPClient.Timeout = 0

	return &Client{
		endpoint:     endpoint,
		defaultModel: defaultModel,
		streamClient: streamClient,
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
	client := httpx.NewClient(httpx.Options{BaseURL: c.endpoint, Timeout: 5 * time.Second})
	req, err := client.NewRequest(context.Background(), http.MethodGet, "/api/tags", nil)
	if err != nil {
		return false, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer func() { _ = resp.Body.Close() }()
	return resp.StatusCode == http.StatusOK, nil
}

func (c *Client) ListModels() ([]ModelInfo, error) {
	client := httpx.NewClient(httpx.Options{BaseURL: c.endpoint, Timeout: 10 * time.Second})

	var models ModelsResponse
	if err := client.DoJSON(context.Background(), http.MethodGet, "/api/tags", nil, &models); err != nil {
		if httpx.IsNetworkError(err) {
			return nil, fmt.Errorf("connecting to ollama: %w", err)
		}
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

	req, err := c.streamClient.NewRequest(context.Background(), http.MethodPost, "/api/generate", reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	resp, err := c.streamClient.Do(req)
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
