package runtime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GrayCodeAI/eyrie/credentials"
)

func TestPreflight_ReportsMissingModel(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HAWK_CONFIG_DIR", dir)
	t.Setenv("EYRIE_MODEL_CATALOG_PATH", filepath.Join(dir, "missing.json"))
	if err := os.WriteFile(filepath.Join(dir, "provider.json"), []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := &credentials.MapStore{}
	credentials.SetDefaultStore(store)
	t.Cleanup(func() { credentials.SetDefaultStore(nil) })

	r := Preflight(context.Background())
	found := false
	for _, c := range r.Checks {
		if c.Name == "model" && c.Status == PreflightFail {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected model fail check: %+v", r.Checks)
	}
	if r.Ready {
		t.Fatal("expected not ready")
	}
}

func TestFormatPreflightReport(t *testing.T) {
	out := FormatPreflightReport(PreflightReport{
		Ready: false,
		Checks: []PreflightCheck{
			{Name: "model", Status: PreflightFail, Detail: "none"},
		},
	})
	if !strings.Contains(out, "setup incomplete") {
		t.Fatal(out)
	}
}
