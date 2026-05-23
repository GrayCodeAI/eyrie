package client

// OpenAICompatConfig holds provider-specific compatibility flags
// that control how API requests are constructed for each provider.
type OpenAICompatConfig struct {
	SupportsStore                    bool   `json:"supports_store,omitempty"`
	SupportsDeveloperRole            bool   `json:"supports_developer_role,omitempty"`
	SupportsReasoningEffort          bool   `json:"supports_reasoning_effort,omitempty"`
	SupportsUsageInStreaming         bool   `json:"supports_usage_in_streaming,omitempty"`
	SupportsStrictMode               bool   `json:"supports_strict_mode,omitempty"`
	MaxTokensField                   string `json:"max_tokens_field,omitempty"` // "max_tokens" or "max_completion_tokens"
	RequiresToolResultName           bool   `json:"requires_tool_result_name,omitempty"`
	RequiresAssistantAfterToolResult bool   `json:"requires_assistant_after_tool_result,omitempty"`
	RequiresThinkingAsText           bool   `json:"requires_thinking_as_text,omitempty"`
	ThinkingFormat                   string `json:"thinking_format,omitempty"` // "openai", "zai", "qwen", "openrouter"
}

// Per-provider compat configs.
var (
	OpenAICompat = OpenAICompatConfig{
		SupportsStore: true, SupportsDeveloperRole: true,
		SupportsReasoningEffort: true, SupportsUsageInStreaming: true,
		MaxTokensField: "max_completion_tokens",
	}
	GrokCompat = OpenAICompatConfig{
		MaxTokensField: "max_tokens",
	}
	OpenRouterCompat = OpenAICompatConfig{
		ThinkingFormat: "openrouter", MaxTokensField: "max_tokens",
		SupportsUsageInStreaming: true,
	}
	GeminiCompat = OpenAICompatConfig{
		MaxTokensField: "max_tokens", SupportsUsageInStreaming: true,
	}
	ZAICompat = OpenAICompatConfig{
		ThinkingFormat: "zai", MaxTokensField: "max_tokens",
		SupportsUsageInStreaming: true,
	}
	CanopyWaveCompat = OpenAICompatConfig{
		MaxTokensField: "max_tokens",
	}
	OllamaCompat = OpenAICompatConfig{
		MaxTokensField: "max_tokens",
	}
	OpenCodeGoCompat = OpenAICompatConfig{
		MaxTokensField: "max_tokens",
	}
	KimiCompat = OpenAICompatConfig{
		MaxTokensField: "max_tokens",
	}
	XiaomiCompat = OpenAICompatConfig{
		MaxTokensField: "max_tokens",
	}
	AzureCompat = OpenAICompatConfig{
		MaxTokensField: "max_tokens",
	}
	BedrockCompat = OpenAICompatConfig{
		MaxTokensField: "max_tokens",
	}
	VertexCompat = OpenAICompatConfig{
		MaxTokensField: "max_tokens",
	}
)

func init() {
	// Attach compat configs to provider registry.
	// Acquire dynamicMu for consistency with the runtime lock protocol,
	// even though init() is single-threaded.
	dynamicMu.Lock()
	defer dynamicMu.Unlock()

	if p, ok := OpenAICompatibleProviders["grok"]; ok {
		p.Compat = &GrokCompat
		OpenAICompatibleProviders["grok"] = p
	}
	if p, ok := OpenAICompatibleProviders["openrouter"]; ok {
		p.Compat = &OpenRouterCompat
		OpenAICompatibleProviders["openrouter"] = p
	}
	if p, ok := OpenAICompatibleProviders["gemini"]; ok {
		p.Compat = &GeminiCompat
		OpenAICompatibleProviders["gemini"] = p
	}
	if p, ok := OpenAICompatibleProviders["z-ai"]; ok {
		p.Compat = &ZAICompat
		OpenAICompatibleProviders["z-ai"] = p
	}
	if p, ok := OpenAICompatibleProviders["canopywave"]; ok {
		p.Compat = &CanopyWaveCompat
		OpenAICompatibleProviders["canopywave"] = p
	}
	if p, ok := OpenAICompatibleProviders["ollama"]; ok {
		p.Compat = &OllamaCompat
		OpenAICompatibleProviders["ollama"] = p
	}
	if p, ok := OpenAICompatibleProviders["opencodego"]; ok {
		p.Compat = &OpenCodeGoCompat
		OpenAICompatibleProviders["opencodego"] = p
	}
	if p, ok := OpenAICompatibleProviders["kimi"]; ok {
		p.Compat = &KimiCompat
		OpenAICompatibleProviders["kimi"] = p
	}
	if p, ok := OpenAICompatibleProviders["xiaomi"]; ok {
		p.Compat = &XiaomiCompat
		OpenAICompatibleProviders["xiaomi"] = p
	}
	if p, ok := CoreProviders["openai"]; ok {
		p.Compat = &OpenAICompat
		CoreProviders["openai"] = p
	}
	if p, ok := CoreProviders["azure"]; ok {
		p.Compat = &AzureCompat
		CoreProviders["azure"] = p
	}
	if p, ok := CoreProviders["bedrock"]; ok {
		p.Compat = &BedrockCompat
		CoreProviders["bedrock"] = p
	}
	if p, ok := CoreProviders["vertex"]; ok {
		p.Compat = &VertexCompat
		CoreProviders["vertex"] = p
	}
}
