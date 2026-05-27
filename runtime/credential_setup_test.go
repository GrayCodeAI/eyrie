package runtime

import (
	"context"
	"strings"
	"testing"

	"github.com/GrayCodeAI/eyrie/credentials"
)

// --- ValidateKeyFormat ---

func TestValidateKeyFormat(t *testing.T) {
	tests := []struct {
		name    string
		secret  string
		wantErr bool
	}{
		{"empty", "", true},
		{"whitespace only", "   ", true},
		{"placeholder your-api-key-here", "your-api-key-here", true},
		{"placeholder YOUR_API_KEY", "YOUR_API_KEY", true},
		{"placeholder sk-xxx", "sk-xxx", true},
		{"placeholder placeholder", "placeholder", true},
		{"too short", "short", true},
		{"valid anthropic key", "sk-ant-api03-valid-key-format-12345", false},
		{"valid generic key", "sk-proj-abcdef0123456789abcdef01", false},
		{"valid long key", "gsk_abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJ", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateKeyFormat(tt.secret)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateKeyFormat(%q) error = %v, wantErr %v", tt.secret, err, tt.wantErr)
			}
		})
	}
}

// --- SetCredential ---

func TestSetCredential(t *testing.T) {
	tests := []struct {
		name    string
		envKey  string
		secret  string
		wantErr bool
	}{
		{"empty env key", "", "some-secret", true},
		{"empty secret", "ANTHROPIC_API_KEY", "", true},
		{"whitespace env key", "   ", "some-secret", true},
		{"whitespace secret", "ANTHROPIC_API_KEY", "   ", true},
		{"both empty", "", "", true},
		{"valid", "ANTHROPIC_API_KEY", "sk-ant-test-key-12345", false},
		{"valid openai", "OPENAI_API_KEY", "sk-proj-test-key-12345678", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &credentials.MapStore{}
			credentials.SetDefaultStore(store)
			t.Cleanup(func() { credentials.SetDefaultStore(nil) })

			err := SetCredential(context.Background(), tt.envKey, tt.secret)
			if (err != nil) != tt.wantErr {
				t.Fatalf("SetCredential(%q, %q) error = %v, wantErr %v", tt.envKey, tt.secret, err, tt.wantErr)
			}
		})
	}
}

func TestSetCredential_StoresSuccessfully(t *testing.T) {
	store := &credentials.MapStore{}
	credentials.SetDefaultStore(store)
	t.Cleanup(func() { credentials.SetDefaultStore(nil) })

	err := SetCredential(context.Background(), "ANTHROPIC_API_KEY", "sk-ant-test-key-12345")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the credential was stored
	if !credentials.HasSecret(context.Background(), "ANTHROPIC_API_KEY") {
		t.Fatal("expected credential to be stored")
	}
}

func TestSetCredential_TrimsWhitespace(t *testing.T) {
	store := &credentials.MapStore{}
	credentials.SetDefaultStore(store)
	t.Cleanup(func() { credentials.SetDefaultStore(nil) })

	err := SetCredential(context.Background(), "  ANTHROPIC_API_KEY  ", "  sk-ant-test-key-12345  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !credentials.HasSecret(context.Background(), "ANTHROPIC_API_KEY") {
		t.Fatal("expected credential to be stored after trimming whitespace")
	}
}

// --- ListCredentialProviders ---

func TestListCredentialProviders_NotEmpty(t *testing.T) {
	providers := ListCredentialProviders()
	if len(providers) == 0 {
		t.Fatal("expected non-empty list of credential providers")
	}
	for _, p := range providers {
		if p.ProviderID == "" {
			t.Fatal("expected non-empty provider ID in credential provider option")
		}
	}
}

func TestListCredentialProviders_ContainsKnownProviders(t *testing.T) {
	providers := ListCredentialProviders()
	ids := make(map[string]bool, len(providers))
	for _, p := range providers {
		ids[p.ProviderID] = true
	}
	for _, want := range []string{"anthropic", "openai"} {
		if !ids[want] {
			t.Fatalf("expected provider %q in list, got: %v", want, providers)
		}
	}
}

// --- ResolveCredential ---

