package registry

// ModelStrategy defines how models are discovered for a provider.
type ModelStrategy string

const (
	StrategyRemoteCatalog   ModelStrategy = "remote_catalog"
	StrategyRemoteThenLive  ModelStrategy = "remote_then_live"
	StrategyLiveOnly        ModelStrategy = "live_only"
)

// ProbeKind identifies HTTP credential validation.
type ProbeKind string

const (
	ProbeAnthropic      ProbeKind = "probe_anthropic"
	ProbeOpenAIModels   ProbeKind = "probe_openai_models"
	ProbeGemini         ProbeKind = "probe_gemini"
	ProbeOllama         ProbeKind = "probe_ollama"
	ProbeNone           ProbeKind = "probe_none"
)

// ProviderSpec is the single source of truth for setup providers.
type ProviderSpec struct {
	ProviderID      string
	DisplayName     string
	DeploymentID    string
	SortOrder       int
	RequiresKey     bool
	CredentialEnv   string
	KeyPrefixes     []string
	BaseURLEnv      []string
	ProbeKind       ProbeKind
	ProbeBaseURL    string
	ModelStrategy   ModelStrategy
	PreferLiveMerge bool
	LiveFetcherKey  string // key in catalog/live registry
	LiveCatalogKey  string // legacy provider key in ModelCatalog.Providers map
	APIProtocolID   string
	AdapterID       string
}

// EnvFallback describes one deployment env_fallback row.
type EnvFallback struct {
	Field string
	Env   []string
}
