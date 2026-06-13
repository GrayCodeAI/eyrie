package client

import (
	"context"
	"strings"
)

// OpenCodeGoClient routes OpenCode Go models to the correct upstream API format.
// Most models use OpenAI-compatible /v1/chat/completions; MiniMax and Qwen3.x
// models require Anthropic-compatible /v1/messages (see opencode.ai/docs/go).
type OpenCodeGoClient struct {
	openAI    *OpenAIClient
	anthropic *AnthropicClient
}

type openCodeGoStreamOpener func(context.Context, []EyrieMessage, ChatOptions) (*StreamResult, error)

// NewOpenCodeGoClient builds a dual-protocol OpenCode Go provider client.
func NewOpenCodeGoClient(apiKey, baseURL string, opts ...ClientOption) *OpenCodeGoClient {
	openBase := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if openBase == "" {
		openBase = "https://opencode.ai/zen/go/v1"
	}
	ocgOpts := append(append([]ClientOption{}, opts...), WithProviderName("opencodego"))
	o := NewOpenAIClient(apiKey, openBase, &OpenCodeGoCompat, ocgOpts...)
	// Anthropic /messages on OpenCode Go uses X-Api-Key (not Bearer, not MiMo api-key).
	a := NewAnthropicClient(apiKey, openCodeGoAnthropicBase(openBase), ocgOpts...)
	return &OpenCodeGoClient{openAI: o, anthropic: a}
}

func (c *OpenCodeGoClient) Name() string { return "opencodego" }

func (c *OpenCodeGoClient) Chat(ctx context.Context, messages []EyrieMessage, opts ChatOptions) (*EyrieResponse, error) {
	if openCodeGoUsesAnthropicMessages(opts.Model) {
		resp, err := c.anthropic.Chat(ctx, messages, opts)
		if err != nil {
			return nil, err
		}
		if resp != nil && strings.TrimSpace(resp.Content) != "" {
			return resp, nil
		}
		if c.openAI == nil {
			return resp, err
		}
		openResp, openErr := c.openAI.Chat(ctx, messages, opts)
		if openErr == nil && openResp != nil && strings.TrimSpace(openResp.Content) != "" {
			return openResp, nil
		}
		return resp, err
	}
	resp, err := c.openAI.Chat(ctx, messages, opts)
	if err != nil || c.anthropic == nil || !openCodeGoMightNeedAnthropicFallback(opts.Model) {
		return resp, err
	}
	if resp != nil && strings.TrimSpace(resp.Content) != "" {
		return resp, nil
	}
	anthropicResp, anthropicErr := c.anthropic.Chat(ctx, messages, opts)
	if anthropicErr == nil && anthropicResp != nil && strings.TrimSpace(anthropicResp.Content) != "" {
		return anthropicResp, nil
	}
	return resp, err
}

func (c *OpenCodeGoClient) StreamChat(ctx context.Context, messages []EyrieMessage, opts ChatOptions) (*StreamResult, error) {
	if openCodeGoUsesAnthropicMessages(opts.Model) {
		primary, err := c.anthropic.StreamChat(ctx, messages, opts)
		if err != nil || c.openAI == nil {
			return primary, err
		}
		return newOpenCodeGoStreamWithFallback(ctx, messages, opts, primary, c.openAI.StreamChat), nil
	}
	primary, err := c.openAI.StreamChat(ctx, messages, opts)
	if err != nil || c.anthropic == nil || !openCodeGoMightNeedAnthropicFallback(opts.Model) {
		return primary, err
	}
	return newOpenCodeGoStreamWithFallback(ctx, messages, opts, primary, c.anthropic.StreamChat), nil
}

func (c *OpenCodeGoClient) Ping(ctx context.Context) error {
	if err := c.openAI.Ping(ctx); err == nil {
		return nil
	}
	return c.anthropic.Ping(ctx)
}

// newOpenCodeGoStreamWithFallback watches a primary stream; if it ends with
// reasoning tokens but no answer, transparently retries via the fallback opener.
// Used for OpenAI→Anthropic and Anthropic→OpenAI on MiniMax/Qwen models.
func newOpenCodeGoStreamWithFallback(ctx context.Context, messages []EyrieMessage, opts ChatOptions, primary *StreamResult, fallback openCodeGoStreamOpener) *StreamResult {
	out := make(chan EyrieStreamEvent, streamChannelBuffer)
	cancelCtx, cancel := context.WithCancel(ctx)
	go func() {
		defer close(out)
		defer primary.Close()
		defer cancel()

		var (
			sawReasoning bool
			contentLen   int
			toolCalls    int
			buffered     []EyrieStreamEvent
			streamErr    bool
		)
		flush := func() {
			for _, ev := range buffered {
				select {
				case out <- ev:
				case <-cancelCtx.Done():
					return
				}
			}
			buffered = nil
		}
		for ev := range primary.Events {
			switch ev.Type {
			case "thinking":
				sawReasoning = true
			case "content":
				contentLen += len(ev.Content)
			case "tool_call":
				toolCalls++
			case "error":
				streamErr = true
			}
			if contentLen > 0 || toolCalls > 0 {
				flush()
				select {
				case out <- ev:
				case <-cancelCtx.Done():
					return
				}
				continue
			}
			buffered = append(buffered, ev)
		}

		health := DetectResponseHealth(ResponseSignals{
			SawReasoning: sawReasoning,
			ContentLen:   contentLen,
			ToolCalls:    toolCalls,
			StreamEnded:  true,
			StreamErr:    streamErr,
		})
		if health != ResponseErrorOnlyReasoning {
			flush()
			return
		}

		fallbackResult, err := fallback(cancelCtx, messages, opts)
		if err != nil {
			flush()
			select {
			case out <- EyrieStreamEvent{Type: "error", Error: err.Error()}:
			case <-cancelCtx.Done():
			}
			return
		}
		defer fallbackResult.Close()
		for ev := range fallbackResult.Events {
			select {
			case out <- ev:
			case <-cancelCtx.Done():
				return
			}
		}
	}()
	return NewStreamResult(out, cancel)
}

// openCodeGoAnthropicBase strips the /v1 suffix so AnthropicClient posts to
// {base}/v1/messages (e.g. https://opencode.ai/zen/go/v1/messages).
func openCodeGoAnthropicBase(openAIBase string) string {
	base := strings.TrimRight(strings.TrimSpace(openAIBase), "/")
	if strings.HasSuffix(base, "/v1") {
		return strings.TrimSuffix(base, "/v1")
	}
	return base
}

// openCodeGoUsesAnthropicMessages reports whether a model must use the Anthropic
// Messages API on OpenCode Go (not OpenAI chat/completions).
func openCodeGoUsesAnthropicMessages(model string) bool {
	model = openCodeGoNativeModel(model)
	switch {
	case strings.Contains(model, "minimax"):
		return true
	case strings.HasPrefix(model, "qwen3."):
		return true
	default:
		return false
	}
}

func openCodeGoMightNeedAnthropicFallback(model string) bool {
	return openCodeGoUsesAnthropicMessages(model)
}

func openCodeGoNativeModel(model string) string {
	model = strings.ToLower(strings.TrimSpace(model))
	if i := strings.LastIndex(model, "/"); i >= 0 {
		model = model[i+1:]
	}
	return model
}

var _ Provider = (*OpenCodeGoClient)(nil)
