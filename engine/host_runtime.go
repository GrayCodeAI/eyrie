package engine

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/GrayCodeAI/eyrie/catalog/registry"
	"github.com/GrayCodeAI/eyrie/client"
	"github.com/GrayCodeAI/eyrie/config"
	"github.com/GrayCodeAI/eyrie/credentials"
)

// CustomGatewayCapabilities declares capabilities known to be supported by a
// custom OpenAI-compatible gateway. A nil declaration is permissive for
// backward compatibility: the remote gateway remains the source of truth.
type CustomGatewayCapabilities struct {
	Streaming      bool `json:"streaming,omitempty"`
	Tools          bool `json:"tools,omitempty"`
	Vision         bool `json:"vision,omitempty"`
	StructuredJSON bool `json:"structured_json,omitempty"`
	Reasoning      bool `json:"reasoning,omitempty"`
}

// CustomGateway describes a user-configured OpenAI-compatible gateway. It
// contains only routing metadata; credential material remains in Eyrie's
// secret store and is discovered by CredentialEnv.
type CustomGateway struct {
	ID             string                     `json:"id"`
	DisplayName    string                     `json:"display_name,omitempty"`
	BaseURL        string                     `json:"base_url"`
	CredentialEnv  string                     `json:"credential_env,omitempty"`
	DefaultModel   string                     `json:"default_model,omitempty"`
	MaxTokensField string                     `json:"max_tokens_field,omitempty"`
	ContextWindow  int                        `json:"context_window,omitempty"`
	Capabilities   *CustomGatewayCapabilities `json:"capabilities,omitempty"`
}

var customGatewayRegistry = struct {
	sync.RWMutex
	gateways map[string]CustomGateway
}{gateways: make(map[string]CustomGateway)}

// RegisterCustomGateway registers safe routing metadata for a host-supplied
// OpenAI-compatible gateway. Registration must happen before Engine creation;
// each Engine snapshots the metadata and always resolves credentials through
// its own injected SecretStore.
func RegisterCustomGateway(gateway CustomGateway) error {
	id := NormalizeProviderID(gateway.ID)
	if id == "" {
		return fmt.Errorf("eyrie engine: custom gateway ID is required")
	}
	if builtIn, ok := registry.SpecByProviderID(id); ok {
		return fmt.Errorf("eyrie engine: custom gateway ID %q collides with built-in gateway %q", id, builtIn.ProviderID)
	}
	baseURL := strings.TrimSpace(gateway.BaseURL)
	if baseURL == "" {
		return fmt.Errorf("eyrie engine: custom gateway %q base URL is required", id)
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return fmt.Errorf("eyrie engine: custom gateway %q has invalid baseURL %q (must be http/https with host)", id, baseURL)
	}
	if gateway.ContextWindow < 0 {
		return fmt.Errorf("eyrie engine: custom gateway %q context window cannot be negative", id)
	}
	maxTokensField := strings.TrimSpace(gateway.MaxTokensField)
	if maxTokensField == "" {
		maxTokensField = "max_tokens"
	}
	if maxTokensField != "max_tokens" && maxTokensField != "max_completion_tokens" {
		return fmt.Errorf("eyrie engine: custom gateway %q max tokens field must be max_tokens or max_completion_tokens", id)
	}
	gateway.ID = id
	gateway.BaseURL = strings.TrimRight(baseURL, "/")
	gateway.DisplayName = strings.TrimSpace(gateway.DisplayName)
	if gateway.DisplayName == "" {
		gateway.DisplayName = id
	}
	gateway.CredentialEnv = strings.TrimSpace(gateway.CredentialEnv)
	gateway.DefaultModel = strings.TrimSpace(gateway.DefaultModel)
	gateway.MaxTokensField = maxTokensField
	customGatewayRegistry.Lock()
	customGatewayRegistry.gateways[id] = cloneCustomGateway(gateway)
	customGatewayRegistry.Unlock()
	return nil
}

func snapshotCustomGateways() map[string]CustomGateway {
	customGatewayRegistry.RLock()
	defer customGatewayRegistry.RUnlock()
	out := make(map[string]CustomGateway, len(customGatewayRegistry.gateways))
	for id, gateway := range customGatewayRegistry.gateways {
		out[id] = cloneCustomGateway(gateway)
	}
	return out
}

func cloneCustomGateway(gateway CustomGateway) CustomGateway {
	if gateway.Capabilities != nil {
		capabilities := *gateway.Capabilities
		gateway.Capabilities = &capabilities
	}
	return gateway
}

func (e *Engine) customGateway(providerID string) (CustomGateway, bool) {
	if e == nil {
		return CustomGateway{}, false
	}
	gateway, ok := e.customGateways[NormalizeProviderID(providerID)]
	return cloneCustomGateway(gateway), ok
}

