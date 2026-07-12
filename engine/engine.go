package engine

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/GrayCodeAI/eyrie/catalog"
	"github.com/GrayCodeAI/eyrie/catalog/registry"
	"github.com/GrayCodeAI/eyrie/client"
	"github.com/GrayCodeAI/eyrie/config"
	"github.com/GrayCodeAI/eyrie/credentials"
	"github.com/GrayCodeAI/eyrie/setup"
)

// ContractVersion is the compatibility version of the host-facing API.
const ContractVersion = "1"

// Options supplies host-owned dependencies. A zero value uses Eyrie's safe
// defaults. Product-specific paths are deliberately not inferred here.
type Options struct {
	SecretStore        credentials.Store
	StateDir           string
	CatalogPath        string
	ProviderConfigPath string
}

// Engine is Eyrie's narrow host facade. It is safe for concurrent use when
// the configured SecretStore is safe for concurrent use.
type Engine struct {
	secretStore        credentials.Store
	catalogPath        string
	providerConfigPath string
	resolveTransport   func(context.Context, Route) (client.Provider, error)
}

// New constructs a host-facing Eyrie engine.
func New(opts Options) (*Engine, error) {
	store := opts.SecretStore
	if store == nil {
		store = credentials.DefaultStore()
	}
	if store == nil {
		return nil, &Error{Code: ErrorInternal, Operation: "new", Message: "eyrie engine: credential store unavailable"}
	}
	stateDir := strings.TrimSpace(opts.StateDir)
	catalogPath := strings.TrimSpace(opts.CatalogPath)
	providerPath := strings.TrimSpace(opts.ProviderConfigPath)
	if catalogPath == "" && stateDir != "" {
		catalogPath = filepath.Join(stateDir, "model_catalog.json")
	}
	if providerPath == "" && stateDir != "" {
		providerPath = filepath.Join(stateDir, "provider.json")
	}
	if catalogPath == "" {
		catalogPath = catalog.DefaultCachePath()
	}
	if providerPath == "" {
		providerPath = config.GetProviderConfigPath()
	}
	engine := &Engine{secretStore: store, catalogPath: catalogPath, providerConfigPath: providerPath}
	engine.resolveTransport = engine.defaultTransport
	return engine, nil
}

// SelectionRequest asks Eyrie to resolve a concrete provider/model route.
type SelectionRequest struct {
	Requirements Requirements
	Preference   Preference
}

// Resolve selects a concrete provider/model and verifies known catalog
// requirements before transport construction.
func (e *Engine) Resolve(ctx context.Context, req SelectionRequest) (Route, error) {
	ctx = nonNilContext(ctx)
	selection, err := e.resolveSelection(ctx, req)
	if err != nil {
		return Route{}, err
	}
	return selection, nil
}

// Generate performs a blocking generation through Eyrie's resolved transport.
func (e *Engine) Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
	ctx = nonNilContext(ctx)
	if err := validateGenerateRequest(req); err != nil {
		return nil, err
	}
	route, provider, err := e.resolveProvider(ctx, req)
	if err != nil {
		return nil, err
	}
	callCtx, cancel := requestContext(ctx, req.Limits.Timeout)
	defer cancel()
	resp, err := provider.Chat(callCtx, toClientMessages(req.Messages), toClientOptions(req, route, false))
	if err != nil {
		return nil, classify("generate", route, err)
	}
	return fromClientResponse(resp, route), nil
}

// Stream starts a normalized streaming generation. The returned stream must be
// closed by the caller.
func (e *Engine) Stream(ctx context.Context, req GenerateRequest) (*Stream, error) {
	ctx = nonNilContext(ctx)
	if err := validateGenerateRequest(req); err != nil {
		return nil, err
	}
	route, provider, err := e.resolveProvider(ctx, req)
	if err != nil {
		return nil, err
	}
	callCtx, cancel := requestContext(ctx, req.Limits.Timeout)
	source, err := streamWithContinuation(callCtx, provider, toClientMessages(req.Messages), toClientOptions(req, route, true), req.Limits)
	if err != nil {
		cancel()
		return nil, classify("stream", route, err)
	}
	return newStream(callCtx, cancel, source, route), nil
}

