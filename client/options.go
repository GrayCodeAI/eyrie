package client

import (
	"log/slog"
	"net/http"
	"time"
)

const defaultTimeout = 10 * time.Minute

// ResponseFormat specifies the desired output format for the model response.
type ResponseFormat struct {
	Type   string `json:"type"`             // "json_object" or "json_schema"
	Schema string `json:"schema,omitempty"` // optional JSON schema for structured output
}

// ChatOptions holds options for a chat request.
type ChatOptions struct {
	Provider       string          `json:"provider,omitempty"`
	Model          string          `json:"model,omitempty"`
	Temperature    *float64        `json:"temperature,omitempty"`
	MaxTokens      int             `json:"max_tokens,omitempty"`
	Stream         bool            `json:"stream,omitempty"`
	Tools          []EyrieTool     `json:"tools,omitempty"`
	System         string          `json:"system,omitempty"`
	EnableCaching  bool            `json:"enable_caching,omitempty"`
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
}

// ClientOption configures clients.
type ClientOption struct {
	applyFn       func(*AnthropicClient)
	applyOpenAIFn func(*OpenAIClient)
}

func (o ClientOption) apply(c *AnthropicClient)    { if o.applyFn != nil { o.applyFn(c) } }
func (o ClientOption) applyOpenAI(c *OpenAIClient)  { if o.applyOpenAIFn != nil { o.applyOpenAIFn(c) } }

// WithTimeout sets the HTTP client timeout.
func WithTimeout(d time.Duration) ClientOption {
	return ClientOption{
		applyFn:       func(c *AnthropicClient) { c.httpClient.Timeout = d },
		applyOpenAIFn: func(c *OpenAIClient) { c.httpClient.Timeout = d },
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(hc *http.Client) ClientOption {
	return ClientOption{
		applyFn:       func(c *AnthropicClient) { c.httpClient = hc },
		applyOpenAIFn: func(c *OpenAIClient) { c.httpClient = hc },
	}
}

// WithRetry sets retry configuration.
func WithRetry(rc RetryConfig) ClientOption {
	return ClientOption{
		applyFn:       func(c *AnthropicClient) { c.retry = rc },
		applyOpenAIFn: func(c *OpenAIClient) { c.retry = rc },
	}
}

// WithLogger sets the logger.
func WithLogger(l *slog.Logger) ClientOption {
	return ClientOption{
		applyFn:       func(c *AnthropicClient) { c.logger = l },
		applyOpenAIFn: func(c *OpenAIClient) { c.logger = l },
	}
}
