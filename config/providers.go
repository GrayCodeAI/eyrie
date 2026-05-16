package config

import (
	"net/url"
	"os"
	"strings"
)

// Default base URLs for each provider.
const (
	DefaultOpenAIBaseURL           = "https://api.openai.com/v1"
	DefaultOpenRouterOpenAIBaseURL = "https://openrouter.ai/api/v1"
	DefaultCanopyWaveOpenAIBaseURL = "https://inference.canopywave.io/v1"
	DefaultGeminiOpenAIBaseURL     = "https://generativelanguage.googleapis.com/v1beta/openai"
	DefaultAnthropicOpenAIBaseURL  = "https://api.anthropic.com/v1"
	DefaultGrokOpenAIBaseURL       = "https://api.x.ai/v1"
	DefaultOpenCodeGoBaseURL       = "https://opencode.ai/zen/go/v1"
)

// ProviderTransport is the transport type for provider requests.
type ProviderTransport string

const TransportChatCompletions ProviderTransport = "chat_completions"

// ReasoningEffort levels.
type ReasoningEffort string

const (
	ReasoningLow    ReasoningEffort = "low"
	ReasoningMedium ReasoningEffort = "medium"
	ReasoningHigh   ReasoningEffort = "high"
)

// ResolvedProviderRequest holds the resolved provider request details.
type ResolvedProviderRequest struct {
	Transport      ProviderTransport `json:"transport"`
	RequestedModel string            `json:"requested_model"`
	ResolvedModel  string            `json:"resolved_model"`
	BaseURL        string            `json:"base_url"`
	Reasoning      *struct {
		Effort ReasoningEffort `json:"effort"`
	} `json:"reasoning,omitempty"`
}

// IsLocalProviderURL checks if a base URL points to localhost.
func IsLocalProviderURL(baseURL string) bool {
	if baseURL == "" {
		return false
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return false
	}
	h := u.Hostname()
	return h == "localhost" || h == "127.0.0.1" || h == "::1"
}

// ResolveProviderRequest resolves model/baseUrl/transport from options and env.
func ResolveProviderRequest(model, baseURL, fallbackModel string) ResolvedProviderRequest {
	requested := strings.TrimSpace(model)
	if requested == "" {
		requested = strings.TrimSpace(os.Getenv("OPENAI_MODEL"))
	}
	if requested == "" {
		requested = strings.TrimSpace(fallbackModel)
	}
	if requested == "" {
		requested = "gpt-4o"
	}

	resolvedModel := requested
	var reasoning *struct {
		Effort ReasoningEffort `json:"effort"`
	}
	if idx := strings.Index(requested, "?"); idx != -1 {
		resolvedModel = strings.TrimSpace(requested[:idx])
		params, _ := url.ParseQuery(requested[idx+1:])
		if r := strings.TrimSpace(strings.ToLower(params.Get("reasoning"))); r == "low" || r == "medium" || r == "high" {
			reasoning = &struct {
				Effort ReasoningEffort `json:"effort"`
			}{Effort: ReasoningEffort(r)}
		}
	}

	rawBase := baseURL
	if rawBase == "" {
		rawBase = os.Getenv("OPENAI_BASE_URL")
	}
	if rawBase == "" {
		rawBase = os.Getenv("OPENAI_API_BASE")
	}
	if rawBase == "" {
		rawBase = DefaultOpenAIBaseURL
	}
	rawBase = strings.TrimRight(rawBase, "/")

	return ResolvedProviderRequest{
		Transport:      TransportChatCompletions,
		RequestedModel: requested,
		ResolvedModel:  resolvedModel,
		BaseURL:        rawBase,
		Reasoning:      reasoning,
	}
}