func (e *Engine) resolveCustomSelection(req SelectionRequest) (Route, bool, error) {
	providerID := NormalizeProviderID(req.Preference.PreferredProvider)
	modelID := strings.TrimSpace(req.Preference.PreferredModelID)
	state := config.LoadProviderConfig(e.providerConfigPath)
	if providerID == "" {
		providerID = NormalizeProviderID(config.ActiveProvider(state))
	}
	gateway, ok := e.customGateway(providerID)
	if !ok {
		return Route{}, false, nil
	}
	if modelID == "" {
		modelID = strings.TrimSpace(config.ActiveModel(state))
	}
	if modelID == "" {
		modelID = gateway.DefaultModel
	}
	if modelID == "" {
		return Route{}, true, &Error{
			Code: ErrorModelUnavailable, Operation: "resolve", Provider: gateway.ID,
			Message: fmt.Sprintf("eyrie engine: custom gateway %q requires a model", gateway.ID),
		}
	}
	if err := validateCustomGatewayRequirements(gateway, modelID, req.Requirements); err != nil {
		return Route{}, true, err
	}
	return Route{Provider: gateway.ID, Model: modelID, DeploymentRouting: false}, true, nil
}

func validateCustomGatewayRequirements(gateway CustomGateway, modelID string, req Requirements) error {
	if gateway.ContextWindow > 0 && req.MinimumContext > gateway.ContextWindow {
		return &Error{
			Code: ErrorCapabilityMismatch, Operation: "resolve", Provider: gateway.ID, Model: modelID,
			Message: fmt.Sprintf("eyrie engine: custom gateway %q context window %d is below requested %d", gateway.ID, gateway.ContextWindow, req.MinimumContext),
		}
	}
	capabilities := gateway.Capabilities
	if capabilities == nil {
		return nil
	}
	supported := (!req.Streaming || capabilities.Streaming) &&
		(!req.Tools || capabilities.Tools) &&
		(!req.Vision || capabilities.Vision) &&
		(!req.StructuredJSON || capabilities.StructuredJSON) &&
		(!req.Reasoning || capabilities.Reasoning)
	if supported {
		return nil
	}
	return &Error{
		Code: ErrorCapabilityMismatch, Operation: "resolve", Provider: gateway.ID, Model: modelID,
		Message: fmt.Sprintf("eyrie engine: custom gateway %q does not satisfy requested capabilities", gateway.ID),
	}
}

func (e *Engine) customGatewayTransport(ctx context.Context, route Route) (client.Provider, bool, error) {
	gateway, ok := e.customGateway(route.Provider)
	if !ok {
		return nil, false, nil
	}
	secret := ""
	if gateway.CredentialEnv != "" {
		var err error
		secret, err = e.secretStore.Get(nonNilContext(ctx), credentials.AccountForEnv(gateway.CredentialEnv))
		if errors.Is(err, credentials.ErrNotFound) {
			return nil, true, &Error{
				Code: ErrorCredentialMissing, Operation: "resolve_transport", Provider: gateway.ID, Model: route.Model,
				Message: fmt.Sprintf("eyrie engine: credential for custom gateway %q is not configured", gateway.ID),
			}
		}
		if err != nil {
			return nil, true, &Error{
				Code: ErrorInternal, Operation: "resolve_transport", Provider: gateway.ID, Model: route.Model,
				Message: fmt.Sprintf("eyrie engine: credential store unavailable for custom gateway %q", gateway.ID), Cause: err,
			}
		}
		if strings.TrimSpace(secret) == "" || config.LooksLikePlaceholderSecret(secret) {
			return nil, true, &Error{
				Code: ErrorCredentialMissing, Operation: "resolve_transport", Provider: gateway.ID, Model: route.Model,
				Message: fmt.Sprintf("eyrie engine: credential for custom gateway %q is not configured", gateway.ID),
			}
		}
	}
	compat := &client.OpenAICompatConfig{MaxTokensField: gateway.MaxTokensField}
	provider := client.NewOpenAIClient(secret, gateway.BaseURL, compat, client.WithProviderName(gateway.ID))
	return provider, true, nil
}

func customGatewayCapabilityNames(gateway CustomGateway) []string {
	if gateway.Capabilities == nil {
		return nil
	}
	var out []string
	if gateway.Capabilities.Streaming {
		out = append(out, "streaming")
	}
	if gateway.Capabilities.Tools {
		out = append(out, "tools")
	}
	if gateway.Capabilities.Vision {
		out = append(out, "vision")
	}
	if gateway.Capabilities.StructuredJSON {
		out = append(out, "structured_json")
	}
	if gateway.Capabilities.Reasoning {
		out = append(out, "reasoning")
	}
	return out
}

// ParseInlineToolCalls normalizes provider text-embedded tool calls into the
// stable engine contract. Providers that emit structured tool calls bypass
// this fallback.
func ParseInlineToolCalls(content string) (string, []ToolCall) {
	clean, calls := client.ParseInlineToolCalls(content)
	if len(calls) == 0 {
		return clean, nil
	}
	out := make([]ToolCall, 0, len(calls))
	for _, call := range calls {
		out = append(out, ToolCall{ID: call.ID, Name: call.Name, Arguments: call.Arguments})
	}
	return clean, out
}
