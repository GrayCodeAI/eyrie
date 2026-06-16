//nolint:errcheck
package client

import (
	"context"
	"testing"

	"github.com/GrayCodeAI/eyrie/credentials"
)

func TestGetOrCreateProvider_VertexUsesAnthropicVertexClient(t *testing.T) {
	store := &credentials.MapStore{}
	credentials.SetDefaultStore(store)
	t.Cleanup(func() { credentials.SetDefaultStore(nil) })

	ctx := context.Background()
	if err := store.Set(ctx, credentials.AccountForEnv("VERTEX_PROJECT_ID"), "my-project"); err != nil {
		t.Fatalf("set VERTEX_PROJECT_ID: %v", err)
	}
	if err := store.Set(ctx, credentials.AccountForEnv("VERTEX_REGION"), "us-east1"); err != nil {
		t.Fatalf("set VERTEX_REGION: %v", err)
	}

	c := Client(&EyrieConfig{Provider: "vertex", APIKey: "test-bearer-token"})
	p, err := c.getOrCreateProvider("vertex")
	if err != nil {
		t.Fatalf("getOrCreateProvider: %v", err)
	}
	vc, ok := p.(*VertexClient)
	if !ok {
		t.Fatalf("provider type = %T, want *VertexClient (regression: registry was creating a GeminiClient for ProviderTypeVertex)", p)
	}
	if vc.projectID != "my-project" {
		t.Errorf("projectID = %q, want %q", vc.projectID, "my-project")
	}
	if vc.region != "us-east1" {
		t.Errorf("region = %q, want %q", vc.region, "us-east1")
	}
	if vc.token != "test-bearer-token" {
		t.Errorf("token = %q, want %q", vc.token, "test-bearer-token")
	}
	if got := vc.baseURL(); got != "https://us-east1-aiplatform.googleapis.com/v1/projects/my-project/locations/us-east1/publishers/anthropic/models" {
		t.Errorf("baseURL() = %q, want Anthropic-on-Vertex URL", got)
	}
}

func TestGetOrCreateProvider_VertexRegionDefaultsToUsCentral1(t *testing.T) {
	store := &credentials.MapStore{}
	credentials.SetDefaultStore(store)
	t.Cleanup(func() { credentials.SetDefaultStore(nil) })

	ctx := context.Background()
	if err := store.Set(ctx, credentials.AccountForEnv("VERTEX_PROJECT_ID"), "my-project"); err != nil {
		t.Fatalf("set VERTEX_PROJECT_ID: %v", err)
	}

	c := Client(&EyrieConfig{Provider: "vertex", APIKey: "test-token"})
	p, err := c.getOrCreateProvider("vertex")
	if err != nil {
		t.Fatalf("getOrCreateProvider: %v", err)
	}
	vc, ok := p.(*VertexClient)
	if !ok {
		t.Fatalf("provider type = %T, want *VertexClient", p)
	}
	if vc.region != "us-central1" {
		t.Errorf("region = %q, want default %q", vc.region, "us-central1")
	}
}

func TestGetOrCreateProvider_VertexRequiresProjectID(t *testing.T) {
	store := &credentials.MapStore{}
	credentials.SetDefaultStore(store)
	t.Cleanup(func() { credentials.SetDefaultStore(nil) })

	c := Client(&EyrieConfig{Provider: "vertex", APIKey: "test-token"})
	_, err := c.getOrCreateProvider("vertex")
	if err == nil {
		t.Fatal("expected error when VERTEX_PROJECT_ID is missing, got nil")
	}
	if got := err.Error(); got != "eyrie: vertex requires VERTEX_PROJECT_ID" {
		t.Errorf("error = %q, want %q", got, "eyrie: vertex requires VERTEX_PROJECT_ID")
	}
}
