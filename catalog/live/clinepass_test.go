package live

import (
	"testing"
)

func TestFetchClinePass_ReturnsStaticList(t *testing.T) {
	t.Parallel()
	entries, err := FetchClinePass(map[string]string{
		"CLINE_API_KEY": "cp-test123",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 11 {
		t.Fatalf("expected 11 curated models, got %d", len(entries))
	}
	expected := []string{
		"cline-pass/deepseek-v4-pro",
		"cline-pass/deepseek-v4-flash",
		"cline-pass/glm-5.2",
		"cline-pass/kimi-k2.7-code",
		"cline-pass/kimi-k2.6",
		"cline-pass/minimax-m3",
		"cline-pass/mimo-v2.5-pro",
		"cline-pass/mimo-v2.5",
		"cline-pass/qwen3.7-max",
		"cline-pass/qwen3.7-plus",
		"cline-pass/poolside-laguna-m.1-free",
	}
	for i, e := range entries {
		if e.ID != expected[i] {
			t.Errorf("entry[%d].ID = %q, want %q", i, e.ID, expected[i])
		}
	}
	// Verify OpenRouter pricing (poolside is free).
	if entries[0].InputPricePer1M != 1.74 || entries[0].OutputPricePer1M != 3.48 {
		t.Errorf("deepseek-v4-pro: in=%f out=%f, want 1.74/3.48", entries[0].InputPricePer1M, entries[0].OutputPricePer1M)
	}
	if entries[10].InputPricePer1M != 0 || entries[10].OutputPricePer1M != 0 {
		t.Errorf("poolside: in=%f out=%f, want 0/0", entries[10].InputPricePer1M, entries[10].OutputPricePer1M)
	}
}

func TestFetchClinePass_NoKey(t *testing.T) {
	t.Parallel()
	entries, err := FetchClinePass(map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries with no key, got %d", len(entries))
	}
}
