package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/GrayCodeAI/eyrie/runtime"
)

func newTestCLI() (cli, *bytes.Buffer, *bytes.Buffer) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	return cli{stdout: stdout, stderr: stderr}, stdout, stderr
}

func TestRun_NoArgsPrintsRootUsage(t *testing.T) {
	app, stdout, _ := newTestCLI()

	if err := app.run(nil); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	out := stdout.String()
	for _, want := range []string{"eyrie", "Quick start", "preflight", "models", "select provider"} {
		if !strings.Contains(out, want) {
			t.Fatalf("root usage missing %q\n%s", want, out)
		}
	}
}

func TestRun_ProvidersJSON(t *testing.T) {
	app, stdout, _ := newTestCLI()

	if err := app.run([]string{"providers", "--json"}); err != nil {
		t.Fatalf("run(providers --json) error = %v", err)
	}

	var providers []runtime.CredentialProviderOption
	if err := json.Unmarshal(stdout.Bytes(), &providers); err != nil {
		t.Fatalf("providers JSON decode: %v\n%s", err, stdout.String())
	}
	if len(providers) == 0 {
		t.Fatal("expected provider rows")
	}
}

func TestRun_PreflightJSONAlwaysEmitsJSON(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	app, stdout, _ := newTestCLI()
	err := app.run([]string{"preflight", "--json"})
	if err == nil {
		t.Fatal("expected preflight to fail in a fresh temp home")
	}

	var report runtime.PreflightReport
	if decodeErr := json.Unmarshal(stdout.Bytes(), &report); decodeErr != nil {
		t.Fatalf("preflight JSON decode: %v\n%s", decodeErr, stdout.String())
	}
	if len(report.Checks) == 0 {
		t.Fatal("expected preflight checks")
	}
}

func TestRun_SelectProviderPersistsActiveProvider(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	app, stdout, _ := newTestCLI()
	if err := app.run([]string{"select", "provider", "anthropic"}); err != nil {
		t.Fatalf("run(select provider) error = %v", err)
	}

	if got := strings.TrimSpace(runtime.ActiveProvider(nil)); got != "anthropic" {
		t.Fatalf("active provider = %q, want anthropic", got)
	}
	if !strings.Contains(stdout.String(), "Active provider set to anthropic") {
		t.Fatalf("unexpected output: %s", stdout.String())
	}
}

func TestRun_ModelsWithoutProviderReturnsActionableError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	app, _, _ := newTestCLI()
	err := app.run([]string{"models"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "select provider") {
		t.Fatalf("expected select-provider hint, got: %v", err)
	}
}

func TestRun_UnknownCommandSuggestsHelp(t *testing.T) {
	app, _, _ := newTestCLI()

	err := app.run([]string{"not-a-command"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "eyrie help") {
		t.Fatalf("expected help suggestion, got: %v", err)
	}
}
