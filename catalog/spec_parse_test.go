package catalog

import (
	"encoding/json"
	"testing"
	"time"
)

// --- ParseCatalogV1 tests ---

func TestParseCatalogV1_V1Format(t *testing.T) {
	c := testLegacyCatalogV1()
	data, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseCatalogV1(data)
	if err != nil {
		t.Fatalf("ParseCatalogV1 failed: %v", err)
	}
	if parsed.SchemaVersion != CatalogV1SchemaVersion {
		t.Fatalf("schema_version = %q", parsed.SchemaVersion)
	}
	if len(parsed.Models) == 0 {
		t.Fatal("expected models in parsed catalog")
	}
}

func TestParseCatalogV1_LegacyFormat(t *testing.T) {
	legacy := testLegacyModelCatalog()
	data, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseCatalogV1(data)
	if err != nil {
		t.Fatalf("ParseCatalogV1 legacy failed: %v", err)
	}
	if parsed.SchemaVersion != CatalogV1SchemaVersion {
		t.Fatalf("schema_version = %q, want %q", parsed.SchemaVersion, CatalogV1SchemaVersion)
	}
	if len(parsed.Offerings) == 0 {
		t.Fatal("expected offerings from legacy conversion")
	}
}

func TestParseCatalogV1_InvalidJSON(t *testing.T) {
	_, err := ParseCatalogV1([]byte(`not json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseCatalogV1_StrictV1RejectsUnknownFields(t *testing.T) {
	data := []byte(`{
		"schema_version": "model-catalog/v1",
		"unknown_field": true,
		"generated_at": "2026-01-01T00:00:00Z",
		"stale_after": "2026-02-01T00:00:00Z",
		"providers": {"p": {"id": "p", "name": "P"}},
		"api_protocols": {"a": {"id": "a", "name": "A"}},
		"deployments": {},
		"models": {},
		"offerings": []
	}`)
	_, err := ParseCatalogV1(data)
	if err == nil {
		t.Fatal("expected error for unknown fields in v1 format")
	}
}

// --- ValidateCatalogV1 tests ---

func TestValidateCatalogV1_NilCatalog(t *testing.T) {
	if err := ValidateCatalogV1(nil); err == nil {
		t.Fatal("expected error for nil catalog")
	}
}

func TestValidateCatalogV1_BadSchemaVersion(t *testing.T) {
	c := testLegacyCatalogV1()
	c.SchemaVersion = "wrong"
	if err := ValidateCatalogV1(&c); err == nil {
		t.Fatal("expected error for wrong schema_version")
	}
}

func TestValidateCatalogV1_MissingGeneratedAt(t *testing.T) {
	c := testLegacyCatalogV1()
	c.GeneratedAt = time.Time{}
	if err := ValidateCatalogV1(&c); err == nil {
		t.Fatal("expected error for zero generated_at")
	}
}

func TestValidateCatalogV1_StaleAfterBeforeGeneratedAt(t *testing.T) {
	c := testLegacyCatalogV1()
	c.GeneratedAt = time.Now().UTC()
	c.StaleAfter = c.GeneratedAt.Add(-time.Hour)
	if err := ValidateCatalogV1(&c); err == nil {
		t.Fatal("expected error when stale_after < generated_at")
	}
}

func TestValidateCatalogV1_BadProviderIDMismatch(t *testing.T) {
	c := testLegacyCatalogV1()
	c.Providers["bad"] = ProviderV1{ID: "mismatch", Name: "Bad"}
	if err := ValidateCatalogV1(&c); err == nil {
		t.Fatal("expected error for provider ID mismatch")
	}
}

func TestValidateCatalogV1_BadDeploymentProviderRef(t *testing.T) {
	c := testLegacyCatalogV1()
	c.Deployments["bad-dep"] = DeploymentV1{
		ID:                  "bad-dep",
		Name:                "Bad",
		ProviderID:          "nonexistent",
		APIProtocolID:       "openai-chat-completions",
		AdapterConstructor:  "openai",
		NativeModelIDSource: NativeModelIDCatalogKnown,
	}
	if err := ValidateCatalogV1(&c); err == nil {
		t.Fatal("expected error for bad provider reference in deployment")
	}
}

func TestValidateCatalogV1_BadDeploymentProtocolRef(t *testing.T) {
	c := testLegacyCatalogV1()
	c.Deployments["bad-dep"] = DeploymentV1{
		ID:                  "bad-dep",
		Name:                "Bad",
		ProviderID:          "openai",
		APIProtocolID:       "nonexistent-protocol",
		AdapterConstructor:  "openai",
		NativeModelIDSource: NativeModelIDCatalogKnown,
	}
	if err := ValidateCatalogV1(&c); err == nil {
		t.Fatal("expected error for bad api_protocol reference in deployment")
	}
}

func TestValidateCatalogV1_BadModelProviderRef(t *testing.T) {
	c := testLegacyCatalogV1()
	c.Models["bad/model"] = ModelV1{
		ID:         "bad/model",
		ProviderID: "nonexistent",
		Name:       "Bad Model",
	}
	if err := ValidateCatalogV1(&c); err == nil {
		t.Fatal("expected error for bad provider reference in model")
	}
}

func TestValidateCatalogV1_DuplicateOffering(t *testing.T) {
	c := testLegacyCatalogV1()
	c.Offerings = append(c.Offerings, c.Offerings[0])
	if err := ValidateCatalogV1(&c); err == nil {
		t.Fatal("expected error for duplicate offering")
	}
}

func TestValidateCatalogV1_OfferingBadModelRef(t *testing.T) {
	c := testLegacyCatalogV1()
	c.Offerings = append(c.Offerings, ModelOfferingV1{
		ID:               "anthropic-direct:fake",
		CanonicalModelID: "nonexistent/model",
		DeploymentID:     "anthropic-direct",
		NativeModelID:    "fake",
		Pricing:          PricingV1{Status: PricingUnknown},
	})
	if err := ValidateCatalogV1(&c); err == nil {
		t.Fatal("expected error for offering referencing unknown model")
	}
}

func TestValidateCatalogV1_InvalidPricingStatus(t *testing.T) {
	c := testLegacyCatalogV1()
	c.Offerings[0].Pricing = PricingV1{Status: "invalid_status"}
	if err := ValidateCatalogV1(&c); err == nil {
		t.Fatal("expected error for invalid pricing status")
	}
}

func TestValidateCatalogV1_InvalidCapabilityState(t *testing.T) {
	c := testLegacyCatalogV1()
	c.Offerings[0].Capabilities = CapabilitySetV1{
		FunctionCalling: "invalid_state",
	}
	if err := ValidateCatalogV1(&c); err == nil {
		t.Fatal("expected error for invalid capability state")
	}
}

func TestValidateCatalogV1_ValidBootstrapCatalog(t *testing.T) {
	c := BootstrapCatalogV1()
	if err := ValidateCatalogV1(&c); err != nil {
		t.Fatalf("bootstrap catalog should validate: %v", err)
	}
}

// --- SplitOfferingIDV1 tests ---

func TestSplitOfferingIDV1_Valid(t *testing.T) {
	dep, native, ok := SplitOfferingIDV1("anthropic-direct:claude-sonnet-4-6")
	if !ok || dep != "anthropic-direct" || native != "claude-sonnet-4-6" {
		t.Fatalf("got (%q, %q, %v)", dep, native, ok)
	}
}

func TestSplitOfferingIDV1_Invalid(t *testing.T) {
	tests := []string{"", "nocolon", ":empty-left", "empty-right:"}
	for _, id := range tests {
		_, _, ok := SplitOfferingIDV1(id)
		if ok {
			t.Errorf("SplitOfferingIDV1(%q) should return ok=false", id)
		}
	}
}

// --- looksCanonicalModelID tests ---

func TestLooksCanonicalModelID(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"anthropic/claude-sonnet-4-6", true},
		{"openai/gpt-4o", true},
		{"google/gemini-2.0-flash", true},
		{"no-slash", false},
		{"/empty-owner", false},
		{"empty-model/", false},
		{"has space/model", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := looksCanonicalModelID(tt.input); got != tt.want {
			t.Errorf("looksCanonicalModelID(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

// --- validNativeModelIDSource tests ---

func TestValidNativeModelIDSource(t *testing.T) {
	valid := []NativeModelIDSource{
		NativeModelIDCatalogKnown, NativeModelIDDiscovered,
		NativeModelIDUserConfigured, NativeModelIDCatalogOrUser,
	}
	for _, s := range valid {
		if !validNativeModelIDSource(s) {
			t.Errorf("validNativeModelIDSource(%q) should be true", s)
		}
	}
	if validNativeModelIDSource("invalid") {
		t.Error("validNativeModelIDSource('invalid') should be false")
	}
}

// --- canonicalModelID tests ---

func TestCanonicalModelID(t *testing.T) {
	tests := []struct {
		provider, native, want string
	}{
		{"anthropic", "claude-sonnet-4-6", "anthropic/claude-sonnet-4-6"},
		{"openai", "gpt-4o", "openai/gpt-4o"},
		{"openrouter", "anthropic/claude-sonnet-4-6", "openrouter/anthropic/claude-sonnet-4-6"},
		{"z-ai", "zai/glm-5.1", "zai/glm-5.1"},
	}
	for _, tt := range tests {
		got := canonicalModelID(tt.provider, tt.native)
		if got != tt.want {
			t.Errorf("canonicalModelID(%q, %q) = %q, want %q", tt.provider, tt.native, got, tt.want)
		}
	}
}

// --- canonicalProviderID tests ---

func TestCanonicalProviderID(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"anthropic", "anthropic"},
		{"openai", "openai"},
		{"gemini", "google"},
		{"grok", "xai"},
		{"zai", "z-ai"},
		{"ollama", "ollama"},
		{"xiaomi-mimo", "xiaomi_mimo_payg"},
		{"xiaomi-mimo-payg", "xiaomi_mimo_payg"},
		{"xiaomi-mimo-token-plan", "xiaomi_mimo_token_plan"},
	}
	for _, tt := range tests {
		got := CanonicalProviderID(tt.input)
		if got != tt.want {
			t.Errorf("CanonicalProviderID(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// --- CatalogV1FromLegacy additional tests ---

func TestCatalogV1FromLegacy_ProducesValidCatalog(t *testing.T) {
	legacy := testLegacyModelCatalog()
	c := CatalogV1FromLegacy(legacy)
	if err := ValidateCatalogV1(&c); err != nil {
		t.Fatalf("CatalogV1FromLegacy produced invalid catalog: %v", err)
	}
}

func TestCatalogV1FromLegacy_PreservesTimestamp(t *testing.T) {
	legacy := testLegacyModelCatalog()
	legacy.UpdatedAt = "2026-01-15T12:00:00Z"
	c := CatalogV1FromLegacy(legacy)
	ts, err := time.Parse(time.RFC3339, "2026-01-15T12:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	if !c.GeneratedAt.Equal(ts) {
		t.Fatalf("generated_at = %v, want %v", c.GeneratedAt, ts)
	}
}

func TestCatalogV1FromLegacy_SkipsEmptyNativeIDs(t *testing.T) {
	legacy := testLegacyModelCatalog()
	legacy.Providers["anthropic"] = append(
		legacy.Providers["anthropic"],
		ModelCatalogEntry{ID: "", DisplayName: "Empty"},
		ModelCatalogEntry{ID: "  ", DisplayName: "Whitespace"},
	)
	c := CatalogV1FromLegacy(legacy)
	for _, o := range c.Offerings {
		if o.NativeModelID == "" || o.NativeModelID == "  " {
			t.Fatalf("should skip empty native IDs, found %q", o.NativeModelID)
		}
	}
}

func TestCatalogV1FromLegacy_SkipsUnknownProviders(t *testing.T) {
	legacy := testLegacyModelCatalog()
	legacy.Providers["unknown_provider"] = []ModelCatalogEntry{
		{ID: "some-model"},
	}
	c := CatalogV1FromLegacy(legacy)
	for _, o := range c.Offerings {
		if o.DeploymentID == "" {
			t.Fatal("should skip unknown providers")
		}
	}
}

// --- pricingFromLegacy tests ---

func TestPricingFromLegacy_KnownPricing(t *testing.T) {
	entry := ModelCatalogEntry{InputPricePer1M: 3, OutputPricePer1M: 15}
	now := time.Now().UTC()
	p := pricingFromLegacy(entry, now, "test")
	if p.Status != PricingKnown {
		t.Fatalf("status = %q, want known", p.Status)
	}
	if p.RatesPer1M["input_tokens"] != 3 || p.RatesPer1M["output_tokens"] != 15 {
		t.Fatalf("rates = %v", p.RatesPer1M)
	}
}

func TestPricingFromLegacy_NegativePricing(t *testing.T) {
	entry := ModelCatalogEntry{InputPricePer1M: -1, OutputPricePer1M: 0}
	now := time.Now().UTC()
	p := pricingFromLegacy(entry, now, "test")
	if p.Status != PricingUnknown {
		t.Fatalf("status = %q, want unknown", p.Status)
	}
}

func TestPricingFromLegacy_ZeroPricing(t *testing.T) {
	entry := ModelCatalogEntry{InputPricePer1M: 0, OutputPricePer1M: 0}
	now := time.Now().UTC()
	p := pricingFromLegacy(entry, now, "test")
	if p.Status != PricingUnknown {
		t.Fatalf("status = %q, want unknown", p.Status)
	}
	if p.RatesPer1M != nil {
		t.Fatal("zero pricing should have nil rates")
	}
}

func TestPricingFromLegacy_FreeModel(t *testing.T) {
	entry := ModelCatalogEntry{ID: "model:free", InputPricePer1M: 0, OutputPricePer1M: 0}
	now := time.Now().UTC()
	p := pricingFromLegacy(entry, now, "test")
	if p.Status != PricingFree {
		t.Fatalf("status = %q, want free", p.Status)
	}
}

// --- sanitizePricingV1 tests ---

func TestSanitizePricingV1_DropsNegativeRates(t *testing.T) {
	p := PricingV1{
		Status:     PricingKnown,
		Currency:   "USD",
		RatesPer1M: map[string]float64{"input_tokens": 3, "output_tokens": -1},
	}
	cleaned := sanitizePricingV1(p)
	if _, ok := cleaned.RatesPer1M["output_tokens"]; ok {
		t.Error("negative rate should be dropped")
	}
	if cleaned.RatesPer1M["input_tokens"] != 3 {
		t.Error("positive rate should be kept")
	}
}

func TestSanitizePricingV1_AllNegativeBecomesUnknown(t *testing.T) {
	p := PricingV1{
		Status:     PricingKnown,
		Currency:   "USD",
		RatesPer1M: map[string]float64{"input_tokens": -1, "output_tokens": -2},
	}
	cleaned := sanitizePricingV1(p)
	if cleaned.Status != PricingUnknown {
		t.Fatalf("status = %q, want unknown", cleaned.Status)
	}
	if cleaned.RatesPer1M != nil {
		t.Fatal("all-negative rates should result in nil RatesPer1M")
	}
}

func TestSanitizePricingV1_EmptyRatesMapReturnsUnchanged(t *testing.T) {
	p := PricingV1{
		Status:     PricingKnown,
		Currency:   "USD",
		RatesPer1M: map[string]float64{},
	}
	cleaned := sanitizePricingV1(p)
	// Empty rates map returns early (len == 0) without modifying status
	if cleaned.Status != PricingKnown {
		t.Fatalf("status = %q, want known (empty map returns unchanged)", cleaned.Status)
	}
}

// --- uniqueNonEmpty tests ---

func TestUniqueNonEmpty(t *testing.T) {
	got := uniqueNonEmpty("a", "", "b", "a", "  ", "c")
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i, v := range got {
		if v != want[i] {
			t.Fatalf("got[%d] = %q, want %q", i, v, want[i])
		}
	}
}

// --- CompileCatalogV1 tests ---

func TestCompileCatalogV1_BuildsIndexes(t *testing.T) {
	c := testLegacyCatalogV1()
	compiled, err := CompileCatalogV1(&c)
	if err != nil {
		t.Fatalf("CompileCatalogV1: %v", err)
	}
	if len(compiled.OfferingsByID) != len(c.Offerings) {
		t.Fatalf("OfferingsByID len = %d, want %d", len(compiled.OfferingsByID), len(c.Offerings))
	}
	for _, o := range c.Offerings {
		if _, ok := compiled.OfferingsByID[o.ID]; !ok {
			t.Errorf("missing offering %q in OfferingsByID", o.ID)
		}
	}
	if len(compiled.OfferingsByCanonicalModel) == 0 {
		t.Fatal("OfferingsByCanonicalModel should not be empty")
	}
	if len(compiled.OfferingsByDeployment) == 0 {
		t.Fatal("OfferingsByDeployment should not be empty")
	}
}

func TestCompileCatalogV1_AppliesEnvFallbacks(t *testing.T) {
	c := testLegacyCatalogV1()
	// Remove env fallbacks to test that compile adds them
	for id, dep := range c.Deployments {
		dep.EnvFallbacks = nil
		c.Deployments[id] = dep
	}
	compiled, err := CompileCatalogV1(&c)
	if err != nil {
		t.Fatalf("CompileCatalogV1: %v", err)
	}
	anthDep := compiled.DeploymentsByID["anthropic-direct"]
	if len(anthDep.EnvFallbacks) == 0 {
		t.Error("CompileCatalogV1 should apply env fallbacks")
	}
}

func TestCompileCatalogV1_InvalidCatalogFails(t *testing.T) {
	c := CatalogV1{} // missing required fields
	_, err := CompileCatalogV1(&c)
	if err == nil {
		t.Fatal("expected error for invalid catalog")
	}
}

// --- CanonicalModelForAliasOrID tests ---

func TestCanonicalModelForAliasOrID_ByDirectID(t *testing.T) {
	c := testLegacyCatalogV1()
	compiled, _ := CompileCatalogV1(&c)
	canonical, ok := compiled.CanonicalModelForAliasOrID("anthropic/claude-sonnet-4-6")
	if !ok || canonical != "anthropic/claude-sonnet-4-6" {
		t.Fatalf("got (%q, %v)", canonical, ok)
	}
}

func TestCanonicalModelForAliasOrID_ByAlias(t *testing.T) {
	c := testLegacyCatalogV1()
	compiled, _ := CompileCatalogV1(&c)
	canonical, ok := compiled.CanonicalModelForAliasOrID("claude-sonnet-4-6")
	if !ok || canonical != "anthropic/claude-sonnet-4-6" {
		t.Fatalf("got (%q, %v)", canonical, ok)
	}
}

func TestCanonicalModelForAliasOrID_NotFound(t *testing.T) {
	c := testLegacyCatalogV1()
	compiled, _ := CompileCatalogV1(&c)
	_, ok := compiled.CanonicalModelForAliasOrID("nonexistent-model")
	if ok {
		t.Fatal("expected not found")
	}
}

func TestCanonicalModelForAliasOrID_NilCompiled(t *testing.T) {
	var c *CompiledCatalogV1
	_, ok := c.CanonicalModelForAliasOrID("anything")
	if ok {
		t.Fatal("nil compiled should return false")
	}
}

// --- OfferingForDeployment tests ---

func TestOfferingForDeployment_Found(t *testing.T) {
	c := testLegacyCatalogV1()
	compiled, _ := CompileCatalogV1(&c)
	offering, ok := compiled.OfferingForDeployment("anthropic/claude-sonnet-4-6", "anthropic-direct")
	if !ok {
		t.Fatal("expected offering")
	}
	if offering.NativeModelID != "claude-sonnet-4-6" {
		t.Fatalf("native = %q", offering.NativeModelID)
	}
}

func TestOfferingForDeployment_NotFound(t *testing.T) {
	c := testLegacyCatalogV1()
	compiled, _ := CompileCatalogV1(&c)
	_, ok := compiled.OfferingForDeployment("anthropic/claude-sonnet-4-6", "nonexistent")
	if ok {
		t.Fatal("expected not found")
	}
}

// --- ResolvedRemoteCatalogURL tests ---

func TestResolvedRemoteCatalogURL_Explicit(t *testing.T) {
	got := ResolvedRemoteCatalogURL("https://custom.example.com/catalog.json")
	if got != "https://custom.example.com/catalog.json" {
		t.Fatalf("got %q", got)
	}
}

func TestResolvedRemoteCatalogURL_Default(t *testing.T) {
	// Clear env var if set
	t.Setenv("EYRIE_MODEL_CATALOG_URL", "")
	got := ResolvedRemoteCatalogURL("")
	if got != DefaultCatalogV1URL {
		t.Fatalf("got %q, want %q", got, DefaultCatalogV1URL)
	}
}

func TestResolvedRemoteCatalogURL_EnvOverride(t *testing.T) {
	t.Setenv("EYRIE_MODEL_CATALOG_URL", "https://env.example.com/catalog.json")
	got := ResolvedRemoteCatalogURL("")
	if got != "https://env.example.com/catalog.json" {
		t.Fatalf("got %q", got)
	}
}