// ListModels returns provider models through the stable facade.
func (e *Engine) ListModels(ctx context.Context, providerID string, refresh bool) ([]Model, error) {
	ctx = nonNilContext(ctx)
	var snapshot CatalogSnapshot
	var err error
	if refresh {
		snapshot, err = e.RefreshCatalog(ctx, providerID)
	} else {
		snapshot, err = e.Catalog(ctx)
	}
	if err != nil {
		return nil, &Error{Code: ErrorCatalogUnavailable, Operation: "list_models", Provider: providerID, Message: err.Error(), Cause: err}
	}
	providerID = catalog.CanonicalProviderID(strings.TrimSpace(providerID))
	if providerID == "" {
		return snapshot.Models, nil
	}
	compiled, loadErr := catalog.LoadCatalog(ctx, catalog.LoadCatalogOptions{CachePath: e.catalogPath, RequireCache: true})
	if loadErr != nil {
		return nil, &Error{Code: ErrorCatalogUnavailable, Operation: "list_models", Provider: providerID, Message: loadErr.Error(), Cause: loadErr}
	}
	entries := catalog.ModelEntriesForProvider(compiled, providerID)
	out := make([]Model, 0, len(entries))
	for _, entry := range entries {
		capabilities := append([]string(nil), entry.ServerTools...)
		if canonical, ok := catalog.CanonicalModelForProviderNative(compiled, providerID, entry.ID); ok {
			capabilities = capabilityNames(offeringForProvider(compiled, providerID, canonical, entry.ID).Capabilities)
		}
		out = append(out, Model{
			ID: entry.ID, DisplayName: entry.DisplayName, Owner: entry.Owner, ProviderID: providerID,
			ContextWindow: entry.ContextWindow, MaxOutputTokens: entry.MaxOutput,
			InputPricePer1M: entry.InputPricePer1M, OutputPricePer1M: entry.OutputPricePer1M,
			Capabilities: capabilities, Source: "cache",
		})
	}
	return out, nil
}

func offeringForProvider(compiled *catalog.CompiledCatalog, providerID, canonicalID, nativeID string) catalog.ModelOffering {
	spec, ok := registry.SpecByProviderID(providerID)
	if !ok {
		return firstOffering(compiled.OfferingsByCanonicalModel[canonicalID])
	}
	for _, offering := range compiled.OfferingsByDeployment[spec.DeploymentID] {
		if offering.CanonicalModelID == canonicalID && (nativeID == "" || offering.NativeModelID == nativeID) {
			return offering
		}
	}
	return firstOffering(compiled.OfferingsByCanonicalModel[canonicalID])
}

func (e *Engine) resolveProvider(ctx context.Context, req GenerateRequest) (Route, client.Provider, error) {
	route, err := e.resolveSelection(ctx, SelectionRequest{Requirements: req.Requirements, Preference: req.Preference})
	if err != nil {
		return Route{}, nil, err
	}
	provider, err := e.resolveTransport(ctx, route)
	if err != nil {
		return Route{}, nil, classify("resolve_transport", route, err)
	}
	return route, provider, nil
}

func (e *Engine) defaultTransport(ctx context.Context, _ Route) (client.Provider, error) {
	compiled, cfg, err := e.loadRuntimeState(ctx)
	if err != nil {
		return nil, err
	}
	return setup.DeploymentProviderFromCatalog(cfg, compiled)
}

func (e *Engine) resolveSelection(ctx context.Context, req SelectionRequest) (Route, error) {
	compiled, cfg, err := e.loadRuntimeState(ctx)
	if err != nil {
		return Route{}, err
	}
	selection := Route{
		Provider:          strings.TrimSpace(req.Preference.PreferredProvider),
		Model:             strings.TrimSpace(req.Preference.PreferredModelID),
		DeploymentRouting: true,
	}
	if selection.Provider == "" {
		selection.Provider = config.ActiveProvider(cfg)
	}
	if selection.Model == "" {
		selection.Model = config.ActiveModel(cfg)
	}
	if selection.Provider == "" && selection.Model != "" {
		selection.Provider = catalog.GatewayForModel(compiled, selection.Model)
		if selection.Provider == "" {
			selection.Provider = catalog.ProviderForModel(compiled, selection.Model)
		}
	}
	if strings.TrimSpace(selection.Model) != "" {
		err := validateRequirementsFromCatalog(compiled, selection.Model, req.Requirements)
		if err == nil {
			return selection, nil
		}
		// A user-selected exact model is a hard constraint unless fallback was
		// explicitly permitted. Persisted/default selections may be replaced by
		// a capability-compatible route.
		if req.Preference.PreferredModelID != "" && !req.Preference.AllowFallback {
			return Route{}, err
		}
	}
	modelID, providerID := selectCompatibleModel(compiled, req)
	if modelID == "" {
		return Route{}, &Error{Code: ErrorCapabilityMismatch, Operation: "resolve", Message: "eyrie engine: no catalog model satisfies the requested capabilities"}
	}
	selection.Model = modelID
	selection.Provider = providerID
	return selection, nil
}

type modelCandidate struct {
	model    catalog.Model
	offering catalog.ModelOffering
	provider string
	cost     float64
}

