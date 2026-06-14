package registry

import (
	"testing"
)

// --- NewProviderRegistry + Register + Get ---

func TestNewProviderRegistry_Empty(t *testing.T) {
	r := NewProviderRegistry()
	if len(r.All()) != 0 {
		t.Fatalf("expected empty registry, got %d specs", len(r.All()))
	}
}

func TestProviderRegistry_RegisterAndGet(t *testing.T) {
	r := NewProviderRegistry()
	spec := ProviderSpec{
		ProviderID:    "test-provider",
		DisplayName:   "Test Provider",
		DeploymentID:  "test-direct",
		RequiresKey:   true,
		CredentialEnv: "TEST_API_KEY",
	}
	r.Register(spec)

	got, ok := r.Get("test-provider")
	if !ok {
		t.Fatal("Get returned false for registered provider")
	}
	if got.ProviderID != "test-provider" {
		t.Errorf("ProviderID = %q", got.ProviderID)
	}
	if got.DisplayName != "Test Provider" {
		t.Errorf("DisplayName = %q", got.DisplayName)
	}
}

func TestProviderRegistry_Get_NotFound(t *testing.T) {
	r := NewProviderRegistry()
	_, ok := r.Get("nonexistent")
	if ok {
		t.Error("expected not found")
	}
}

func TestProviderRegistry_Get_AliasResolution(t *testing.T) {
	r := NewProviderRegistry()
	r.Register(ProviderSpec{ProviderID: "gemini", DisplayName: "Gemini", DeploymentID: "gemini-direct"})

	got, ok := r.Get("google")
	if !ok {
		t.Fatal("expected google->gemini alias resolution")
	}
	if got.ProviderID != "gemini" {
		t.Errorf("resolved to %q, want gemini", got.ProviderID)
	}
}

func TestProviderRegistry_Register_OverwritesExisting(t *testing.T) {
	r := NewProviderRegistry()
	r.Register(ProviderSpec{ProviderID: "p", DisplayName: "Original"})
	r.Register(ProviderSpec{ProviderID: "p", DisplayName: "Updated"})

	got, ok := r.Get("p")
	if !ok {
		t.Fatal("expected provider to exist")
	}
	if got.DisplayName != "Updated" {
		t.Errorf("DisplayName = %q, want Updated", got.DisplayName)
	}
	if len(r.All()) != 1 {
		t.Errorf("All() len = %d, want 1 (no duplicate)", len(r.All()))
	}
}

// --- GetByEnv tests ---

func TestProviderRegistry_GetByEnv(t *testing.T) {
	r := NewProviderRegistry()
	r.Register(ProviderSpec{ProviderID: "a", CredentialEnv: "A_KEY"})
	r.Register(ProviderSpec{ProviderID: "b", CredentialEnv: "B_KEY"})

	tests := []struct {
		env    string
		wantID string
		wantOK bool
	}{
		{"A_KEY", "a", true},
		{"B_KEY", "b", true},
		{"C_KEY", "", false},
		{"", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			got, ok := r.GetByEnv(tt.env)
			if ok != tt.wantOK {
				t.Errorf("ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && got.ProviderID != tt.wantID {
				t.Errorf("ProviderID = %q, want %q", got.ProviderID, tt.wantID)
			}
		})
	}
}

// --- GetForLiveFetcher tests ---

func TestProviderRegistry_GetForLiveFetcher(t *testing.T) {
	r := NewProviderRegistry()
	r.Register(ProviderSpec{ProviderID: "a", LiveFetcherKey: "fetcher-a"})
	r.Register(ProviderSpec{ProviderID: "b", LiveFetcherKey: "fetcher-b"})

	tests := []struct {
		key    string
		wantID string
		wantOK bool
	}{
		{"fetcher-a", "a", true},
		{"fetcher-b", "b", true},
		{"nonexistent", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got, ok := r.GetForLiveFetcher(tt.key)
			if ok != tt.wantOK {
				t.Errorf("ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && got.ProviderID != tt.wantID {
				t.Errorf("ProviderID = %q, want %q", got.ProviderID, tt.wantID)
			}
		})
	}
}

// --- All tests ---

func TestProviderRegistry_All_ReturnsCopy(t *testing.T) {
	r := NewProviderRegistry()
	r.Register(ProviderSpec{ProviderID: "a"})
	r.Register(ProviderSpec{ProviderID: "b"})

	all := r.All()
	if len(all) != 2 {
		t.Fatalf("All() len = %d, want 2", len(all))
	}
	// Modifying returned slice should not affect registry
	all[0].ProviderID = "mutated"
	fresh := r.All()
	for _, s := range fresh {
		if s.ProviderID == "mutated" {
			t.Error("All() returned a reference, not a copy")
		}
	}
}

