package engine

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/GrayCodeAI/eyrie/credentials"
)

func TestControlPlaneUsesInjectedCredentialStore(t *testing.T) {
	ctx := context.Background()
	store := &credentials.MapStore{}
	eng, err := New(Options{
		SecretStore: store,
		StateDir:    t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}

	providers := eng.CredentialProviders(ctx)
	if len(providers) == 0 {
		t.Fatal("expected registry-backed credential providers")
	}
	resolved := eng.ResolveCredential(ctx, "short")
	if resolved.FormatOK {
		t.Fatal("expected invalid credential format")
	}

	if err := store.Set(ctx, credentials.AccountForEnv("OPENAI_API_KEY"), "sk-injected-secret"); err != nil {
		t.Fatal(err)
	}
	status, err := eng.CredentialStatus(ctx, "openai")
	if err != nil {
		t.Fatal(err)
	}
	if !status.Configured || status.Masked != "••••••••cret" {
		t.Fatalf("unexpected safe status: %+v", status)
	}
	for _, gateway := range eng.Gateways(ctx) {
		if gateway.ID == "openai" && !gateway.CredentialConfigured {
			t.Fatal("gateway status ignored injected credential store")
		}
	}

	if eng.catalogPath != filepath.Join(filepath.Dir(eng.providerConfigPath), "model_catalog.json") {
		// Both paths must derive from the injected StateDir. The exact assertion
		// catches accidental fallback to process-global Hawk paths.
		t.Fatalf("control-plane paths escaped injected state dir: catalog=%q provider=%q", eng.catalogPath, eng.providerConfigPath)
	}
}

func TestMaskedCredentialNeverReturnsFullSecret(t *testing.T) {
	secret := "sk-sensitive-value"
	masked := maskedCredential(secret)
	if masked == secret || masked == "" {
		t.Fatalf("unsafe masked value %q", masked)
	}
}

func TestEffectiveSelectionUsesEngineOwnedState(t *testing.T) {
	ctx := context.Background()
	store := &credentials.MapStore{}
	stateDir := t.TempDir()
	eng, err := New(Options{SecretStore: store, StateDir: stateDir})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Set(ctx, credentials.AccountForEnv("OPENAI_API_KEY"), "sk-injected-secret"); err != nil {
		t.Fatal(err)
	}
	if err := eng.SetActiveProvider(ctx, "openai"); err != nil {
		t.Fatal(err)
	}

	selection := eng.EffectiveSelection(ctx, SelectionOptions{ModelOverride: "gpt-test"})
	if selection.Provider != "openai" || selection.Model != "gpt-test" {
		t.Fatalf("unexpected selection: %+v", selection)
	}
	if !selection.HasConfiguredDeployment {
		t.Fatal("selection ignored injected credential store")
	}
}

func TestHostControlUsesInjectedStoreAndPaths(t *testing.T) {
	ctx := context.Background()
	store := &credentials.MapStore{}
	eng, err := New(Options{SecretStore: store, StateDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if err := eng.SaveCredentialEnv(ctx, "OPENAI_API_KEY", "sk-injected-value"); err != nil {
		t.Fatal(err)
	}
	if !eng.HasCredentialEnv(ctx, "OPENAI_API_KEY") {
		t.Fatal("host control ignored injected credential store")
	}
	if err := eng.SetGatewayRegion(ctx, "zai_coding", "international"); err != nil {
		t.Fatal(err)
	}
	label, required := eng.GatewayRegion("zai_coding")
	if label == "" || required {
		t.Fatalf("unexpected gateway region label=%q required=%v", label, required)
	}
}
