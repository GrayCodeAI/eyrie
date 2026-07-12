package client

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/GrayCodeAI/eyrie/client/core"
)

// Functional-option setter surface for the two protocol adapters (see
// core.Configurable in options.go). Exported so the implementations keep
// satisfying the interface when the adapters move to their own package.

var (
	_ core.Configurable = (*AnthropicClient)(nil)
	_ core.Configurable = (*OpenAIClient)(nil)
)

// SetTimeout sets the HTTP client timeout.
func (c *AnthropicClient) SetTimeout(d time.Duration) { c.httpClient.Timeout = d }

// SetHTTPClient replaces the HTTP client.
func (c *AnthropicClient) SetHTTPClient(hc *http.Client) { c.httpClient = hc }

// SetRetry sets the retry configuration.
func (c *AnthropicClient) SetRetry(rc RetryConfig) { c.retry = rc }

// SetLogger sets the logger.
func (c *AnthropicClient) SetLogger(l *slog.Logger) { c.logger = l }

// SetAPIKey sets the API key.
func (c *AnthropicClient) SetAPIKey(key string) { c.apiKey = key }

// SetBaseURL sets the base URL.
func (c *AnthropicClient) SetBaseURL(url string) { c.baseURL = url }

// SetDefaultModel sets the default model for requests.
func (c *AnthropicClient) SetDefaultModel(model string) { c.defaultModel = model }

// SetDefaultMaxTokens sets the default max tokens for requests.
func (c *AnthropicClient) SetDefaultMaxTokens(n int) { c.defaultMaxTokens = n }

// SetDefaultTemperature sets the default temperature for requests.
func (c *AnthropicClient) SetDefaultTemperature(t float64) { c.defaultTemperature = &t }

// SetGuardrails attaches output guardrails.
func (c *AnthropicClient) SetGuardrails(g *Guardrails) { c.guardrails = g }

// SetProviderName is a no-op: the Anthropic adapter reports a fixed provider
// name. Present to satisfy core.Configurable; WithProviderName only affects
// OpenAI-compatible adapters.
func (c *AnthropicClient) SetProviderName(string) {}

// SetMimoAuth switches authentication to MiMo's api-key header.
func (c *AnthropicClient) SetMimoAuth() { c.useMimoAuth = true }

// SetTimeout sets the HTTP client timeout.
func (c *OpenAIClient) SetTimeout(d time.Duration) { c.httpClient.Timeout = d }

// SetHTTPClient replaces the HTTP client.
func (c *OpenAIClient) SetHTTPClient(hc *http.Client) { c.httpClient = hc }

// SetRetry sets the retry configuration.
func (c *OpenAIClient) SetRetry(rc RetryConfig) { c.retry = rc }

// SetLogger sets the logger.
func (c *OpenAIClient) SetLogger(l *slog.Logger) { c.logger = l }

// SetAPIKey sets the API key.
func (c *OpenAIClient) SetAPIKey(key string) { c.apiKey = key }

// SetBaseURL sets the base URL.
func (c *OpenAIClient) SetBaseURL(url string) { c.baseURL = url }

// SetDefaultModel sets the default model for requests.
func (c *OpenAIClient) SetDefaultModel(model string) { c.defaultModel = model }

// SetDefaultMaxTokens sets the default max tokens for requests.
func (c *OpenAIClient) SetDefaultMaxTokens(n int) { c.defaultMaxTokens = n }

// SetDefaultTemperature sets the default temperature for requests.
func (c *OpenAIClient) SetDefaultTemperature(t float64) { c.defaultTemperature = &t }

// SetGuardrails attaches output guardrails.
func (c *OpenAIClient) SetGuardrails(g *Guardrails) { c.guardrails = g }

// SetProviderName sets the provider name used in errors and logging.
func (c *OpenAIClient) SetProviderName(name string) { c.providerName = name }

// SetMimoAuth switches authentication to MiMo's api-key header.
func (c *OpenAIClient) SetMimoAuth() { c.useMimoAuth = true }