// --- CredentialProviders tests ---

func TestProviderRegistry_CredentialProviders(t *testing.T) {
	r := NewProviderRegistry()
	r.Register(ProviderSpec{ProviderID: "a", RequiresKey: true})
	r.Register(ProviderSpec{ProviderID: "b", RequiresKey: false})

	cp := r.CredentialProviders()
	if len(cp) != 2 {
		t.Fatalf("CredentialProviders() len = %d, want 2 (all specs)", len(cp))
	}
}

// --- LiveDiscoverable tests ---

func TestProviderRegistry_LiveDiscoverable(t *testing.T) {
	r := NewProviderRegistry()
	r.Register(ProviderSpec{ProviderID: "a", LiveFetcherKey: "fetch-a"})
	r.Register(ProviderSpec{ProviderID: "b", LiveFetcherKey: ""})
	r.Register(ProviderSpec{ProviderID: "c", LiveFetcherKey: "fetch-c"})

	ld := r.LiveDiscoverable()
	if len(ld) != 2 {
		t.Fatalf("LiveDiscoverable() len = %d, want 2", len(ld))
	}
	ids := map[string]bool{}
	for _, s := range ld {
		ids[s.ProviderID] = true
	}
	if !ids["a"] || !ids["c"] {
		t.Errorf("expected a and c in live discoverable, got %v", ids)
	}
}

// --- LiveFetcherKeys tests ---

func TestProviderRegistry_LiveFetcherKeys_Sorted(t *testing.T) {
	r := NewProviderRegistry()
	r.Register(ProviderSpec{ProviderID: "z", LiveFetcherKey: "z-fetch"})
	r.Register(ProviderSpec{ProviderID: "a", LiveFetcherKey: "a-fetch"})
	r.Register(ProviderSpec{ProviderID: "m", LiveFetcherKey: ""}) // no fetcher

	keys := r.LiveFetcherKeys()
	if len(keys) != 2 {
		t.Fatalf("LiveFetcherKeys() len = %d, want 2", len(keys))
	}
	if keys[0] != "a-fetch" || keys[1] != "z-fetch" {
		t.Errorf("expected sorted keys, got %v", keys)
	}
}

func TestProviderRegistry_LiveFetcherKeys_Deduplicates(t *testing.T) {
	r := NewProviderRegistry()
	r.Register(ProviderSpec{ProviderID: "a", LiveFetcherKey: "shared"})
	r.Register(ProviderSpec{ProviderID: "b", LiveFetcherKey: "shared"})

	keys := r.LiveFetcherKeys()
	if len(keys) != 1 {
		t.Fatalf("expected deduplication, got %v", keys)
	}
}

// --- DeploymentEnvFallbacks tests ---

func TestProviderRegistry_DeploymentEnvFallbacks_WithAPIKey(t *testing.T) {
	r := NewProviderRegistry()
	r.Register(ProviderSpec{
		ProviderID: "test", DeploymentID: "test-direct",
		RequiresKey: true, CredentialEnv: "TEST_API_KEY",
	})

	fbs := r.DeploymentEnvFallbacks()
	dep, ok := fbs["test-direct"]
	if !ok {
		t.Fatal("missing test-direct in fallbacks")
	}
	found := false
	for _, fb := range dep {
		if fb.Field == "api_key" {
			for _, env := range fb.Env {
				if env == "TEST_API_KEY" {
					found = true
				}
			}
		}
	}
	if !found {
		t.Error("expected TEST_API_KEY in api_key fallbacks")
	}
}

func TestProviderRegistry_DeploymentEnvFallbacks_BaseURL(t *testing.T) {
	r := NewProviderRegistry()
	r.Register(ProviderSpec{
		ProviderID: "test", DeploymentID: "test-direct",
		RequiresKey: false, CredentialEnv: "TEST_BASE_URL",
		BaseURLEnv: []string{"TEST_BASE_URL"},
	})

	fbs := r.DeploymentEnvFallbacks()
	dep, ok := fbs["test-direct"]
	if !ok {
		t.Fatal("missing test-direct in fallbacks")
	}
	foundBaseURL := false
	for _, fb := range dep {
		if fb.Field == "base_url" {
			for _, env := range fb.Env {
				if env == "TEST_BASE_URL" {
					foundBaseURL = true
				}
			}
		}
	}
	if !foundBaseURL {
		t.Error("expected TEST_BASE_URL in base_url fallbacks")
	}
}

// --- CredentialPresent tests ---

