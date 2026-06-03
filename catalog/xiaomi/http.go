package xiaomi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ProbeOpenAIModels GETs {baseURL}/models using api-key auth, then Bearer on 401.
func ProbeOpenAIModels(ctx context.Context, baseURL, apiKey string) error {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return fmt.Errorf("xiaomi probe: missing base URL")
	}
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return fmt.Errorf("xiaomi probe: missing API key")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/models", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "eyrie-model-catalog/1.0")
	setAPIKeyAuth(req, apiKey)

	status, err := doProbe(req)
	if err != nil {
		return err
	}
	if status == http.StatusUnauthorized {
		req2, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/models", nil)
		if err != nil {
			return err
		}
		req2.Header.Set("Accept", "application/json")
		req2.Header.Set("User-Agent", "eyrie-model-catalog/1.0")
		req2.Header.Set("Authorization", "Bearer "+apiKey)
		status, err = doProbe(req2)
		if err != nil {
			return err
		}
	}
	return probeStatusErr(status)
}

func setAPIKeyAuth(req *http.Request, apiKey string) {
	req.Header.Set("api-key", apiKey)
}

func doProbe(req *http.Request) (int, error) {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("xiaomi probe: network error: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode, nil
}

func probeStatusErr(status int) error {
	if status >= 200 && status < 300 {
		return nil
	}
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

// SetMimoRequestAuth applies MiMo-preferred auth (api-key header).
func SetMimoRequestAuth(req *http.Request, apiKey string) {
	setAPIKeyAuth(req, apiKey)
}

// FetchOpenAIModelsJSON GETs /models and returns raw model objects from the OpenAI list response.
func FetchOpenAIModelsJSON(ctx context.Context, baseURL, apiKey string) ([]json.RawMessage, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	apiKey = strings.TrimSpace(apiKey)
	if baseURL == "" || apiKey == "" {
		return nil, fmt.Errorf("xiaomi: base URL and API key required")
	}
	body, status, err := getModelsBody(ctx, baseURL, apiKey)
	if err != nil {
		return nil, err
	}
	if status == http.StatusUnauthorized {
		body, status, err = getModelsBodyBearer(ctx, baseURL, apiKey)
		if err != nil {
			return nil, err
		}
	}
	if status < 200 || status >= 300 {
		return nil, probeStatusErr(status)
	}
	var payload struct {
		Data []json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	return payload.Data, nil
}

func getModelsBody(ctx context.Context, baseURL, apiKey string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/models", nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "eyrie-model-catalog/1.0")
	SetMimoRequestAuth(req, apiKey)
	return doModelsRequest(req)
}

func getModelsBodyBearer(ctx context.Context, baseURL, apiKey string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/models", nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "eyrie-model-catalog/1.0")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	return doModelsRequest(req)
}

func doModelsRequest(req *http.Request) ([]byte, int, error) {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return body, resp.StatusCode, nil
}

// IsRetryableHTTPStatus reports whether chat may retry via Anthropic compatibility.
func IsRetryableHTTPStatus(status int) bool {
	switch status {
	case http.StatusUnauthorized, http.StatusForbidden,
		http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return status >= 500
	}
}