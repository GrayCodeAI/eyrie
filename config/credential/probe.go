package credential

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/GrayCodeAI/eyrie/catalog"
	"github.com/GrayCodeAI/eyrie/catalog/registry"
)

const credentialProbeTimeout = 8 * time.Second

// ProbeCredential verifies a key against the provider API when a probe is configured.
func ProbeCredential(ctx context.Context, envKey, secret string) error {
	secret = strings.TrimSpace(secret)
	envKey = strings.TrimSpace(envKey)
	if secret == "" || envKey == "" {
		return fmt.Errorf("credential probe: key and env var required")
	}
	spec, ok := registry.DefaultRegistry.GetByEnv(envKey)
	if !ok {
		return nil
	}
	if spec.ProbeKind == "" || spec.ProbeKind == registry.ProbeNone {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, credentialProbeTimeout)
	defer cancel()

	switch spec.ProbeKind {
	case registry.ProbeOpenAIModels:
		return probeOpenAIModels(ctx, spec.ProbeBaseURL, secret)
	case registry.ProbeAnthropic:
		return probeAnthropic(ctx, secret)
	case registry.ProbeGemini:
		return probeGemini(ctx, secret)
	case registry.ProbeOllama:
		err := probeOllama(ctx, secret)
		if err != nil {
			return FormatOllamaConnectError(err)
		}
		return nil
	default:
		return nil
	}
}

func probeOllama(ctx context.Context, baseURL string) error {
	models, err := catalog.FetchOllamaModels(map[string]string{"OLLAMA_BASE_URL": baseURL})
	if err != nil {
		return FormatOllamaConnectError(err)
	}
	if len(models) == 0 {
		return errOllamaNoModels
	}
	return nil
}

func probeOpenAIModels(ctx context.Context, baseURL, secret string) error {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return fmt.Errorf("credential probe: missing base URL")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/models", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+secret)
	return doProbeRequest(req)
}

func probeAnthropic(ctx context.Context, secret string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.anthropic.com/v1/models", nil)
	if err != nil {
		return err
	}
	req.Header.Set("x-api-key", secret)
	req.Header.Set("anthropic-version", "2023-06-01")
	return doProbeRequest(req)
}

func probeGemini(ctx context.Context, secret string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://generativelanguage.googleapis.com/v1beta/models", nil)
	if err != nil {
		return err
	}
	req.Header.Set("x-goog-api-key", secret)
	return doProbeRequest(req)
}

func doProbeRequest(req *http.Request) error {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("credential probe: network error: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	_, _ = io.ReadAll(io.LimitReader(resp.Body, 512))
	return probeHTTPError(resp.StatusCode)
}

func probeHTTPError(status int) error {
	switch status {
	case http.StatusUnauthorized, http.StatusForbidden:
		return fmt.Errorf("credential probe failed: invalid API key (HTTP %d)", status)
	case http.StatusTooManyRequests:
		return fmt.Errorf("credential probe failed: rate limited — try again shortly")
	default:
		if status >= 500 {
			return fmt.Errorf("credential probe failed: provider unavailable (HTTP %d)", status)
		}
		return fmt.Errorf("credential probe failed: HTTP %d", status)
	}
}