func TestCredentialPresent(t *testing.T) {
	tests := []struct {
		name string
		spec ProviderSpec
		env  map[string]string
		want bool
	}{
		{
			name: "key_present",
			spec: ProviderSpec{RequiresKey: true, CredentialEnv: "API_KEY"},
			env:  map[string]string{"API_KEY": "sk-test"},
			want: true,
		},
		{
			name: "key_missing",
			spec: ProviderSpec{RequiresKey: true, CredentialEnv: "API_KEY"},
			env:  map[string]string{},
			want: false,
		},
		{
			name: "key_empty_value",
			spec: ProviderSpec{RequiresKey: true, CredentialEnv: "API_KEY"},
			env:  map[string]string{"API_KEY": ""},
			want: false,
		},
		{
			name: "no_key_required_base_url_present",
			spec: ProviderSpec{RequiresKey: false, CredentialEnv: "BASE_URL"},
			env:  map[string]string{"BASE_URL": "http://localhost"},
			want: true,
		},
		{
			name: "no_key_required_base_url_missing",
			spec: ProviderSpec{RequiresKey: false, CredentialEnv: "BASE_URL"},
			env:  map[string]string{},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CredentialPresent(tt.spec, tt.env)
			if got != tt.want {
				t.Errorf("CredentialPresent() = %v, want %v", got, tt.want)
			}
		})
	}
}

// --- CredentialRegistry tests ---

func TestCredentialRegistry_SortedByOrder(t *testing.T) {
	creds := CredentialRegistry()
	if len(creds) == 0 {
		t.Fatal("expected non-empty credential registry")
	}
	for i := 1; i < len(creds); i++ {
		if creds[i].SortOrder < creds[i-1].SortOrder {
			t.Errorf("not sorted: creds[%d].SortOrder=%d < creds[%d].SortOrder=%d",
				i, creds[i].SortOrder, i-1, creds[i-1].SortOrder)
		}
	}
}

func TestCredentialRegistry_FieldsPopulated(t *testing.T) {
	creds := CredentialRegistry()
	for _, c := range creds {
		if c.ProviderID == "" {
			t.Error("empty ProviderID in credential registry")
		}
		if c.DisplayName == "" {
			t.Errorf("empty DisplayName for %q", c.ProviderID)
		}
		if c.DeploymentID == "" {
			t.Errorf("empty DeploymentID for %q", c.ProviderID)
		}
	}
}

// --- DisplayName tests ---

func TestDisplayName_Registered(t *testing.T) {
	tests := []struct {
		id   string
		want string
	}{
		{"anthropic", "Anthropic"},
		{"openai", "OpenAI"},
		{"gemini", "Gemini API"},
		{"ollama", "Ollama"},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			got := DisplayName(tt.id)
			if got != tt.want {
				t.Errorf("DisplayName(%q) = %q, want %q", tt.id, got, tt.want)
			}
		})
	}
}

func TestDisplayName_Unknown(t *testing.T) {
	got := DisplayName("nonexistent_provider_xyz")
	if got != "nonexistent_provider_xyz" {
		t.Errorf("DisplayName(unknown) = %q, want fallback to input", got)
	}
}

// --- SpecForLiveFetcher tests ---

func TestSpecForLiveFetcher(t *testing.T) {
	spec, ok := SpecForLiveFetcher("anthropic")
	if !ok {
		t.Fatal("expected anthropic fetcher spec")
	}
	if spec.ProviderID != "anthropic" {
		t.Errorf("ProviderID = %q", spec.ProviderID)
	}
}

func TestSpecForLiveFetcher_NotFound(t *testing.T) {
	_, ok := SpecForLiveFetcher("nonexistent_fetcher")
	if ok {
		t.Error("expected not found")
	}
}

// --- LiveCatalogKeyForFetcher tests ---

func TestLiveCatalogKeyForFetcher(t *testing.T) {
	tests := []struct {
		fetcherKey string
		wantKey    string
	}{
		{"anthropic", "anthropic"},
		{"openai", "openai"},
		{"gemini", "gemini"},
	}
	for _, tt := range tests {
		t.Run(tt.fetcherKey, func(t *testing.T) {
			got := LiveCatalogKeyForFetcher(tt.fetcherKey)
			if got != tt.wantKey {
				t.Errorf("LiveCatalogKeyForFetcher(%q) = %q, want %q", tt.fetcherKey, got, tt.wantKey)
			}
		})
	}
}

func TestLiveCatalogKeyForFetcher_Unknown(t *testing.T) {
	got := LiveCatalogKeyForFetcher("unknown_fetcher")
	if got != "unknown_fetcher" {
		t.Errorf("expected fallback to input, got %q", got)
	}
}
