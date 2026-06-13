package client

import (
	"context"
	"strings"
)

// ChatProtocol selects which HTTP API a dual-protocol gateway exposes.
type ChatProtocol int

const (
	// ChatProtocolCompletions uses OpenAI-style POST /v1/chat/completions.
	ChatProtocolCompletions ChatProtocol = iota
	// ChatProtocolMessages uses Anthropic-style POST /v1/messages.
	ChatProtocolMessages
)

type streamOpener func(context.Context, []EyrieMessage, ChatOptions) (*StreamResult, error)

// DualProtocolChatFallback decides whether to retry via the alternate protocol
// after the primary attempt. resp is non-nil only when primary returned err=nil.
type DualProtocolChatFallback func(primaryErr error, primaryResp *EyrieResponse) bool

// DualProtocolPair holds OpenAI and Anthropic clients that share credentials
// against the same gateway (OpenCode Go, MiMo, etc.).
type DualProtocolPair struct {
	OpenAI    *OpenAIClient
	Anthropic *AnthropicClient
}

// DualProtocolStreamConfig controls streaming across two protocols.
type DualProtocolStreamConfig struct {
	Primary               ChatProtocol
	FallbackOnError       func(error) bool
	ReasoningOnlyFallback bool // retry alternate protocol when primary stream is reasoning-only
}

// Chat sends a request via the primary protocol, optionally falling back to the
// alternate protocol when fallback returns true.
func (p DualProtocolPair) Chat(ctx context.Context, messages []EyrieMessage, opts ChatOptions, primary ChatProtocol, fallback DualProtocolChatFallback) (*EyrieResponse, error) {
	primaryClient, fallbackClient := p.providers(primary)
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
func (p DualProtocolPair) StreamChat(ctx context.Context, messages []EyrieMessage, opts ChatOptions, cfg DualProtocolStreamConfig) (*StreamResult, error) {
	primaryClient, fallbackClient := p.providers(cfg.Primary)
	result, err := primaryClient.StreamChat(ctx, messages, opts)
	if err != nil {
		if cfg.FallbackOnError != nil && fallbackClient != nil && cfg.FallbackOnError(err) {
			return fallbackClient.StreamChat(ctx, messages, opts)
		}
		return result, err
	}
	if cfg.ReasoningOnlyFallback && cfg.Primary == ChatProtocolMessages && fallbackClient != nil {
		var fallbackStream streamOpener
		if openAI, ok := fallbackClient.(*OpenAIClient); ok {
			fallbackStream = openAI.StreamChat
		}
		if fallbackStream != nil {
			return newStreamWithReasoningFallback(ctx, messages, opts, result, fallbackStream), nil
		}
	}
	return result, nil
}

func (p DualProtocolPair) providers(primary ChatProtocol) (Provider, Provider) {
	if primary == ChatProtocolMessages {
		return p.Anthropic, p.OpenAI
	}
	return p.OpenAI, p.Anthropic
}

// AnthropicBaseFromOpenAIV1 strips a trailing /v1 from an OpenAI-compatible base URL.
func AnthropicBaseFromOpenAIV1(openAIBase string) string {
	base := strings.TrimRight(strings.TrimSpace(openAIBase), "/")
	if strings.HasSuffix(base, "/v1") {
		return strings.TrimSuffix(base, "/v1")
	}
	return base
}

// OACompatUnsupportedError reports OpenCode Go errors where /v1/messages should
// fall back to /v1/chat/completions (e.g. qwen3.7-max oa-compat not supported).
func OACompatUnsupportedError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "status=401") ||
		strings.Contains(msg, "http 401") ||
		strings.Contains(msg, "oa-compat") ||
		strings.Contains(msg, "not supported")
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
