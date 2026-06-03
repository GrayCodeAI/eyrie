package client

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/GrayCodeAI/eyrie/catalog/xiaomi"
)

// MiMoClient uses OpenAI-compatible MiMo endpoints first, with optional Anthropic-compat fallback.
type MiMoClient struct {
	openAI     *OpenAIClient
	anthropic  *AnthropicClient
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
	return &MiMoClient{openAI: o, anthropic: a, providerID: providerID, logger: slog.Default()}
}

// WithProviderName sets the OpenAI client provider name for errors/logging.
func WithProviderName(name string) ClientOption {
	return ClientOption{
		applyOpenAIFn: func(c *OpenAIClient) { c.providerName = name },
	}
}

// WithMimoAuth uses api-key header per MiMo documentation (OpenAI + Anthropic compat).
func WithMimoAuth() ClientOption {
	return ClientOption{
		applyFn:       func(c *AnthropicClient) { c.useMimoAuth = true },
		applyOpenAIFn: func(c *OpenAIClient) { c.useMimoAuth = true },
	}
}

func (c *MiMoClient) Name() string {
	if c.openAI != nil {
		return c.openAI.Name()
	}
	return c.providerID
}

func (c *MiMoClient) Chat(ctx context.Context, messages []EyrieMessage, opts ChatOptions) (*EyrieResponse, error) {
	resp, err := c.openAI.Chat(ctx, messages, opts)
	if err == nil || c.anthropic == nil || !mimoFallbackChatError(err) {
		return resp, err
	}
	c.logger.Info("MiMo: OpenAI endpoint failed; retrying via Anthropic compatibility", "provider", c.providerID, "error", err)
	return c.anthropic.Chat(ctx, messages, opts)
}

func (c *MiMoClient) StreamChat(ctx context.Context, messages []EyrieMessage, opts ChatOptions) (*StreamResult, error) {
	result, err := c.openAI.StreamChat(ctx, messages, opts)
	if err == nil || c.anthropic == nil || !mimoFallbackChatError(err) {
		return result, err
	}
	c.logger.Info("MiMo: OpenAI stream failed; retrying via Anthropic compatibility", "provider", c.providerID, "error", err)
	return c.anthropic.StreamChat(ctx, messages, opts)
}

func (c *MiMoClient) Ping(ctx context.Context) error {
	if err := c.openAI.Ping(ctx); err == nil {
		return nil
	} else if c.anthropic == nil || !mimoRetryableChatError(err) {
		return err
	}
	return c.anthropic.Ping(ctx)
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
	msg := err.Error()
	if strings.Contains(msg, "connection reset") || strings.Contains(msg, "timeout") {
		return true
	}
	for _, code := range []int{
		http.StatusUnauthorized, http.StatusForbidden,
		http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout,
	} {
		if strings.Contains(msg, fmt.Sprintf("%d", code)) && xiaomi.IsRetryableHTTPStatus(code) {
			return true
		}
	}
	if n := parseHTTPStatusFromError(msg); n > 0 {
		return xiaomi.IsRetryableHTTPStatus(n)
	}
	return false
}

func parseHTTPStatusFromError(msg string) int {
	for _, prefix := range []string{"HTTP ", "status ", "error ("} {
		if i := strings.Index(msg, prefix); i >= 0 {
			rest := msg[i+len(prefix):]
			for j := 0; j < len(rest) && j < 3; j++ {
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
