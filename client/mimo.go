package client

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/GrayCodeAI/eyrie/catalog/xiaomi"
	"github.com/GrayCodeAI/eyrie/types"
)

// MiMoClient uses OpenAI-compatible MiMo endpoints first, with optional Anthropic-compat fallback.
type MiMoClient struct {
	router     ProtocolRouter
	providerID string
	logger     *slog.Logger
}

// NewMiMoClient builds a MiMo provider client (payg or token_plan gateway).
func NewMiMoClient(apiKey, openAIBase, anthropicBase string, compat *OpenAICompatConfig, providerID string, opts ...ClientOption) *MiMoClient {
	openAIBase = strings.TrimRight(strings.TrimSpace(openAIBase), "/")
	anthropicBase = strings.TrimRight(strings.TrimSpace(anthropicBase), "/")
	mimoOpts := append(append([]ClientOption{}, opts...), WithMimoAuth(), WithProviderName(providerID))
	o := NewOpenAIClient(apiKey, openAIBase, compat, mimoOpts...)
	var a *AnthropicClient
	if anthropicBase != "" {
		a = NewAnthropicClient(apiKey, anthropicBase, mimoOpts...)
	}
	return &MiMoClient{
		router:     ProtocolRouter{OpenAI: o, Anthropic: a},
		providerID: providerID,
		logger:     slog.Default(),
	}
}

// WithProviderName and WithMimoAuth live in client/core (options.go wraps them).

func (c *MiMoClient) Name() string {
	if c.router.OpenAI != nil {
		return c.router.OpenAI.Name()
	}
	return c.providerID
}

func (c *MiMoClient) Chat(ctx context.Context, messages []EyrieMessage, opts ChatOptions) (*EyrieResponse, error) {
	return c.router.Chat(ctx, messages, opts, ChatProtocolCompletions, func(err error, _ *EyrieResponse) bool {
		if err != nil && c.router.Anthropic != nil && mimoFallbackChatError(err) {
			c.logger.Info("MiMo: OpenAI endpoint failed; retrying via Anthropic compatibility", "provider", c.providerID, "error", err)
			return true
		}
		return false
	})
}

func (c *MiMoClient) StreamChat(ctx context.Context, messages []EyrieMessage, opts ChatOptions) (*StreamResult, error) {
	return c.router.StreamChat(ctx, messages, opts, ProtocolStreamConfig{
		Primary: ChatProtocolCompletions,
		FallbackOnError: func(err error) bool {
			if c.router.Anthropic != nil && mimoFallbackChatError(err) {
				c.logger.Info("MiMo: OpenAI stream failed; retrying via Anthropic compatibility", "provider", c.providerID, "error", err)
				return true
			}
			return false
		},
	})
}

func (c *MiMoClient) Ping(ctx context.Context) error {
	if err := c.router.OpenAI.Ping(ctx); err == nil {
		return nil
	} else if c.router.Anthropic == nil || !mimoRetryableChatError(err) {
		return err
	}
	return c.router.Anthropic.Ping(ctx)
}

func mimoFallbackChatError(err error) bool {
	if mimoRetryableChatError(err) {
		return true
	}
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "param incorrect") ||
		strings.Contains(msg, "invalid format") ||
		strings.Contains(msg, "reasoning_content") ||
		(strings.Contains(msg, "http 400") && strings.Contains(msg, "xiaomi"))
}

func mimoRetryableChatError(err error) bool {
	if err == nil {
		return false
	}
	// MiMo-specific: check xiaomi helper first (401/403 are retryable for MiMo)
	msg := err.Error()
	if n := parseHTTPStatusFromError(msg); n > 0 {
		if xiaomi.IsRetryableHTTPStatus(n) {
			return true
		}
	}
	// Structured path: trust EyrieError's IsRetriable
	var eyrieErr *EyrieError
	if errors.As(err, &eyrieErr) {
		return eyrieErr.IsRetriable()
	}
	// Conservative: only retry on explicitly transient errors (not the optimistic "unknown → true")
	return types.IsTransient(err)
}

func parseHTTPStatusFromError(msg string) int {
	for _, prefix := range []string{"HTTP ", "status ", "error ("} {
		if i := strings.Index(msg, prefix); i >= 0 {
			rest := msg[i+len(prefix):]
			for j := 0; j < len(rest); j++ {
				if rest[j] < '0' || rest[j] > '9' {
					rest = rest[:j]
					break
				}
			}
			if n, err := strconv.Atoi(strings.TrimSpace(rest)); err == nil {
				return n
			}
		}
	}
	return 0
}

var _ Provider = (*MiMoClient)(nil)

// mimoAuthHeaders sets MiMo-preferred authentication on outbound requests.
func mimoAuthHeaders(req *http.Request, apiKey string) {
	xiaomi.SetMimoRequestAuth(req, apiKey)
}
