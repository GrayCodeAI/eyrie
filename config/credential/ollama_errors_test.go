package credential

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFormatOllamaConnectError_ConnectionRefused(t *testing.T) {
	err := FormatOllamaConnectError(context.DeadlineExceeded)
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); got == context.DeadlineExceeded.Error() {
		t.Fatalf("expected friendly timeout message, got %q", got)
	}
}

func TestCommitLocalCredential_OllamaNoModels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"models": []any{}})
	}))
	defer srv.Close()

	inf, err := LocalCredentialInference("ollama")
	if err != nil {
		t.Fatal(err)
	}
	if err := CommitLocalCredential(context.Background(), inf, srv.URL+"/v1"); err == nil {
		t.Fatal("expected error when ollama has no models")
	}
}
