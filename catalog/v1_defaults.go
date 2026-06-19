package catalog

import (
	"fmt"
	"strings"
	"time"
)

// Default catalog construction, legacy ModelCatalog conversion, and pricing/
// capability sanitization+validation helpers. Split out of v1.go for clarity.
func defaultProvidersV1() map[string]ProviderV1 {
	return map[string]ProviderV1{
		"anthropic":              {ID: "anthropic", Name: "Anthropic"},
		"openai":                 {ID: "openai", Name: "OpenAI"},
		"google":                 {ID: "google", Name: "Google"},
		"xai":                    {ID: "xai", Name: "xAI"},
		"openrouter":             {ID: "openrouter", Name: "OpenRouter"},
		"canopywave":             {ID: "canopywave", Name: "CanopyWave"},
		"zai_payg":               {ID: "zai_payg", Name: "Z.AI Pay-as-you-go"},
		"zai_coding":             {ID: "zai_coding", Name: "Z.AI Coding Plan"},
		"ollama":                 {ID: "ollama", Name: "Ollama"},
		"opencodego":             {ID: "opencodego", Name: "OpenCode Go"},
		"moonshotai":             {ID: "moonshotai", Name: "Moonshot AI"},
		"kimi":                   {ID: "kimi", Name: "Kimi (Moonshot)"},
		"xiaomi_mimo_payg":       {ID: "xiaomi_mimo_payg", Name: "Xiaomi MiMo (Pay-as-you-go)"},
		"xiaomi_mimo_token_plan": {ID: "xiaomi_mimo_token_plan", Name: "Xiaomi MiMo (Token Plan)"},
		"deepseek":               {ID: "deepseek", Name: "DeepSeek"},
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
		"anthropic-direct":              deployment("anthropic-direct", "Anthropic", "anthropic", "anthropic-messages", "anthropic", NativeModelIDCatalogKnown),
		"anthropic-bedrock":             deployment("anthropic-bedrock", "Anthropic on Bedrock", "anthropic", "anthropic-messages", "anthropic-bedrock", NativeModelIDCatalogKnown),
		"anthropic-vertex":              deployment("anthropic-vertex", "Anthropic on Vertex", "anthropic", "anthropic-messages", "anthropic-vertex", NativeModelIDCatalogKnown),
		"openai-direct":                 deployment("openai-direct", "OpenAI", "openai", "openai-chat-completions", "openai", NativeModelIDCatalogKnown),
		"openai-azure":                  azureDeployment(),
		"gemini-direct":                 deployment("gemini-direct", "Gemini", "google", "gemini-generate-content", "gemini", NativeModelIDCatalogKnown),
		"gemini-vertex":                 deployment("gemini-vertex", "Gemini on Vertex", "google", "gemini-generate-content", "gemini-vertex", NativeModelIDCatalogKnown),
		"grok-direct":                   deployment("grok-direct", "Grok", "xai", "openai-chat-completions", "grok", NativeModelIDCatalogKnown),
		"openrouter":                    deployment("openrouter", "OpenRouter", "openrouter", "openai-chat-completions", "openrouter", NativeModelIDDiscovered),
		"zai_payg-direct":               deployment("zai_payg-direct", "Z.AI Pay-as-you-go", "zai_payg", "openai-chat-completions", "zai_payg", NativeModelIDCatalogKnown),
		"zai_coding-direct":             deployment("zai_coding-direct", "Z.AI Coding Plan", "zai_coding", "openai-chat-completions", "zai_coding", NativeModelIDCatalogKnown),
		"canopywave":                    deployment("canopywave", "CanopyWave", "canopywave", "openai-chat-completions", "canopywave", NativeModelIDDiscovered),
		"ollama-local":                  localDeployment(),
		"opencodego":                    deployment("opencodego", "OpenCode Go", "opencodego", "openai-chat-completions", "opencodego", NativeModelIDDiscovered),
		"kimi-direct":                   deployment("kimi-direct", "Kimi (Moonshot)", "kimi", "openai-chat-completions", "kimi", NativeModelIDDiscovered),
		"xiaomi_mimo_payg-direct":       deployment("xiaomi_mimo_payg-direct", "Xiaomi MiMo Pay-as-you-go", "xiaomi_mimo_payg", "openai-chat-completions", "xiaomi_mimo", NativeModelIDDiscovered),
		"xiaomi_mimo_token_plan-direct": deployment("xiaomi_mimo_token_plan-direct", "Xiaomi MiMo Token Plan", "xiaomi_mimo_token_plan", "openai-chat-completions", "xiaomi_mimo", NativeModelIDDiscovered),
		"deepseek-direct":               deployment("deepseek-direct", "DeepSeek", "deepseek", "openai-chat-completions", "deepseek", NativeModelIDCatalogKnown),
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
	for _, model := range testOpenAIModels {
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
	case "azure":
		return "openai-azure", "openai"
	case "grok":
		return "grok-direct", "xai"
	case "gemini":
		return "gemini-direct", "google"
	case "bedrock":
		return "anthropic-bedrock", "anthropic"
	case "vertex":
		return "gemini-vertex", "google"
	case "openrouter":
		return "openrouter", "openrouter"
	case "zai_payg":
		return "zai_payg-direct", "zai_payg"
	case "zai_coding":
		return "zai_coding-direct", "zai_coding"
	case "canopywave":
		return "canopywave", "canopywave"
	case "ollama":
		return "ollama-local", "ollama"
	case "opencodego":
		return "opencodego", "opencodego"
	case "kimi", "moonshotai":
		return "kimi-direct", "kimi"
	case "xiaomi_mimo", "xiaomi_mimo_payg":
		return "xiaomi_mimo_payg-direct", "xiaomi_mimo_payg"
	case "xiaomi_mimo_token_plan":
		return "xiaomi_mimo_token_plan-direct", "xiaomi_mimo_token_plan"
	case "deepseek":
		return "deepseek-direct", "deepseek"
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
	if ownerProviderID == "zai_payg" && strings.HasPrefix(nativeID, "zai/") {
		return "zai_payg/" + strings.TrimPrefix(nativeID, "zai/")
	}
	return ownerProviderID + "/" + nativeID
}

// CanonicalProviderID normalizes legacy provider aliases (e.g. gemini -> google).
func CanonicalProviderID(providerID string) string {
	return canonicalProviderID(providerID)
}

func canonicalProviderID(providerID string) string {
	switch providerID {
	case "gemini":
		return "google"
	case "grok":
		return "xai"
	// No legacy aliases — zai_payg and zai_coding are the only valid IDs.
	case "moonshotai":
		return "moonshotai"
	case "xiaomi-mimo", "xiaomi_mimo", "xiaomi-mimo-payg":
		return "xiaomi_mimo_payg"
	case "xiaomi-mimo-token-plan":
		return "xiaomi_mimo_token_plan"
	default:
		return providerID
	}
}

func capabilitySetFromLegacy(entry ModelCatalogEntry) CapabilitySetV1 {
	set := CapabilitySetV1{
		ServerTools:     map[string]CapabilityState{},
		MaxInputTokens:  entry.ContextWindow,
		MaxOutputTokens: entry.MaxOutput,
	}
	for _, tool := range entry.ServerTools {
		if tool != "" {
			set.ServerTools[tool] = CapabilitySupported
		}
	}
	if len(set.ServerTools) == 0 {
		set.ServerTools = nil
	}
	for _, feat := range entry.ServerTools {
		switch strings.ToLower(strings.TrimSpace(feat)) {
		case "function-calling", "tools":
			set.FunctionCalling = CapabilitySupported
		case "thinking:enabled":
			set.ExplicitThinkingBudget = CapabilitySupported
			set.ThinkingTypes = append(set.ThinkingTypes, "enabled")
		case "thinking:adaptive":
			set.AdaptiveThinking = CapabilitySupported
			set.ThinkingTypes = append(set.ThinkingTypes, "adaptive")
		case "effort":
			set.Effort = CapabilitySupported
		case "structured_output":
			set.StructuredOutput = CapabilitySupported
		case "code_execution":
			set.CodeExecution = CapabilitySupported
		case "citations":
			set.Citations = CapabilitySupported
		case "pdf_input":
			set.PDFInput = CapabilitySupported
		case "image_input":
			set.ImageInput = CapabilitySupported
		}
	}
	// Parse effort levels from features (format: "effort:low,medium,high")
	for _, feat := range entry.ServerTools {
		if strings.HasPrefix(strings.ToLower(feat), "effort:") {
			levels := strings.TrimPrefix(strings.ToLower(feat), "effort:")
			set.EffortLevels = strings.Split(levels, ",")
		}
	}
	return set
}

func pricingFromLegacy(entry ModelCatalogEntry, effectiveAt time.Time, source string) PricingV1 {
	in := entry.InputPricePer1M
	out := entry.OutputPricePer1M
	if in < 0 || out < 0 {
		return PricingV1{
			Status:      PricingUnknown,
			Currency:    "USD",
			EffectiveAt: effectiveAt,
			Source:      source,
		}
	}
	pricing := PricingV1{
		Status:      PricingKnown,
		Currency:    "USD",
		EffectiveAt: effectiveAt,
		RatesPer1M:  map[string]float64{"input_tokens": in, "output_tokens": out},
		Source:      source,
	}
	if in == 0 && out == 0 {
		pricing.Status = PricingUnknown
		pricing.RatesPer1M = nil
		if strings.Contains(entry.ID, ":free") {
			pricing.Status = PricingFree
			pricing.RatesPer1M = map[string]float64{"input_tokens": 0, "output_tokens": 0}
		}
	}
	return pricing
}

// SanitizeCatalogV1Pricing drops invalid rate dimensions (e.g. negative OpenRouter prices).
func SanitizeCatalogV1Pricing(c *CatalogV1) {
	if c == nil {
		return
	}
	for i := range c.Offerings {
		c.Offerings[i].Pricing = sanitizePricingV1(c.Offerings[i].Pricing)
	}
	for i := range c.OfferingTemplates {
		c.OfferingTemplates[i].Pricing = sanitizePricingV1(c.OfferingTemplates[i].Pricing)
	}
}

func sanitizePricingV1(p PricingV1) PricingV1 {
	if len(p.RatesPer1M) == 0 {
		return p
	}
	clean := make(map[string]float64, len(p.RatesPer1M))
	for dim, rate := range p.RatesPer1M {
		if dim == "" || rate < 0 {
			continue
		}
		clean[dim] = rate
	}
	if len(clean) == 0 {
		p.Status = PricingUnknown
		p.RatesPer1M = nil
		return p
	}
	p.RatesPer1M = clean
	if p.Status == PricingKnown && (p.Currency == "" || len(p.RatesPer1M) == 0) {
		p.Status = PricingUnknown
		p.RatesPer1M = nil
	}
	return p
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
