package client

import (
	"context"
	"strings"

	"github.com/GrayCodeAI/eyrie/catalog/opencodego"
)

// openCodeGoClient wraps OpenAIClient for OpenCode Go (/zen/go/v1/chat/completions).
type openCodeGoClient struct {
	inner *OpenAIClient
}

// NewOpenCodeGoClient builds an OpenAI-compatible client for OpenCode Go.
func NewOpenCodeGoClient(apiKey, baseURL string, opts ...ClientOption) *openCodeGoClient {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = opencodego.DefaultBaseURL
	}
	ocgOpts := append(append([]ClientOption{}, opts...), WithProviderName("opencodego"))
	return &openCodeGoClient{
		inner: NewOpenAIClient(apiKey, baseURL, &OpenCodeGoCompat, ocgOpts...),
	}
}

func (c *openCodeGoClient) Name() string { return c.inner.Name() }

func (c *openCodeGoClient) Chat(ctx context.Context, messages []EyrieMessage, opts ChatOptions) (*EyrieResponse, error) {
	opts.Model = opencodego.NativeModelID(opts.Model)
	return c.inner.Chat(ctx, messages, opts)
}

func (c *openCodeGoClient) StreamChat(ctx context.Context, messages []EyrieMessage, opts ChatOptions) (*StreamResult, error) {
	opts.Model = opencodego.NativeModelID(opts.Model)
	return c.inner.StreamChat(ctx, messages, opts)
}

func (c *openCodeGoClient) Ping(ctx context.Context) error {
	return c.inner.Ping(ctx)
}

var _ Provider = (*openCodeGoClient)(nil)