func TestResolveCredential(t *testing.T) {
	tests := []struct {
		name          string
		secret        string
		wantFormatOK  bool
		wantInferred  string // expected inferred provider prefix, empty if none
		wantProviders bool
	}{
		{
			name:         "empty secret",
			secret:       "",
			wantFormatOK: false,
		},
		{
			name:         "placeholder secret",
			secret:       "your-api-key-here",
			wantFormatOK: false,
		},
		{
			name:         "too short",
			secret:       "abc",
			wantFormatOK: false,
		},
		{
			name:          "anthropic prefix",
			secret:        "sk-ant-api03-valid-key-format-12345",
			wantFormatOK:  true,
			wantInferred:  "anthropic",
			wantProviders: true,
		},
		{
			name:          "openai prefix",
			secret:        "sk-proj-abcdef0123456789abcdef01",
			wantFormatOK:  true,
			wantInferred:  "openai",
			wantProviders: true,
		},
		{
			name:          "generic valid key no prefix",
			secret:        "gsk_abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJ",
			wantFormatOK:  true,
			wantProviders: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResolveCredential(context.Background(), tt.secret)
			if result.FormatOK != tt.wantFormatOK {
				t.Fatalf("FormatOK = %v, want %v", result.FormatOK, tt.wantFormatOK)
			}
			if tt.wantProviders && len(result.Providers) == 0 {
				t.Fatal("expected non-empty providers list")
			}
			if tt.wantInferred != "" {
				found := false
				for _, p := range result.Providers {
					if p.Inferred && p.ProviderID == tt.wantInferred {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("expected inferred provider %q, providers: %+v", tt.wantInferred, result.Providers)
				}
			}
		})
	}
}

func TestResolveCredential_InferredRankedFirst(t *testing.T) {
	result := ResolveCredential(context.Background(), "sk-ant-api03-valid-key-format-12345")
	if !result.FormatOK {
		t.Fatal("expected FormatOK=true")
	}
	if len(result.Providers) < 2 {
		t.Fatal("expected at least 2 providers")
	}
	// First provider should be inferred
	if !result.Providers[0].Inferred {
		t.Fatalf("expected first provider to be inferred, got %+v", result.Providers[0])
	}
}

// --- InferCredentialsFromAPIKey ---

func TestInferCredentialsFromAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		secret   string
		wantZero bool
		wantID   string
	}{
		{"empty", "", true, ""},
		{"placeholder", "your-api-key-here", true, ""},
		{"too short", "short", true, ""},
		{"anthropic prefix", "sk-ant-api03-valid-key-format-12345", false, "anthropic"},
		{"openai prefix", "sk-proj-abcdef0123456789abcdef01", false, "openai"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inferences := InferCredentialsFromAPIKey(context.Background(), tt.secret)
			if tt.wantZero {
				if len(inferences) != 0 {
					t.Fatalf("expected 0 inferences, got %d", len(inferences))
				}
				return
			}
			if len(inferences) == 0 {
				t.Fatal("expected non-empty inferences")
			}
			found := false
			for _, inf := range inferences {
				if inf.ProviderID == tt.wantID {
					found = true
				}
				if inf.EnvVar == "" {
					t.Fatal("expected non-empty EnvVar in inference")
				}
			}
			if !found {
				t.Fatalf("expected inference with ProviderID=%q, got %+v", tt.wantID, inferences)
			}
		})
	}
}

// --- LocalCredentialInference ---

func TestLocalCredentialInference(t *testing.T) {
	tests := []struct {
		name       string
		providerID string
		wantErr    bool
		wantEnv    string
	}{
		{"ollama", "ollama", false, "OLLAMA_BASE_URL"},
		{"unknown provider", "nonexistent-xyz", true, ""},
		{"empty provider", "", true, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inf, err := LocalCredentialInference(tt.providerID)
			if (err != nil) != tt.wantErr {
				t.Fatalf("LocalCredentialInference(%q) error = %v, wantErr %v", tt.providerID, err, tt.wantErr)
			}
			if !tt.wantErr {
				if inf.EnvVar != tt.wantEnv {
					t.Fatalf("EnvVar = %q, want %q", inf.EnvVar, tt.wantEnv)
				}
				if inf.ProviderID == "" {
					t.Fatal("expected non-empty ProviderID")
				}
				if inf.DeploymentID == "" {
					t.Fatal("expected non-empty DeploymentID")
				}
			}
		})
	}
}

