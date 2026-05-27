package catalog

import (
	"testing"
)

func TestCredentials_Env_FiltersEmpty(t *testing.T) {
	c := Credentials{
		APIKeys: map[string]string{
			"OPENAI_API_KEY":    "sk-test",
			"EMPTY_KEY":         "",
			"":                  "orphan-value",
			"ANTHROPIC_API_KEY": "sk-ant-test",
		},
	}
	env := c.Env()
	if len(env) != 2 {
		t.Fatalf("expected 2 entries, got %d: %v", len(env), env)
	}
	if env["OPENAI_API_KEY"] != "sk-test" {
		t.Errorf("OPENAI_API_KEY = %q", env["OPENAI_API_KEY"])
	}
	if env["ANTHROPIC_API_KEY"] != "sk-ant-test" {
		t.Errorf("ANTHROPIC_API_KEY = %q", env["ANTHROPIC_API_KEY"])
	}
	if _, ok := env["EMPTY_KEY"]; ok {
		t.Error("EMPTY_KEY should be filtered out")
	}
	if _, ok := env[""]; ok {
		t.Error("empty key should be filtered out")
	}
}

func TestCredentials_Env_ReturnsCopy(t *testing.T) {
	c := Credentials{
		APIKeys: map[string]string{"KEY": "val"},
	}
	env := c.Env()
	env["KEY"] = "mutated"
	if c.APIKeys["KEY"] != "val" {
		t.Error("Env() should return a copy, not a reference")
	}
}

func TestCredentials_Env_NilMap(t *testing.T) {
	var c Credentials
	env := c.Env()
	if len(env) != 0 {
		t.Fatalf("expected empty env from nil map, got %d", len(env))
	}
}

func TestCredentials_Merge_AddsKeys(t *testing.T) {
	c := Credentials{
		APIKeys: map[string]string{"A": "1"},
	}
	c.Merge(Credentials{
		APIKeys: map[string]string{"B": "2"},
	})
	if c.APIKeys["A"] != "1" || c.APIKeys["B"] != "2" {
		t.Fatalf("merged keys: %v", c.APIKeys)
	}
}

func TestCredentials_Merge_OverwritesExisting(t *testing.T) {
	c := Credentials{
		APIKeys: map[string]string{"KEY": "old"},
	}
	c.Merge(Credentials{
		APIKeys: map[string]string{"KEY": "new"},
	})
	if c.APIKeys["KEY"] != "new" {
		t.Fatalf("expected overwritten value, got %q", c.APIKeys["KEY"])
	}
}

func TestCredentials_Merge_InitializesNilMap(t *testing.T) {
	var c Credentials
	c.Merge(Credentials{
		APIKeys: map[string]string{"KEY": "val"},
	})
	if c.APIKeys == nil {
		t.Fatal("Merge should initialize nil map")
	}
	if c.APIKeys["KEY"] != "val" {
		t.Fatalf("expected val, got %q", c.APIKeys["KEY"])
	}
}

func TestCredentials_Merge_SkipsEmptyKeys(t *testing.T) {
	c := Credentials{}
	c.Merge(Credentials{
		APIKeys: map[string]string{
			"":     "orphan",
			"GOOD": "val",
		},
	})
	if _, ok := c.APIKeys[""]; ok {
		t.Error("empty key should not be merged")
	}
	if c.APIKeys["GOOD"] != "val" {
		t.Error("GOOD key should be merged")
	}
}
