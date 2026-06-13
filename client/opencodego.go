package client

import (
	"context"
	"strings"

	"github.com/GrayCodeAI/eyrie/catalog/opencodego"
)

// OpenCodeGoClient routes OpenCode Go models to /v1/chat/completions or /v1/messages
// per opencode.ai/docs/go, with cross-protocol fallback when the primary path
// returns no answer text (e.g. reasoning-only MiniMax streams).
type OpenCodeGoClient struct {
	pair DualProtocolPair
}

// NewOpenCodeGoClient builds an OpenCode Go provider client.
func NewOpenCodeGoClient(apiKey, baseURL string, opts ...ClientOption) *OpenCodeGoClient {
	openBase := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if openBase == "" {
		openBase = opencodego.DefaultBaseURL
	}
	ocgOpts := append(append([]ClientOption{}, opts...), WithProviderName("opencodego"))
	o := NewOpenAIClient(apiKey, openBase, &OpenCodeGoCompat, ocgOpts...)
	a := NewAnthropicClient(apiKey, AnthropicBaseFromOpenAIV1(openBase), ocgOpts...)
	return &OpenCodeGoClient{pair: DualProtocolPair{OpenAI: o, Anthropic: a}}
}

func (c *OpenCodeGoClient) Name() string { return "opencodego" }

func (c *OpenCodeGoClient) Chat(ctx context.Context, messages []EyrieMessage, opts ChatOptions) (*EyrieResponse, error) {
	opts.Model = opencodego.NativeModelID(opts.Model)
	if opencodego.UsesMessagesAPI(opts.Model) {
		return c.pair.Chat(ctx, messages, opts, ChatProtocolMessages, openCodeGoMessagesFallback)
	}
	return c.pair.Chat(ctx, messages, opts, ChatProtocolCompletions, nil)
}

func (c *OpenCodeGoClient) StreamChat(ctx context.Context, messages []EyrieMessage, opts ChatOptions) (*StreamResult, error) {
	opts.Model = opencodego.NativeModelID(opts.Model)
	if opencodego.UsesMessagesAPI(opts.Model) {
		return c.pair.StreamChat(ctx, messages, opts, DualProtocolStreamConfig{
			Primary:               ChatProtocolMessages,
			ReasoningOnlyFallback: true,
		})
	}
	return c.pair.StreamChat(ctx, messages, opts, DualProtocolStreamConfig{
		Primary: ChatProtocolCompletions,
	})
}

func (c *OpenCodeGoClient) Ping(ctx context.Context) error {
	if err := c.pair.OpenAI.Ping(ctx); err == nil {
		return nil
	}
	return c.pair.Anthropic.Ping(ctx)
}

func openCodeGoMessagesFallback(primaryErr error, primaryResp *EyrieResponse) bool {
	if primaryErr != nil {
		return OACompatUnsupportedError(primaryErr)
	}
	return !ResponseHasContent(primaryResp)
}

var _ Provider = (*OpenCodeGoClient)(nil)
