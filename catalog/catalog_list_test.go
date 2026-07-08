package catalog

import (
	"encoding/json"
	"testing"
)

// --- ownerFromLiveMetadata tests ---

func TestOwnerFromLiveMetadata(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		raw  json.RawMessage
		want string
	}{
		{"nil", nil, ""},
		{"empty", json.RawMessage(`{}`), ""},
		{"with_owner", json.RawMessage(`{"owned_by":"openai"}`), "openai"},
		{"whitespace_owner", json.RawMessage(`{"owned_by":"  anthropic  "}`), "anthropic"},
		{"invalid_json", json.RawMessage(`not json`), ""},
		{"empty_owner", json.RawMessage(`{"owned_by":""}`), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ownerFromLiveMetadata(tt.raw)
			if got != tt.want {
				t.Errorf("ownerFromLiveMetadata(%s) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

// --- ownerFromModelID tests ---

func TestOwnerFromModelID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  string
	}{
		{"anthropic/claude-sonnet-4-6", "anthropic"},
		{"openai/gpt-4o", "openai"},
		{"gpt-4o", ""},
		{"", ""},
		{"  ", ""},
		{"owner/model/extra", "owner"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ownerFromModelID(tt.input)
			if got != tt.want {
				t.Errorf("ownerFromModelID(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// --- descriptionFromLiveMetadata tests ---

func TestDescriptionFromLiveMetadata(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		raw  json.RawMessage
		want string
	}{
		{"nil", nil, ""},
		{"empty_object", json.RawMessage(`{}`), ""},
		{"with_description", json.RawMessage(`{"description":"A helpful model"}`), "A helpful model"},
		{"whitespace_description", json.RawMessage(`{"description":"  padded  "}`), "padded"},
		{"empty_description", json.RawMessage(`{"description":""}`), ""},
		{"invalid_json", json.RawMessage(`not json`), ""},
		{"no_description_field", json.RawMessage(`{"other":"value"}`), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := descriptionFromLiveMetadata(tt.raw)
			if got != tt.want {
				t.Errorf("descriptionFromLiveMetadata(%s) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

// --- serverToolsFromOffering tests ---

func TestServerToolsFromOffering(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		offering ModelOffering
		want     []string
	}{
		{
			name:     "nil_server_tools",
			offering: ModelOffering{},
			want:     nil,
		},
		{
			name: "supported_tool",
			offering: ModelOffering{
				Capabilities: CapabilitySet{
					ServerTools: map[string]CapabilityState{"web_search": CapabilitySupported},
				},
			},
			want: []string{"web_search"},
		},
		{
			name: "unsupported_tool_filtered",
			offering: ModelOffering{
				Capabilities: CapabilitySet{
					ServerTools: map[string]CapabilityState{"web_search": CapabilityUnsupported},
				},
			},
			want: nil,
		},
		{
			name: "mixed_tools",
			offering: ModelOffering{
				Capabilities: CapabilitySet{
					ServerTools: map[string]CapabilityState{
						"web_search":  CapabilitySupported,
						"code_interp": CapabilityUnsupported,
						"retrieval":   CapabilitySupported,
					},
				},
			},
			want: []string{"retrieval", "web_search"},
		},
		{
			name: "empty_tool_name_filtered",
			offering: ModelOffering{
				Capabilities: CapabilitySet{
					ServerTools: map[string]CapabilityState{
						"":           CapabilitySupported,
						"web_search": CapabilitySupported,
					},
				},
			},
			want: []string{"web_search"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := serverToolsFromOffering(tt.offering)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i, v := range got {
				if v != tt.want[i] {
					t.Errorf("got[%d] = %q, want %q", i, v, tt.want[i])
				}
			}
		})
	}
}

// --- modelEntryFromOffering tests ---

func TestModelEntryFromOffering(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		model        Model
		offering     ModelOffering
		wantID       string
		wantContext  int
		wantMaxOut   int
		wantInPrice  float64
		wantOutPrice float64
	}{
		{
			name:  "uses_native_model_id",
			model: Model{ID: "anthropic/claude-sonnet-4-6", Name: "Sonnet 4.6", ContextWindow: 200000, MaxOutput: 32000},
			offering: ModelOffering{
				NativeModelID: "claude-sonnet-4-6",
				Pricing:       Pricing{RatesPer1M: map[string]float64{"input_tokens": 3, "output_tokens": 15}},
			},
			wantID: "claude-sonnet-4-6", wantContext: 200000, wantMaxOut: 32000, wantInPrice: 3, wantOutPrice: 15,
		},
		{
			name:  "empty_native_falls_back_to_model_id",
			model: Model{ID: "openai/gpt-4o", Name: "GPT-4o"},
			offering: ModelOffering{
				NativeModelID: "",
				Pricing:       Pricing{Status: PricingUnknown},
			},
			wantID: "openai/gpt-4o",
		},
		{
			name:  "nil_rates_zeroes_prices",
			model: Model{ID: "x/model", Name: "Model"},
			offering: ModelOffering{
				NativeModelID: "model",
				Pricing:       Pricing{Status: PricingUnknown},
			},
			wantID: "model", wantInPrice: 0, wantOutPrice: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := modelEntryFromOffering(tt.model, tt.offering)
			if got.ID != tt.wantID {
				t.Errorf("ID = %q, want %q", got.ID, tt.wantID)
			}
			if got.ContextWindow != tt.wantContext {
				t.Errorf("ContextWindow = %d, want %d", got.ContextWindow, tt.wantContext)
			}
			if got.MaxOutput != tt.wantMaxOut {
				t.Errorf("MaxOutput = %d, want %d", got.MaxOutput, tt.wantMaxOut)
			}
			if got.InputPricePer1M != tt.wantInPrice {
				t.Errorf("InputPricePer1M = %f, want %f", got.InputPricePer1M, tt.wantInPrice)
			}
			if got.OutputPricePer1M != tt.wantOutPrice {
				t.Errorf("OutputPricePer1M = %f, want %f", got.OutputPricePer1M, tt.wantOutPrice)
			}
		})
	}
}

// --- ModelEntriesForProvider additional tests ---

func TestModelEntriesForProvider_NilCompiled(t *testing.T) {
	t.Parallel()
	entries := ModelEntriesForProvider(nil, "anthropic")
	if entries != nil {
		t.Fatalf("expected nil, got %v", entries)
	}
}

func TestModelEntriesForProvider_EmptyProvider(t *testing.T) {
	t.Parallel()
	compiled := &CompiledCatalog{
		ModelsByID: map[string]Model{
			"anthropic/claude-sonnet-4-6": {ID: "anthropic/claude-sonnet-4-6", Name: "Sonnet", ProviderID: "anthropic"},
		},
	}
	entries := ModelEntriesForProvider(compiled, "")
	if entries != nil {
		t.Fatalf("expected nil for empty provider, got %v", entries)
	}
}

func TestModelEntriesForProvider_DeduplicatesByNativeID(t *testing.T) {
	t.Parallel()
	compiled := &CompiledCatalog{
		ModelsByID: map[string]Model{
			"anthropic/claude-sonnet-4-6": {ID: "anthropic/claude-sonnet-4-6", Name: "Sonnet", ProviderID: "anthropic"},
		},
		OfferingsByDeployment: map[string][]ModelOffering{
			"anthropic-direct": {
				{CanonicalModelID: "anthropic/claude-sonnet-4-6", DeploymentID: "anthropic-direct", NativeModelID: "claude-sonnet-4-6"},
			},
		},
	}
	entries := ModelEntriesForProvider(compiled, "anthropic")
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry after dedup, got %d", len(entries))
	}
}

// --- DiscoveryEnvKeysFromCatalog tests ---

func TestDiscoveryEnvKeysFromCatalog(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		compiled *CompiledCatalog
		wantNil  bool
	}{
		{
			name:     "nil_compiled",
			compiled: nil,
			wantNil:  true,
		},
		{
			name:     "nil_catalog",
			compiled: &CompiledCatalog{},
			wantNil:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DiscoveryEnvKeysFromCatalog(tt.compiled)
			if tt.wantNil && got != nil {
				t.Fatalf("expected nil, got %v", got)
			}
		})
	}
}

func TestDiscoveryEnvKeysFromCatalog_ReturnsUniqueKeys(t *testing.T) {
	t.Parallel()
	c := SeedCatalog()
	compiled, err := CompileCatalog(&c)
	if err != nil {
		t.Fatal(err)
	}
	keys := DiscoveryEnvKeysFromCatalog(compiled)
	if len(keys) == 0 {
		t.Fatal("expected env keys from compiled catalog")
	}
	seen := map[string]bool{}
	for _, k := range keys {
		if k == "" {
			t.Error("empty key in result")
		}
		if seen[k] {
			t.Errorf("duplicate key %q", k)
		}
		seen[k] = true
	}
}

// --- APIKeyEnvsForProvider tests ---

func TestAPIKeyEnvsForProvider(t *testing.T) {
	t.Parallel()
	c := SeedCatalog()
	compiled, err := CompileCatalog(&c)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		provider string
		wantMin  int
	}{
		{"anthropic", 1},
		{"openai", 1},
		{"ollama", 0}, // local, may have optional key
		{"nonexistent", 0},
	}
	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			got := APIKeyEnvsForProvider(compiled, tt.provider)
			if len(got) < tt.wantMin {
				t.Errorf("APIKeyEnvsForProvider(%q) = %v, want at least %d", tt.provider, got, tt.wantMin)
			}
		})
	}
}

func TestAPIKeyEnvsForProvider_NilCompiled(t *testing.T) {
	t.Parallel()
	got := APIKeyEnvsForProvider(nil, "anthropic")
	if got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

// --- PrimaryAPIKeyEnvForDeployment tests ---

func TestPrimaryAPIKeyEnvForDeployment(t *testing.T) {
	t.Parallel()
	tests := []struct {
		deploymentID string
		wantEmpty    bool
	}{
		{"anthropic-direct", false},
		{"openai-direct", false},
		{"nonexistent", true},
	}
	for _, tt := range tests {
		t.Run(tt.deploymentID, func(t *testing.T) {
			got := PrimaryAPIKeyEnvForDeployment(nil, tt.deploymentID)
			if tt.wantEmpty && got != "" {
				t.Errorf("expected empty, got %q", got)
			}
			if !tt.wantEmpty && got == "" {
				t.Errorf("expected non-empty for %q", tt.deploymentID)
			}
		})
	}
}

func TestPrimaryAPIKeyEnvForDeployment_WithCompiled(t *testing.T) {
	t.Parallel()
	c := SeedCatalog()
	compiled, err := CompileCatalog(&c)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		deploymentID string
		wantEmpty    bool
	}{
		{"anthropic-direct", false},
		{"openrouter", false},
		{"nonexistent-deployment", true},
	}
	for _, tt := range tests {
		t.Run(tt.deploymentID, func(t *testing.T) {
			got := PrimaryAPIKeyEnvForDeployment(compiled, tt.deploymentID)
			if tt.wantEmpty && got != "" {
				t.Errorf("expected empty, got %q", got)
			}
			if !tt.wantEmpty && got == "" {
				t.Errorf("expected non-empty for %q", tt.deploymentID)
			}
		})
	}
}
