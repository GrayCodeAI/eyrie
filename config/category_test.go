package config

import "testing"

func TestDefaultCategories_AllPresent(t *testing.T) {
	t.Parallel()
	cats := DefaultCategories()
	expected := []ModelCategory{
		CategoryDeep, CategoryQuick, CategoryUltraBrain,
		CategoryVisualEngineering, CategoryArtistry, CategoryWriting,
		CategoryUnspecifiedLow, CategoryUnspecifiedHigh,
	}
	for _, cat := range expected {
		if _, ok := cats[cat]; !ok {
			t.Errorf("missing default category: %s", cat)
		}
	}
}

func TestCategoryRegistry_Resolve(t *testing.T) {
	t.Parallel()
	ResetCategoryRegistry()
	defer ResetCategoryRegistry()

	reg := GetCategoryRegistry()

	cfg := reg.Resolve(CategoryDeep)
	if cfg.Model != "claude-sonnet-4-6" {
		t.Errorf("expected claude-sonnet-4-6 for deep, got %s", cfg.Model)
	}
	if cfg.Temperature != 0.3 {
		t.Errorf("expected 0.3 temp for deep, got %f", cfg.Temperature)
	}
}

func TestCategoryRegistry_ResolveFallback(t *testing.T) {
	t.Parallel()
	ResetCategoryRegistry()
	defer ResetCategoryRegistry()

	reg := GetCategoryRegistry()

	// Unknown category should fall back to unspecified-high
	cfg := reg.Resolve(ModelCategory("nonexistent"))
	expected := reg.Resolve(CategoryUnspecifiedHigh)
	if cfg.Model != expected.Model {
		t.Errorf("expected fallback to unspecified-high model %s, got %s", expected.Model, cfg.Model)
	}
}

func TestCategoryRegistry_Override(t *testing.T) {
	t.Parallel()
	ResetCategoryRegistry()
	defer ResetCategoryRegistry()

	reg := GetCategoryRegistry()
	reg.SetCategory(CategoryDeep, CategoryConfig{
		Model:       "custom-model",
		Temperature: 0.9,
	})

	cfg := reg.Resolve(CategoryDeep)
	if cfg.Model != "custom-model" {
		t.Errorf("expected custom-model after override, got %s", cfg.Model)
	}
}

func TestCategoryRegistry_ResolveWithDefaults(t *testing.T) {
	t.Parallel()
	ResetCategoryRegistry()
	defer ResetCategoryRegistry()

	reg := GetCategoryRegistry()
	cfg := reg.ResolveWithDefaults(CategoryQuick, "fallback-model")
	if cfg.Model != "claude-haiku-4-5-20251001" {
		t.Errorf("expected haiku for quick, got %s", cfg.Model)
	}

	// With empty category, should use default
	emptyReg := &CategoryRegistry{
		categories: map[ModelCategory]CategoryConfig{
			CategoryUnspecifiedHigh: {Model: "fallback"},
		},
	}
	cfg = emptyReg.ResolveWithDefaults(CategoryDeep, "fallback")
	if cfg.Model != "fallback" {
		t.Errorf("expected fallback model, got %s", cfg.Model)
	}
}

func TestCategoryRegistry_AllCategories(t *testing.T) {
	t.Parallel()
	ResetCategoryRegistry()
	defer ResetCategoryRegistry()

	reg := GetCategoryRegistry()
	all := reg.AllCategories()
	if len(all) != 8 {
		t.Errorf("expected 8 categories, got %d", len(all))
	}
}

func TestResolveCategory_Convenience(t *testing.T) {
	t.Parallel()
	ResetCategoryRegistry()
	defer ResetCategoryRegistry()

	cfg := ResolveCategory(CategoryUltraBrain)
	if cfg.Model != "claude-opus-4-8" {
		t.Errorf("expected opus for ultrabrain, got %s", cfg.Model)
	}
}

func TestCategoryDescriptions(t *testing.T) {
	t.Parallel()
	cats := DefaultCategories()
	for cat, cfg := range cats {
		if cfg.Description == "" {
			t.Errorf("category %s has no description", cat)
		}
	}
}