// --- ProbeCredential ---

func TestProbeCredential_EmptyArgs(t *testing.T) {
	tests := []struct {
		name   string
		envKey string
		secret string
	}{
		{"empty env key", "", "sk-ant-test"},
		{"empty secret", "ANTHROPIC_API_KEY", ""},
		{"both empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ProbeCredential(context.Background(), tt.envKey, tt.secret)
			// ProbeCredential may or may not error on empty args depending on implementation;
			// at minimum it should not panic.
			_ = err
		})
	}
}

func TestProbeCredential_UnknownEnvKey(t *testing.T) {
	// An env key not in the registry should return nil (no-op probe).
	err := ProbeCredential(context.Background(), "COMPLETELY_UNKNOWN_ENV_KEY", "some-value-12345")
	if err != nil {
		t.Fatalf("expected nil error for unknown env key, got %v", err)
	}
}

// --- CommitCredential ---

func TestCommitCredential(t *testing.T) {
	tests := []struct {
		name      string
		inference CredentialInference
		secret    string
		wantErr   bool
	}{
		{
			name:      "empty secret",
			inference: CredentialInference{ProviderID: "anthropic", EnvVar: "ANTHROPIC_API_KEY"},
			secret:    "",
			wantErr:   true,
		},
		{
			name:      "empty env var",
			inference: CredentialInference{ProviderID: "anthropic", EnvVar: ""},
			secret:    "sk-ant-test-key-12345",
			wantErr:   true,
		},
		{
			name:      "placeholder secret",
			inference: CredentialInference{ProviderID: "anthropic", EnvVar: "ANTHROPIC_API_KEY"},
			secret:    "your-api-key-here",
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CommitCredential(context.Background(), tt.inference, tt.secret)
			if (err != nil) != tt.wantErr {
				t.Fatalf("CommitCredential() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCommitCredential_NilInference(t *testing.T) {
	// Zero-value inference with valid secret should fail because EnvVar is empty.
	err := CommitCredential(context.Background(), CredentialInference{}, "sk-ant-test-key-12345")
	if err == nil {
		t.Fatal("expected error for zero-value inference")
	}
}

// --- SaveCredential ---

func TestSaveCredential_ValidatesFormat(t *testing.T) {
	store := &credentials.MapStore{}
	credentials.SetDefaultStore(store)
	t.Cleanup(func() { credentials.SetDefaultStore(nil) })

	// Empty secret should fail before storage.
	err := SaveCredential(context.Background(), CredentialInference{
		ProviderID: "anthropic", EnvVar: "ANTHROPIC_API_KEY",
	}, "")
	if err == nil {
		t.Fatal("expected error for empty secret")
	}
}

func TestSaveCredential_EmptyEnvVar(t *testing.T) {
	store := &credentials.MapStore{}
	credentials.SetDefaultStore(store)
	t.Cleanup(func() { credentials.SetDefaultStore(nil) })

	err := SaveCredential(context.Background(), CredentialInference{
		ProviderID: "anthropic", EnvVar: "",
	}, "sk-ant-test-key-12345")
	if err == nil {
		t.Fatal("expected error for empty env var")
	}
}

// --- Credential types re-exported ---

func TestCredentialTypes_Reexported(t *testing.T) {
	// Verify the re-exported types are usable.
	var inf CredentialInference
	inf.ProviderID = "test"
	if inf.ProviderID != "test" {
		t.Fatal("CredentialInference type not usable")
	}

	var opt CredentialProviderOption
	opt.ProviderID = "test"
	if opt.ProviderID != "test" {
		t.Fatal("CredentialProviderOption type not usable")
	}

	var res CredentialResolveResult
	res.FormatOK = true
	if !res.FormatOK {
		t.Fatal("CredentialResolveResult type not usable")
	}
}

// --- ResolveCredential format error ---

func TestResolveCredential_FormatErrorMessage(t *testing.T) {
	result := ResolveCredential(context.Background(), "")
	if result.FormatOK {
		t.Fatal("expected FormatOK=false")
	}
	if result.FormatError == "" {
		t.Fatal("expected non-empty FormatError")
	}
	if strings.Contains(result.FormatError, "empty") || strings.Contains(result.FormatError, "required") {
		// ok — error mentions emptiness
	} else {
		// other error text is fine too
	}
}
