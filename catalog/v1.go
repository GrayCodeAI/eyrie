package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	CatalogV1SchemaVersion = "model-catalog/v1"
	DefaultCatalogV1URL    = "https://langdag.com/model-catalog/v1/catalog.json"
)

type CapabilityState string

const (
	CapabilitySupported   CapabilityState = "supported"
	CapabilityUnsupported CapabilityState = "unsupported"
	CapabilityUnknown     CapabilityState = "unknown"
)

type PricingStatus string

const (
	PricingKnown   PricingStatus = "known"
	PricingPartial PricingStatus = "partial"
	PricingUnknown PricingStatus = "unknown"
	PricingFree    PricingStatus = "free"
)

type NativeModelIDSource string

const (
	NativeModelIDCatalogKnown   NativeModelIDSource = "catalog_known"
	NativeModelIDDiscovered     NativeModelIDSource = "discovered"
	NativeModelIDUserConfigured NativeModelIDSource = "user_configured"
	NativeModelIDCatalogOrUser  NativeModelIDSource = "catalog_or_user_configured"
)

// CatalogV1 separates model ownership from the API protocol and deployment used
// to call the model. It is intentionally data-only; adapters remain code.
type CatalogV1 struct {
	SchemaVersion     string                    `json:"schema_version"`
	GeneratedAt       time.Time                 `json:"generated_at"`
	StaleAfter        time.Time                 `json:"stale_after"`
	Providers         map[string]ProviderV1     `json:"providers"`
	APIProtocols      map[string]APIProtocolV1  `json:"api_protocols"`
	Deployments       map[string]DeploymentV1   `json:"deployments"`
	Models            map[string]ModelV1        `json:"models"`
	Offerings         []ModelOfferingV1         `json:"offerings"`
	OfferingTemplates []ModelOfferingTemplateV1 `json:"offering_templates,omitempty"`
	Aliases           map[string]string         `json:"aliases,omitempty"`
	Provenance        *CatalogProvenanceV1      `json:"provenance,omitempty"`
}

type ProviderV1 struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	Description string               `json:"description,omitempty"`
	HomepageURL string               `json:"homepage_url,omitempty"`
	Aliases     []string             `json:"aliases,omitempty"`
	Provenance  *CatalogProvenanceV1 `json:"provenance,omitempty"`
}

type APIProtocolV1 struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	Description string               `json:"description,omitempty"`
	Provenance  *CatalogProvenanceV1 `json:"provenance,omitempty"`
}

type DeploymentV1 struct {
	ID                     string               `json:"id"`
	Name                   string               `json:"name"`
	ProviderID             string               `json:"provider_id"`
	APIProtocolID          string               `json:"api_protocol_id"`
	AdapterConstructor     string               `json:"adapter_constructor"`
	CredentialRequirements []CredentialV1       `json:"credential_requirements,omitempty"`
	EnvFallbacks           []EnvFallbackV1      `json:"env_fallbacks,omitempty"`
	NativeModelIDSource    NativeModelIDSource  `json:"native_model_id_source"`
	ModelMappingsRequired  bool                 `json:"model_mappings_required,omitempty"`
	Local                  bool                 `json:"local,omitempty"`
	Provenance             *CatalogProvenanceV1 `json:"provenance,omitempty"`
}

type CredentialV1 struct {
	Field    string `json:"field"`
	Secret   bool   `json:"secret,omitempty"`
	Required bool   `json:"required,omitempty"`
}

type EnvFallbackV1 struct {
	Field string   `json:"field"`
	Env   []string `json:"env"`
}

type ModelV1 struct {
	ID            string               `json:"id"`
	ProviderID    string               `json:"provider_id"`
	Name          string               `json:"name"`
	Family        string               `json:"family,omitempty"`
	ContextWindow int                  `json:"context_window,omitempty"`
	MaxOutput     int                  `json:"max_output,omitempty"`
	Aliases       []string             `json:"aliases,omitempty"`
	Provenance    *CatalogProvenanceV1 `json:"provenance,omitempty"`
}

type ModelOfferingV1 struct {
	ID               string               `json:"id"`
	CanonicalModelID string               `json:"canonical_model_id"`
	DeploymentID     string               `json:"deployment_id"`
	NativeModelID    string               `json:"native_model_id"`
	Capabilities     CapabilitySetV1      `json:"capabilities,omitempty"`
	Pricing          PricingV1            `json:"pricing"`
	Provenance       *CatalogProvenanceV1 `json:"provenance,omitempty"`
}

