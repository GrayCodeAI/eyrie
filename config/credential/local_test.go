package credential

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLocalCredentialInference_Ollama(t *testing.T) {
	inf, err := LocalCredentialInference("ollama")
	if err != nil {
		t.Fatal(err)
	}
	if inf.EnvVar != "OLLAMA_BASE_URL" || inf.DeploymentID != "ollama-local" {
		t.Fatalf("unexpected inference: %+v", inf)
	}
}

func TestCommitLocalCredential_Ollama(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"models": []map[string]string{{"name": "llama3.2:latest"}},
		})
	}))
	defer srv.Close()

	inf, err := LocalCredentialInference("ollama")
	if err != nil {
		t.Fatal(err)
	}
	if err := CommitLocalCredential(context.Background(), inf, srv.URL+"/v1"); err != nil {
		t.Fatalf("commit local: %v", err)
	}
}

func TestCommitLocalCredential_InvalidURL(t *testing.T) {
	inf, err := LocalCredentialInference("ollama")
	if err != nil {
		t.Fatal(err)
	}
	if err := CommitLocalCredential(context.Background(), inf, "not-a-url"); err == nil {
		t.Fatal("expected invalid URL error")
	}
}
