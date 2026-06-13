package client

import (
	"context"
	"strings"
)

// ChatProtocol selects which existing eyrie client handles a gateway request.
type ChatProtocol int

const (
	// ChatProtocolCompletions routes via OpenAIClient (POST /v1/chat/completions).
	ChatProtocolCompletions ChatProtocol = iota
	// ChatProtocolMessages routes via AnthropicClient (POST /v1/messages).
	ChatProtocolMessages
)

type streamOpener func(context.Context, []EyrieMessage, ChatOptions) (*StreamResult, error)

// ProtocolChatFallback decides whether to retry via the alternate protocol
// after the primary attempt. resp is non-nil only when primary returned err=nil.
type ProtocolChatFallback func(primaryErr error, primaryResp *EyrieResponse) bool

// ProtocolRouter picks between OpenAIClient and AnthropicClient for gateways
// that expose both APIs (OpenCode Go, MiMo). All HTTP work stays in openai.go
// and anthropic.go; this type only orchestrates routing and fallback.
type ProtocolRouter struct {
	OpenAI    *OpenAIClient
	Anthropic *AnthropicClient
}

// ProtocolStreamConfig controls streaming across two protocols.
type ProtocolStreamConfig struct {
	Primary               ChatProtocol
	FallbackOnError       func(error) bool
	ReasoningOnlyFallback bool // retry alternate protocol when primary stream is reasoning-only
}

// Chat sends a request via the primary protocol, optionally falling back to the
// alternate protocol when fallback returns true.
func (r ProtocolRouter) Chat(ctx context.Context, messages []EyrieMessage, opts ChatOptions, primary ChatProtocol, fallback ProtocolChatFallback) (*EyrieResponse, error) {
	primaryClient, fallbackClient := r.providers(primary)
	resp, err := primaryClient.Chat(ctx, messages, opts)
	if fallback == nil || fallbackClient == nil || !fallback(err, resp) {
		return resp, err
	}
	fallbackResp, fallbackErr := fallbackClient.Chat(ctx, messages, opts)
	if fallbackErr == nil && ResponseHasContent(fallbackResp) {
		return fallbackResp, nil
	}
	return resp, err
}

// StreamChat streams via the primary protocol. When FallbackOnError matches,
// the alternate protocol is tried immediately. When ReasoningOnlyFallback is set
// and the primary is /v1/messages, an empty reasoning-only stream retries via
// chat/completions.
func (r ProtocolRouter) StreamChat(ctx context.Context, messages []EyrieMessage, opts ChatOptions, cfg ProtocolStreamConfig) (*StreamResult, error) {
	primaryClient, fallbackClient := r.providers(cfg.Primary)
	result, err := primaryClient.StreamChat(ctx, messages, opts)
	if err != nil {
		if cfg.FallbackOnError != nil && fallbackClient != nil && cfg.FallbackOnError(err) {
			return fallbackClient.StreamChat(ctx, messages, opts)
		}
		return result, err
	}
	if cfg.ReasoningOnlyFallback && cfg.Primary == ChatProtocolMessages && fallbackClient != nil {
		fallback := fallbackClient
		return newStreamWithReasoningFallback(ctx, messages, opts, result, func(ctx context.Context, messages []EyrieMessage, opts ChatOptions) (*StreamResult, error) {
			return fallback.StreamChat(ctx, messages, opts)
		}), nil
	}
	return result, nil
}

func (r ProtocolRouter) providers(primary ChatProtocol) (Provider, Provider) {
	if primary == ChatProtocolMessages {
		return r.Anthropic, r.OpenAI
	}
	return r.OpenAI, r.Anthropic
}

// AnthropicBaseFromOpenAIV1 strips a trailing /v1 from an OpenAI-compatible base URL.
func AnthropicBaseFromOpenAIV1(openAIBase string) string {
	base := strings.TrimRight(strings.TrimSpace(openAIBase), "/")
	if strings.HasSuffix(base, "/v1") {
		return strings.TrimSuffix(base, "/v1")
	}
	return base
}

func newStreamWithReasoningFallback(ctx context.Context, messages []EyrieMessage, opts ChatOptions, primary *StreamResult, fallback streamOpener) *StreamResult {
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