type ModelOfferingTemplateV1 struct {
	ID                  string               `json:"id"`
	CanonicalModelID    string               `json:"canonical_model_id"`
	DeploymentID        string               `json:"deployment_id"`
	NativeModelIDSource NativeModelIDSource  `json:"native_model_id_source"`
	MappingRequired     bool                 `json:"mapping_required"`
	Capabilities        CapabilitySetV1      `json:"capabilities,omitempty"`
	Pricing             PricingV1            `json:"pricing"`
	Provenance          *CatalogProvenanceV1 `json:"provenance,omitempty"`
}

type CapabilitySetV1 struct {
	ServerTools            map[string]CapabilityState `json:"server_tools,omitempty"`
	FunctionCalling        CapabilityState            `json:"function_calling,omitempty"`
	ExplicitThinkingBudget CapabilityState            `json:"explicit_thinking_budget,omitempty"`
}

type PricingV1 struct {
	Status            PricingStatus      `json:"status"`
	Currency          string             `json:"currency,omitempty"`
	EffectiveAt       time.Time          `json:"effective_at,omitempty"`
	RatesPer1M        map[string]float64 `json:"rates_per_1m,omitempty"`
	MissingDimensions []string           `json:"missing_dimensions,omitempty"`
	Notes             []string           `json:"notes,omitempty"`
	Source            string             `json:"source,omitempty"`
}

type CatalogProvenanceV1 struct {
	Source     string    `json:"source"`
	SourceURL  string    `json:"source_url,omitempty"`
	ObservedAt time.Time `json:"observed_at,omitempty"`
}

type CatalogDiagnosticV1 struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type CompiledCatalogV1 struct {
	Catalog                   *CatalogV1
	ProvidersByID             map[string]ProviderV1
	APIProtocolsByID          map[string]APIProtocolV1
	DeploymentsByID           map[string]DeploymentV1
	ModelsByID                map[string]ModelV1
	OfferingsByID             map[string]ModelOfferingV1
	OfferingsByCanonicalModel map[string][]ModelOfferingV1
	OfferingsByDeployment     map[string][]ModelOfferingV1
	TemplatesByCanonicalModel map[string][]ModelOfferingTemplateV1
	Diagnostics               []CatalogDiagnosticV1
}

type LoadCatalogV1Options struct {
	CachePath     string
	RemoteURL     string
	RefreshRemote bool
	HTTPClient    *http.Client
	Timeout       time.Duration
}

func DefaultCatalogV1() CatalogV1 {
	return CatalogV1FromLegacy(DefaultModelCatalog())
}

func CatalogV1FromLegacy(legacy ModelCatalog) CatalogV1 {
	generatedAt := time.Now().UTC().Truncate(time.Second)
	if ts, err := time.Parse(time.RFC3339, legacy.UpdatedAt); err == nil {
		generatedAt = ts
	}
	c := CatalogV1{
		SchemaVersion: CatalogV1SchemaVersion,
		GeneratedAt:   generatedAt,
		StaleAfter:    generatedAt.Add(30 * 24 * time.Hour),
		Providers:     defaultProvidersV1(),
		APIProtocols:  defaultAPIProtocolsV1(),
		Deployments:   defaultDeploymentsV1(),
		Models:        map[string]ModelV1{},
		Aliases:       map[string]string{},
		Provenance:    &CatalogProvenanceV1{Source: legacy.Source, ObservedAt: generatedAt},
	}
	for legacyProvider, entries := range legacy.Providers {
		deploymentID, ownerProviderID := legacyDeploymentAndOwner(legacyProvider)
		if deploymentID == "" {
			continue
		}
		for _, entry := range entries {
			nativeID := strings.TrimSpace(entry.ID)
			if nativeID == "" {
				continue
			}
			modelProviderID := ownerProviderID
			if deploymentID == "openrouter" {
				if owner, _, ok := strings.Cut(nativeID, "/"); ok && owner != "" {
					modelProviderID = canonicalProviderID(owner)
					if c.Providers[modelProviderID].ID == "" {
						c.Providers[modelProviderID] = ProviderV1{ID: modelProviderID, Name: modelProviderID}
					}
				}
			}
			canonicalID := canonicalModelID(modelProviderID, nativeID)
			if c.Models[canonicalID].ID == "" {
				name := entry.DisplayName
				if name == "" {
					name = nativeID
				}
				c.Models[canonicalID] = ModelV1{
					ID:            canonicalID,
					ProviderID:    modelProviderID,
					Name:          name,
					ContextWindow: entry.ContextWindow,
					MaxOutput:     entry.MaxOutput,
					Aliases:       uniqueNonEmpty(nativeID, entry.DisplayName),
				}
			}
			if c.Aliases[nativeID] == "" {
				c.Aliases[nativeID] = canonicalID
			}
			c.Offerings = append(c.Offerings, ModelOfferingV1{
				ID:               deploymentID + ":" + nativeID,
				CanonicalModelID: canonicalID,
				DeploymentID:     deploymentID,
				NativeModelID:    nativeID,
				Capabilities:     capabilitySetFromLegacy(entry),
				Pricing:          pricingFromLegacy(entry, generatedAt, legacy.Source),
			})
		}
	}
	c.Offerings = appendDerivedDeploymentOfferings(c.Offerings)
	c.OfferingTemplates = defaultOfferingTemplatesV1(generatedAt)
	return c
}

