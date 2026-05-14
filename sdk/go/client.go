package eyrie

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewClient(baseURL string, apiKey string) *Client {
	return &Client{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 120 * time.Second},
	}
}

type PromptRequest struct {
	Message      string `json:"message"`
	Model        string `json:"model,omitempty"`
	SystemPrompt string `json:"system_prompt,omitempty"`
	MaxTokens    int    `json:"max_tokens,omitempty"`
}

type PromptResponse struct {
	Content string `json:"content"`
	NodeID  string `json:"node_id"`
}

type Node struct {
	ID        string `json:"id"`
	ParentID  string `json:"parent_id,omitempty"`
	RootID    string `json:"root_id,omitempty"`
	Sequence  int    `json:"sequence"`
	NodeType  string `json:"node_type"`
	Content   string `json:"content"`
	Model     string `json:"model,omitempty"`
	Title     string `json:"title,omitempty"`
	CreatedAt string `json:"created_at"`
}

func (c *Client) Prompt(ctx context.Context, req PromptRequest) (*PromptResponse, error) {
	var resp PromptResponse
	err := c.post(ctx, "/prompt", req, &resp)
	return &resp, err
}

func (c *Client) PromptFrom(ctx context.Context, nodeID string, req PromptRequest) (*PromptResponse, error) {
	var resp PromptResponse
	err := c.post(ctx, "/nodes/"+nodeID+"/prompt", req, &resp)
	return &resp, err
}

func (c *Client) ListConversations(ctx context.Context) ([]Node, error) {
	var nodes []Node
	err := c.get(ctx, "/nodes", &nodes)
	return nodes, err
}

func (c *Client) GetNode(ctx context.Context, id string) (*Node, error) {
	var node Node
	err := c.get(ctx, "/nodes/"+id, &node)
	return &node, err
}

func (c *Client) GetTree(ctx context.Context, id string) ([]Node, error) {
	var nodes []Node
	err := c.get(ctx, "/nodes/"+id+"/tree", &nodes)
	return nodes, err
}

func (c *Client) DeleteNode(ctx context.Context, id string) error {
	return c.do(ctx, "DELETE", "/nodes/"+id, nil, nil)
}

func (c *Client) Health(ctx context.Context) error {
	return c.get(ctx, "/health", nil)
}

func (c *Client) get(ctx context.Context, path string, out interface{}) error {
	return c.do(ctx, "GET", path, nil, out)
}

func (c *Client) post(ctx context.Context, path string, body, out interface{}) error {
	return c.do(ctx, "POST", path, body, out)
}

func (c *Client) do(ctx context.Context, method, path string, body, out interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("eyrie: %s %s: %d %s", method, path, resp.StatusCode, string(b))
	}

	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}
