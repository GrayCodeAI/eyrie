package catalog

import "testing"

func TestModelOwner_FromLiveMetadata(t *testing.T) {
	entry := ModelCatalogEntry{
		ID:           "moonshotai/kimi-k2.6",
		LiveMetadata: []byte(`{"owned_by":"moonshotai"}`),
	}
	if got := ModelOwner(entry); got != "moonshotai" {
		t.Fatalf("owner = %q", got)
	}
}

func TestModelOwner_FromIDPrefix(t *testing.T) {
	entry := ModelCatalogEntry{ID: "zai/glm-5.1"}
	if got := ModelOwner(entry); got != "zai" {
		t.Fatalf("owner = %q", got)
	}
}

func TestModelOwner_ExplicitField(t *testing.T) {
	entry := ModelCatalogEntry{ID: "gpt-4o", Owner: "openai"}
	if got := ModelOwner(entry); got != "openai" {
		t.Fatalf("owner = %q", got)
	}
}

func TestDisplayModelLabel_StripsOpenRouterLatestPrefix(t *testing.T) {
	got := DisplayModelLabel("~anthropic/claude-haiku-latest", "")
	if got != "anthropic/claude-haiku-latest" {
		t.Fatalf("label = %q", got)
	}
}

func TestDisplayModelOwner_StripsOpenRouterLatestPrefix(t *testing.T) {
	got := DisplayModelOwner("", "~anthropic/claude-haiku-latest")
	if got != "anthropic" {
		t.Fatalf("owner = %q", got)
	}
}