func ParseCatalogV1(data []byte) (*CatalogV1, error) {
	var envelope struct {
		SchemaVersion string `json:"schema_version"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, fmt.Errorf("catalog v1: decode envelope: %w", err)
	}
	if envelope.SchemaVersion == "" {
		var legacy ModelCatalog
		if err := json.Unmarshal(data, &legacy); err != nil {
			return nil, fmt.Errorf("catalog v1: decode legacy catalog: %w", err)
		}
		c := CatalogV1FromLegacy(legacy)
		return &c, ValidateCatalogV1(&c)
	}
	var c CatalogV1
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil {
		return nil, fmt.Errorf("catalog v1: decode: %w", err)
	}
	return &c, ValidateCatalogV1(&c)
}

func ValidateCatalogV1(c *CatalogV1) error {
	var problems []string
	add := func(format string, args ...any) { problems = append(problems, fmt.Sprintf(format, args...)) }
	if c == nil {
		return fmt.Errorf("catalog v1: nil catalog")
	}
	if c.SchemaVersion != CatalogV1SchemaVersion {
		add("schema_version must be %q", CatalogV1SchemaVersion)
	}
	if c.GeneratedAt.IsZero() {
		add("generated_at is required")
	}
	if c.StaleAfter.IsZero() {
		add("stale_after is required")
	} else if !c.GeneratedAt.IsZero() && !c.StaleAfter.After(c.GeneratedAt) {
		add("stale_after must be after generated_at")
	}
	if len(c.Providers) == 0 {
		add("providers is required")
	}
	if len(c.APIProtocols) == 0 {
		add("api_protocols is required")
	}
	if len(c.Deployments) == 0 {
		add("deployments is required")
	}
	if len(c.Models) == 0 {
		add("models is required")
	}
	if len(c.Offerings) == 0 {
		add("offerings is required")
	}
	for id, provider := range c.Providers {
		if id == "" || provider.ID != id || provider.Name == "" {
			add("provider %q must have matching non-empty id and name", id)
		}
	}
	for id, protocol := range c.APIProtocols {
		if id == "" || protocol.ID != id || protocol.Name == "" {
			add("api_protocol %q must have matching non-empty id and name", id)
		}
	}
	for id, deployment := range c.Deployments {
		if id == "" || deployment.ID != id || deployment.Name == "" {
			add("deployment %q must have matching non-empty id and name", id)
		}
		if c.Providers[deployment.ProviderID].ID == "" {
			add("deployment %q references unknown provider %q", id, deployment.ProviderID)
		}
		if c.APIProtocols[deployment.APIProtocolID].ID == "" {
			add("deployment %q references unknown api_protocol %q", id, deployment.APIProtocolID)
		}
		if deployment.AdapterConstructor == "" {
			add("deployment %q missing adapter_constructor", id)
		}
		if !validNativeModelIDSource(deployment.NativeModelIDSource) {
			add("deployment %q has invalid native_model_id_source %q", id, deployment.NativeModelIDSource)
		}
	}
	for id, model := range c.Models {
		if id == "" || model.ID != id || model.Name == "" {
			add("model %q must have matching non-empty id and name", id)
		}
		if !looksCanonicalModelID(id) {
			add("model %q is not an owner-qualified canonical model id", id)
		}
		if c.Providers[model.ProviderID].ID == "" {
			add("model %q references unknown provider %q", id, model.ProviderID)
		}
	}
	seenOfferings := map[string]bool{}
	for _, offering := range c.Offerings {
		if offering.ID == "" {
			add("offering missing id")
			continue
		}
		if seenOfferings[offering.ID] {
			add("offering %q is duplicated", offering.ID)
			continue
		}
		seenOfferings[offering.ID] = true
		deploymentID, nativeID, ok := SplitOfferingIDV1(offering.ID)
		if !ok || deploymentID != offering.DeploymentID || nativeID != offering.NativeModelID {
			add("offering %q must be deployment_id:native_model_id", offering.ID)
		}
		if c.Models[offering.CanonicalModelID].ID == "" {
			add("offering %q references unknown model %q", offering.ID, offering.CanonicalModelID)
		}
		if c.Deployments[offering.DeploymentID].ID == "" {
			add("offering %q references unknown deployment %q", offering.ID, offering.DeploymentID)
		}
		validatePricing(&problems, offering.ID, offering.Pricing)
		validateCapabilities(&problems, offering.ID, offering.Capabilities)
	}
	for _, tmpl := range c.OfferingTemplates {
		if tmpl.ID == "" {
			add("offering_template missing id")
			continue
		}
		if c.Models[tmpl.CanonicalModelID].ID == "" {
			add("offering_template %q references unknown model %q", tmpl.ID, tmpl.CanonicalModelID)
		}
		deployment := c.Deployments[tmpl.DeploymentID]
		if deployment.ID == "" {
			add("offering_template %q references unknown deployment %q", tmpl.ID, tmpl.DeploymentID)
		} else if !deployment.ModelMappingsRequired {
			add("offering_template %q references deployment %q that does not require mappings", tmpl.ID, tmpl.DeploymentID)
		}
		if !tmpl.MappingRequired || tmpl.NativeModelIDSource != NativeModelIDUserConfigured {
			add("offering_template %q must require user-configured mappings", tmpl.ID)
		}
		validatePricing(&problems, tmpl.ID, tmpl.Pricing)
		validateCapabilities(&problems, tmpl.ID, tmpl.Capabilities)
	}
	if len(problems) > 0 {
		sort.Strings(problems)
		return fmt.Errorf("catalog v1 validation failed: %s", strings.Join(problems, "; "))
	}
	return nil
}

func CompileCatalogV1(c *CatalogV1) (*CompiledCatalogV1, error) {
	if err := ValidateCatalogV1(c); err != nil {
		return nil, err
	}
	compiled := &CompiledCatalogV1{
		Catalog:                   c,
		ProvidersByID:             cloneMap(c.Providers),
		APIProtocolsByID:          cloneMap(c.APIProtocols),
		DeploymentsByID:           cloneMap(c.Deployments),
		ModelsByID:                cloneMap(c.Models),
		OfferingsByID:             map[string]ModelOfferingV1{},
		OfferingsByCanonicalModel: map[string][]ModelOfferingV1{},
		OfferingsByDeployment:     map[string][]ModelOfferingV1{},
		TemplatesByCanonicalModel: map[string][]ModelOfferingTemplateV1{},
	}
	if time.Now().UTC().After(c.StaleAfter) {
		compiled.Diagnostics = append(compiled.Diagnostics, CatalogDiagnosticV1{Code: "stale_catalog", Message: "catalog is stale"})
	}
	for _, offering := range c.Offerings {
		compiled.OfferingsByID[offering.ID] = offering
		compiled.OfferingsByCanonicalModel[offering.CanonicalModelID] = append(compiled.OfferingsByCanonicalModel[offering.CanonicalModelID], offering)
		compiled.OfferingsByDeployment[offering.DeploymentID] = append(compiled.OfferingsByDeployment[offering.DeploymentID], offering)
	}
	for _, tmpl := range c.OfferingTemplates {
		compiled.TemplatesByCanonicalModel[tmpl.CanonicalModelID] = append(compiled.TemplatesByCanonicalModel[tmpl.CanonicalModelID], tmpl)
	}
	return compiled, nil
}

func LoadCatalogV1(ctx context.Context, opts LoadCatalogV1Options) (*CompiledCatalogV1, error) {
	if opts.CachePath != "" {
		if data, err := os.ReadFile(opts.CachePath); err == nil {
			c, err := ParseCatalogV1(data)
			if err == nil {
				if compiled, compileErr := CompileCatalogV1(c); compileErr == nil {
					return compiled, nil
				}
			}
		}
	}
	embedded := DefaultCatalogV1()
	compiled, err := CompileCatalogV1(&embedded)
	if err != nil {
		return nil, err
	}
	if opts.RefreshRemote {
		remote, err := FetchRemoteCatalogV1(ctx, opts)
		if err == nil {
			if opts.CachePath != "" {
				_ = WriteCatalogV1Cache(opts.CachePath, remote)
			}
			return CompileCatalogV1(remote)
		}
		compiled.Diagnostics = append(compiled.Diagnostics, CatalogDiagnosticV1{Code: "remote_refresh_failed", Message: err.Error()})
	}
	return compiled, nil
}

func FetchRemoteCatalogV1(ctx context.Context, opts LoadCatalogV1Options) (*CatalogV1, error) {
	url := opts.RemoteURL
	if url == "" {
		url = DefaultCatalogV1URL
	}
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	client := opts.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: timeout}
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "eyrie-model-catalog/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("catalog v1: remote returned HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return ParseCatalogV1(body)
}

func WriteCatalogV1Cache(cachePath string, c *CatalogV1) error {
	if cachePath == "" {
		return nil
	}
	if err := ValidateCatalogV1(c); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(cachePath, append(data, '\n'), 0o600)
}

func (c *CompiledCatalogV1) CanonicalModelForAliasOrID(value string) (string, bool) {
	if c == nil || value == "" {
		return "", false
	}
	if c.ModelsByID[value].ID != "" {
		return value, true
	}
	if target := c.Catalog.Aliases[value]; c.ModelsByID[target].ID != "" {
		return target, true
	}
	for _, model := range c.ModelsByID {
		for _, alias := range model.Aliases {
			if alias == value {
				return model.ID, true
			}
		}
	}
	return "", false
}

func (c *CompiledCatalogV1) OfferingForDeployment(canonicalModelID, deploymentID string) (ModelOfferingV1, bool) {
	if c == nil {
		return ModelOfferingV1{}, false
	}
	for _, offering := range c.OfferingsByCanonicalModel[canonicalModelID] {
		if offering.DeploymentID == deploymentID {
			return offering, true
		}
	}
	return ModelOfferingV1{}, false
}

func SplitOfferingIDV1(id string) (deploymentID, nativeModelID string, ok bool) {
	left, right, found := strings.Cut(id, ":")
	return left, right, found && left != "" && right != ""
}

func defaultProvidersV1() map[string]ProviderV1 {
	return map[string]ProviderV1{
		"anthropic":  {ID: "anthropic", Name: "Anthropic"},
		"openai":     {ID: "openai", Name: "OpenAI"},
		"google":     {ID: "google", Name: "Google"},
		"xai":        {ID: "xai", Name: "xAI"},
		"openrouter": {ID: "openrouter", Name: "OpenRouter"},
		"z-ai":       {ID: "z-ai", Name: "Z.AI"},
		"ollama":     {ID: "ollama", Name: "Ollama"},
		"opencodego": {ID: "opencodego", Name: "OpenCode Go"},
	}
}

func defaultAPIProtocolsV1() map[string]APIProtocolV1 {
	return map[string]APIProtocolV1{
		"anthropic-messages":      {ID: "anthropic-messages", Name: "Anthropic Messages"},
		"openai-chat-completions": {ID: "openai-chat-completions", Name: "OpenAI Chat Completions"},
		"gemini-generate-content": {ID: "gemini-generate-content", Name: "Gemini generateContent"},
	}
}

func defaultDeploymentsV1() map[string]DeploymentV1 {
	return map[string]DeploymentV1{
		"anthropic-direct":  deployment("anthropic-direct", "Anthropic", "anthropic", "anthropic-messages", "anthropic", NativeModelIDCatalogKnown),
		"anthropic-bedrock": deployment("anthropic-bedrock", "Anthropic on Bedrock", "anthropic", "anthropic-messages", "anthropic-bedrock", NativeModelIDCatalogKnown),
		"anthropic-vertex":  deployment("anthropic-vertex", "Anthropic on Vertex", "anthropic", "anthropic-messages", "anthropic-vertex", NativeModelIDCatalogKnown),
		"openai-direct":     deployment("openai-direct", "OpenAI", "openai", "openai-chat-completions", "openai", NativeModelIDCatalogKnown),
		"openai-azure":      azureDeployment(),
		"gemini-direct":     deployment("gemini-direct", "Gemini", "google", "gemini-generate-content", "gemini", NativeModelIDCatalogKnown),
		"gemini-vertex":     deployment("gemini-vertex", "Gemini on Vertex", "google", "gemini-generate-content", "gemini-vertex", NativeModelIDCatalogKnown),
		"grok-direct":       deployment("grok-direct", "Grok", "xai", "openai-chat-completions", "grok", NativeModelIDCatalogKnown),
		"openrouter":        deployment("openrouter", "OpenRouter", "openrouter", "openai-chat-completions", "openrouter", NativeModelIDDiscovered),
		"canopywave":        deployment("canopywave", "CanopyWave", "z-ai", "openai-chat-completions", "canopywave", NativeModelIDCatalogKnown),
		"ollama-local":      localDeployment(),
		"opencodego":        deployment("opencodego", "OpenCode Go", "opencodego", "openai-chat-completions", "opencodego", NativeModelIDCatalogKnown),
	}
}

func deployment(id, name, providerID, protocolID, adapter string, source NativeModelIDSource) DeploymentV1 {
	return DeploymentV1{ID: id, Name: name, ProviderID: providerID, APIProtocolID: protocolID, AdapterConstructor: adapter, NativeModelIDSource: source}
}

func azureDeployment() DeploymentV1 {
	d := deployment("openai-azure", "Azure OpenAI", "openai", "openai-chat-completions", "openai-azure", NativeModelIDUserConfigured)
	d.ModelMappingsRequired = true
	return d
}

func localDeployment() DeploymentV1 {
	d := deployment("ollama-local", "Ollama local", "ollama", "openai-chat-completions", "ollama", NativeModelIDDiscovered)
	d.Local = true
	return d
}

func defaultOfferingTemplatesV1(generatedAt time.Time) []ModelOfferingTemplateV1 {
	var out []ModelOfferingTemplateV1
	for _, model := range OpenAIModels {
		canonical := canonicalModelID("openai", model.ID)
		out = append(out, ModelOfferingTemplateV1{
			ID:                  "openai-azure:" + canonical,
			CanonicalModelID:    canonical,
			DeploymentID:        "openai-azure",
			NativeModelIDSource: NativeModelIDUserConfigured,
			MappingRequired:     true,
			Capabilities:        capabilitySetFromLegacy(model),
			Pricing:             pricingFromLegacy(model, generatedAt, "embedded"),
		})
	}
	return out
}

func appendDerivedDeploymentOfferings(offerings []ModelOfferingV1) []ModelOfferingV1 {
	seen := make(map[string]bool, len(offerings))
	for _, offering := range offerings {
		seen[offering.ID] = true
	}
	addCopy := func(source ModelOfferingV1, deploymentID string) {
		copied := source
		copied.DeploymentID = deploymentID
		copied.ID = deploymentID + ":" + source.NativeModelID
		if !seen[copied.ID] {
			seen[copied.ID] = true
			offerings = append(offerings, copied)
		}
	}
	for _, offering := range append([]ModelOfferingV1(nil), offerings...) {
		switch offering.DeploymentID {
		case "anthropic-direct":
			addCopy(offering, "anthropic-bedrock")
			addCopy(offering, "anthropic-vertex")
		case "gemini-direct":
			addCopy(offering, "gemini-vertex")
		}
	}
	return offerings
}

func legacyDeploymentAndOwner(provider string) (deploymentID, ownerProviderID string) {
	switch provider {
	case "anthropic":
		return "anthropic-direct", "anthropic"
	case "openai":
		return "openai-direct", "openai"
	case "grok":
		return "grok-direct", "xai"
	case "gemini":
		return "gemini-direct", "google"
	case "openrouter":
		return "openrouter", "openrouter"
	case "canopywave":
		return "canopywave", "z-ai"
	case "ollama":
		return "ollama-local", "ollama"
	case "opencodego":
		return "opencodego", "opencodego"
	default:
		return "", ""
	}
}

func canonicalModelID(ownerProviderID, nativeID string) string {
	if strings.Contains(nativeID, "/") {
		owner, _, _ := strings.Cut(nativeID, "/")
		if owner != "" && ownerProviderID == canonicalProviderID(owner) {
			return nativeID
		}
	}
	if ownerProviderID == "z-ai" && strings.HasPrefix(nativeID, "zai/") {
		return "z-ai/" + strings.TrimPrefix(nativeID, "zai/")
	}
	return ownerProviderID + "/" + nativeID
}

func canonicalProviderID(providerID string) string {
	switch providerID {
	case "gemini":
		return "google"
	case "grok":
		return "xai"
	case "zai":
		return "z-ai"
	default:
		return providerID
	}
}

func capabilitySetFromLegacy(entry ModelCatalogEntry) CapabilitySetV1 {
	set := CapabilitySetV1{ServerTools: map[string]CapabilityState{}}
	for _, tool := range entry.ServerTools {
		if tool != "" {
			set.ServerTools[tool] = CapabilitySupported
		}
	}
	if len(set.ServerTools) == 0 {
		set.ServerTools = nil
	}
	return set
}

func pricingFromLegacy(entry ModelCatalogEntry, effectiveAt time.Time, source string) PricingV1 {
	pricing := PricingV1{
		Status:      PricingKnown,
		Currency:    "USD",
		EffectiveAt: effectiveAt,
		RatesPer1M:  map[string]float64{"input_tokens": entry.InputPricePer1M, "output_tokens": entry.OutputPricePer1M},
		Source:      source,
	}
	if entry.InputPricePer1M == 0 && entry.OutputPricePer1M == 0 {
		pricing.Status = PricingUnknown
		pricing.RatesPer1M = nil
		if strings.Contains(entry.ID, ":free") {
			pricing.Status = PricingFree
			pricing.RatesPer1M = map[string]float64{"input_tokens": 0, "output_tokens": 0}
		}
	}
	return pricing
}

func uniqueNonEmpty(values ...string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func looksCanonicalModelID(value string) bool {
	owner, model, ok := strings.Cut(value, "/")
	return ok && owner != "" && model != "" && !strings.ContainsAny(value, " \t\r\n")
}

func validNativeModelIDSource(source NativeModelIDSource) bool {
	switch source {
	case NativeModelIDCatalogKnown, NativeModelIDDiscovered, NativeModelIDUserConfigured, NativeModelIDCatalogOrUser:
		return true
	default:
		return false
	}
}

func validatePricing(problems *[]string, id string, pricing PricingV1) {
	switch pricing.Status {
	case PricingKnown, PricingPartial:
		if pricing.Currency == "" || len(pricing.RatesPer1M) == 0 {
			*problems = append(*problems, fmt.Sprintf("%s pricing is missing currency or rates", id))
		}
	case PricingUnknown:
		if len(pricing.RatesPer1M) > 0 {
			*problems = append(*problems, fmt.Sprintf("%s unknown pricing must not include rates", id))
		}
	case PricingFree:
		if pricing.Currency == "" {
			*problems = append(*problems, fmt.Sprintf("%s free pricing missing currency", id))
		}
	default:
		*problems = append(*problems, fmt.Sprintf("%s invalid pricing status %q", id, pricing.Status))
	}
	for dim, rate := range pricing.RatesPer1M {
		if dim == "" || rate < 0 {
			*problems = append(*problems, fmt.Sprintf("%s invalid pricing dimension %q", id, dim))
		}
	}
}

func validateCapabilities(problems *[]string, id string, capabilities CapabilitySetV1) {
	valid := func(state CapabilityState) bool {
		return state == "" || state == CapabilitySupported || state == CapabilityUnsupported || state == CapabilityUnknown
	}
	if !valid(capabilities.FunctionCalling) {
		*problems = append(*problems, fmt.Sprintf("%s invalid function_calling capability", id))
	}
	if !valid(capabilities.ExplicitThinkingBudget) {
		*problems = append(*problems, fmt.Sprintf("%s invalid explicit_thinking_budget capability", id))
	}
	for tool, state := range capabilities.ServerTools {
		if tool == "" || !valid(state) {
			*problems = append(*problems, fmt.Sprintf("%s invalid server tool capability", id))
		}
	}
}

func cloneMap[T any](in map[string]T) map[string]T {
	out := make(map[string]T, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