func selectCompatibleModel(compiled *catalog.CompiledCatalog, req SelectionRequest) (string, string) {
	if compiled == nil {
		return "", ""
	}
	preferredProvider := catalog.CanonicalProviderID(req.Preference.PreferredProvider)
	var candidates []modelCandidate
	for id, model := range compiled.ModelsByID {
		if req.Requirements.MinimumContext > 0 && model.ContextWindow < req.Requirements.MinimumContext {
			continue
		}
		if !offeringSupports(compiled, id, req.Requirements) {
			continue
		}
		provider := catalog.CanonicalProviderID(catalog.GatewayForModel(compiled, id))
		if provider == "" {
			provider = catalog.CanonicalProviderID(model.ProviderID)
		}
		if preferredProvider != "" && provider != preferredProvider {
			continue
		}
		offering := firstOffering(compiled.OfferingsByCanonicalModel[id])
		cost := offering.Pricing.RatesPer1M["input_tokens"] + offering.Pricing.RatesPer1M["output_tokens"]
		candidates = append(candidates, modelCandidate{model: model, offering: offering, provider: provider, cost: cost})
	}
	if len(candidates) == 0 {
		return "", ""
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		a, b := candidates[i], candidates[j]
		switch req.Preference.Intent {
		case IntentEconomical:
			if a.cost != b.cost {
				return a.cost < b.cost
			}
		case IntentReasoning:
			if a.model.ContextWindow != b.model.ContextWindow {
				return a.model.ContextWindow > b.model.ContextWindow
			}
		case IntentFast:
			if a.model.MaxOutput != b.model.MaxOutput {
				return a.model.MaxOutput < b.model.MaxOutput
			}
		}
		return a.model.ID < b.model.ID
	})
	return candidates[0].model.ID, candidates[0].provider
}

func validateGenerateRequest(req GenerateRequest) error {
	if len(req.Messages) == 0 {
		return invalid("generate", "eyrie engine: at least one message is required")
	}
	for _, message := range req.Messages {
		if strings.TrimSpace(message.Role) == "" {
			return invalid("generate", "eyrie engine: every message requires a role")
		}
	}
	if req.Limits.MaxOutputTokens < 0 {
		return invalid("generate", "eyrie engine: max output tokens cannot be negative")
	}
	if req.Limits.MaxContinuations < 0 || req.Limits.MaxTotalOutputTokens < 0 {
		return invalid("generate", "eyrie engine: continuation limits cannot be negative")
	}
	if req.Requirements.MinimumContext < 0 {
		return invalid("generate", "eyrie engine: minimum context cannot be negative")
	}
	return nil
}

func validateRequirementsFromCatalog(compiled *catalog.CompiledCatalog, modelID string, req Requirements) error {
	if req.MinimumContext <= 0 && !req.Tools && !req.Vision && !req.StructuredJSON && !req.Reasoning {
		return nil
	}
	if compiled == nil {
		return &Error{Code: ErrorCatalogUnavailable, Operation: "validate_capabilities", Model: modelID, Message: "eyrie engine: catalog unavailable for capability validation"}
	}
	canonical, ok := compiled.CanonicalModelForAliasOrID(modelID)
	if !ok {
		canonical = modelID
	}
	model, ok := compiled.ModelsByID[canonical]
	if !ok {
		return &Error{Code: ErrorModelUnavailable, Operation: "validate_capabilities", Model: modelID, Message: fmt.Sprintf("eyrie engine: model %q is not in the catalog", modelID)}
	}
	if req.MinimumContext > 0 && model.ContextWindow < req.MinimumContext {
		return &Error{Code: ErrorCapabilityMismatch, Operation: "validate_capabilities", Model: canonical, Message: fmt.Sprintf("eyrie engine: model %q has context window %d, need at least %d", canonical, model.ContextWindow, req.MinimumContext)}
	}
	if req.Tools || req.Vision || req.StructuredJSON || req.Reasoning {
		if !offeringSupports(compiled, canonical, req) {
			return &Error{Code: ErrorCapabilityMismatch, Operation: "validate_capabilities", Model: canonical, Message: fmt.Sprintf("eyrie engine: model %q does not satisfy requested capabilities", canonical)}
		}
	}
	return nil
}

func offeringSupports(compiled *catalog.CompiledCatalog, modelID string, req Requirements) bool {
	offerings := compiled.OfferingsByCanonicalModel[modelID]
	if len(offerings) == 0 {
		return !req.Tools && !req.Vision && !req.StructuredJSON && !req.Reasoning
	}
	for _, offering := range offerings {
		caps := offering.Capabilities
		if req.Tools && caps.FunctionCalling != catalog.CapabilitySupported {
			continue
		}
		if req.Vision && caps.ImageInput != catalog.CapabilitySupported {
			continue
		}
		if req.StructuredJSON && caps.StructuredOutput != catalog.CapabilitySupported {
			continue
		}
		if req.Reasoning && caps.ExplicitThinkingBudget != catalog.CapabilitySupported && caps.AdaptiveThinking != catalog.CapabilitySupported && caps.Effort != catalog.CapabilitySupported {
			continue
		}
		return true
	}
	return false
}

func requestContext(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout > 0 {
		return context.WithTimeout(ctx, timeout)
	}
	return context.WithCancel(ctx)
}

func nonNilContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
