package client

import "testing"

func TestFeatureDefaultProviders(t *testing.T) {
	pf := NewProviderFeatures()

	// Anthropic should have all core features
	anthropic := pf.Get("anthropic")
	if !anthropic.Thinking {
		t.Error("anthropic should support thinking")
	}
	if !anthropic.ToolUse {
		t.Error("anthropic should support tool_use")
	}
	if !anthropic.Images {
		t.Error("anthropic should support images")
	}
	if !anthropic.Streaming {
		t.Error("anthropic should support streaming")
	}
	if !anthropic.Caching {
		t.Error("anthropic should support caching")
	}
	if !anthropic.JSON {
		t.Error("anthropic should support JSON mode")
	}
	if anthropic.Embeddings {
		t.Error("anthropic should NOT support embeddings")
	}
	if anthropic.MaxContext != 200000 {
		t.Errorf("anthropic MaxContext = %d, want 200000", anthropic.MaxContext)
	}
}

func TestFeatureSupportsFeatureChecks(t *testing.T) {
	pf := NewProviderFeatures()

	tests := []struct {
		provider string
		feature  string
		want     bool
	}{
		{"anthropic", "thinking", true},
		{"anthropic", "caching", true},
		{"anthropic", "embeddings", false},
		{"openai", "thinking", true},
		{"openai", "caching", false},
		{"openai", "embeddings", true},
		{"ollama", "thinking", false},
		{"ollama", "tool_use", true},
		{"grok", "images", false},
		{"grok", "streaming", true},
		{"gemini", "thinking", true},
		{"gemini", "embeddings", true},
	}

	for _, tc := range tests {
		got := pf.Supports(tc.provider, tc.feature)
		if got != tc.want {
			t.Errorf("Supports(%q, %q) = %v, want %v", tc.provider, tc.feature, got, tc.want)
		}
	}
}

func TestFeatureProviderSpecific(t *testing.T) {
	pf := NewProviderFeatures()

	// OpenAI specifics
	openai := pf.Get("openai")
	if openai.MaxContext != 128000 {
		t.Errorf("openai MaxContext = %d, want 128000", openai.MaxContext)
	}
	if openai.Caching {
		t.Error("openai should not support caching")
	}

	// Gemini specifics
	gemini := pf.Get("gemini")
	if gemini.MaxContext != 1000000 {
		t.Errorf("gemini MaxContext = %d, want 1000000", gemini.MaxContext)
	}

	// Ollama specifics
	ollama := pf.Get("ollama")
	if ollama.MaxContext != 32000 {
		t.Errorf("ollama MaxContext = %d, want 32000", ollama.MaxContext)
	}
	if ollama.Thinking {
		t.Error("ollama should not support thinking")
	}

	// Grok specifics
	grok := pf.Get("grok")
	if grok.MaxContext != 131072 {
		t.Errorf("grok MaxContext = %d, want 131072", grok.MaxContext)
	}
	if grok.Thinking {
		t.Error("grok should not support thinking")
	}
}

func TestFeatureUnknownProviderDefaults(t *testing.T) {
	pf := NewProviderFeatures()

	unknown := pf.Get("some-unknown-provider")
	if !unknown.ToolUse {
		t.Error("unknown provider should default to ToolUse=true")
	}
	if !unknown.Streaming {
		t.Error("unknown provider should default to Streaming=true")
	}
	if unknown.MaxContext != 32000 {
		t.Errorf("unknown provider MaxContext = %d, want 32000", unknown.MaxContext)
	}
	if unknown.Thinking {
		t.Error("unknown provider should default to Thinking=false")
	}
	if unknown.Caching {
		t.Error("unknown provider should default to Caching=false")
	}
}

func TestFeatureCaseInsensitiveProvider(t *testing.T) {
	pf := NewProviderFeatures()

	// Provider lookup should be case-insensitive
	upper := pf.Get("Anthropic")
	lower := pf.Get("anthropic")
	if upper.MaxContext != lower.MaxContext {
		t.Errorf("case-insensitive lookup failed: %d != %d", upper.MaxContext, lower.MaxContext)
	}
}

func TestFeatureDeprecationChecker(t *testing.T) {
	dc := NewDeprecationChecker()

	// Check deprecated models
	info := dc.Check("claude-3-opus-20240229")
	if info == nil {
		t.Fatal("claude-3-opus-20240229 should be deprecated")
	}
	if info.Replacement != "claude-opus-4-6" {
		t.Errorf("replacement = %q, want claude-opus-4-6", info.Replacement)
	}

	// Non-deprecated model
	info = dc.Check("claude-sonnet-4-6")
	if info != nil {
		t.Errorf("claude-sonnet-4-6 should not be deprecated, got %+v", info)
	}
}
