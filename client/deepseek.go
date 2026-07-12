package client

import (
	"context"
	"log/slog"
	"strings"
)

// DeepSeekClient uses OpenAI-compatible DeepSeek endpoints first,
// with optional Anthropic-compat fallback if the OpenAI endpoint is down.
type DeepSeekClient struct {
	router ProtocolRouter
	logger *slog.Logger
}

// NewDeepSeekClient builds a DeepSeek provider client.
// openAIBase is typically "https://api.deepseek.com/v1"
// anthropicBase is typically "https://api.deepseek.com/anthropic"
func NewDeepSeekClient(apiKey, openAIBase, anthropicBase string, compat *OpenAICompatConfig, opts ...ClientOption) *DeepSeekClient {
	openAIBase = strings.TrimRight(strings.TrimSpace(openAIBase), "/")
	anthropicBase = strings.TrimRight(strings.TrimSpace(anthropicBase), "/")
	dsOpts := append(append([]ClientOption{}, opts...), WithProviderName("deepseek"))
	o := NewOpenAIClient(apiKey, openAIBase, compat, dsOpts...)
	var a *AnthropicClient
	if anthropicBase != "" {
		a = NewAnthropicClient(apiKey, anthropicBase, dsOpts...)
	}
	return &DeepSeekClient{
		router: ProtocolRouter{OpenAI: o, Anthropic: a},
		logger: slog.Default(),
	}
}

func (c *DeepSeekClient) Name() string { return "deepseek" }

func (c *DeepSeekClient) Chat(ctx context.Context, messages []EyrieMessage, opts ChatOptions) (*EyrieResponse, error) {
	return c.router.Chat(ctx, messages, opts, ChatProtocolCompletions, func(err error, _ *EyrieResponse) bool {
		if err != nil && c.router.Anthropic != nil && isRetriableError(err) {
			c.logger.Info("DeepSeek: OpenAI endpoint failed; retrying via Anthropic compatibility", "error", err)
			return true
		}
		return false
	})
}

func (c *DeepSeekClient) StreamChat(ctx context.Context, messages []EyrieMessage, opts ChatOptions) (*StreamResult, error) {
	return c.router.StreamChat(ctx, messages, opts, ProtocolStreamConfig{
		Primary: ChatProtocolCompletions,
		FallbackOnError: func(err error) bool {
			if c.router.Anthropic != nil && isRetriableError(err) {
				c.logger.Info("DeepSeek: OpenAI stream failed; retrying via Anthropic compatibility", "error", err)
				return true
			}
			return false
		},
	})
}

func (c *DeepSeekClient) Ping(ctx context.Context) error {
	if err := c.router.OpenAI.Ping(ctx); err == nil {
		return nil
	} else if c.router.Anthropic == nil || !isRetriableError(err) {
		return err
	}
	return c.router.Anthropic.Ping(ctx)
}

var _ Provider = (*DeepSeekClient)(nil)
