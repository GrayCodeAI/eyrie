package live

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

const (
	DefaultOpenRouterBaseURL  = "https://openrouter.ai/api/v1"
	DefaultCanopyWaveBaseURL  = "https://inference.canopywave.io/v1"
	DefaultOpenAIBaseURL      = "https://api.openai.com/v1"
	DefaultGrokBaseURL        = "https://api.x.ai/v1"
	DefaultOpenCodeGoBaseURL  = "https://api.opencodego.ai/v1"
)

// FetchFunc lists models from a live provider API.
type FetchFunc func(env map[string]string) ([]Entry, error)

// Registry maps fetcher keys to implementations.
var Registry = map[string]FetchFunc{
	"anthropic":  FetchAnthropic,
	"openai":     FetchOpenAI,
	"gemini":     FetchGemini,
	"openrouter": FetchOpenRouter,
	"grok":       FetchGrok,
	"canopywave": FetchCanopyWave,
	"opencodego": FetchOpenCodeGo,
	"ollama":     FetchOllama,
}

// Fetch runs a registered live fetcher.
func Fetch(key string, env map[string]string) ([]Entry, error) {
	fn, ok := Registry[key]
	if !ok {
		return nil, fmt.Errorf("live: unknown fetcher %q", key)
	}
	return fn(env)
}

type openAICompatModel struct {
	ID                  string `json:"id"`
	ContextLength       *int   `json:"context_length"`
	MaxCompletionTokens *int   `json:"max_completion_tokens"`
	Pricing             *struct {
		Prompt     interface{} `json:"prompt"`
		Completion interface{} `json:"completion"`
	} `json:"pricing"`
}

type openRouterModel struct {
	ID            string `json:"id"`
	ContextLength *int   `json:"context_length"`
	TopProvider   *struct {
		ContextLength       *int `json:"context_length"`
		MaxCompletionTokens *int `json:"max_completion_tokens"`
	} `json:"top_provider"`
	Pricing *struct {
		Prompt     interface{} `json:"prompt"`
		Completion interface{} `json:"completion"`
	} `json:"pricing"`
}

func asFloat(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case string:
		var f float64
		_, _ = fmt.Sscanf(n, "%f", &f)
		return f
	}
	return 0
}

func intOr(p *int, def int) int {
	if p != nil {
		return *p
	}
	return def
}

func fetchOpenAICompatModels(baseURL, apiKey, authHeader string) ([]Entry, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, nil
	}
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("live: missing base URL")
	}
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, baseURL+"/models", nil)
	if authHeader == "x-api-key" {
		req.Header.Set("x-api-key", apiKey)
	} else {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "eyrie-model-catalog/1.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("live model fetch failed (%d)", resp.StatusCode)
	}

	var payload struct {
		Data []openAICompatModel `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	var entries []Entry
	for _, raw := range payload.Data {
		id := strings.TrimSpace(raw.ID)
		if id == "" {
			continue
		}
		var inPrice, outPrice float64
		if raw.Pricing != nil {
			inPrice = asFloat(raw.Pricing.Prompt) * 1_000_000
			outPrice = asFloat(raw.Pricing.Completion) * 1_000_000
		}
		entries = append(entries, Entry{
			ID: id, InputPricePer1M: inPrice, OutputPricePer1M: outPrice,
			ContextWindow: intOr(raw.ContextLength, 128000),
			MaxOutput:     intOr(raw.MaxCompletionTokens, 16384),
			DisplayName:   id,
		})
	}
	return entries, nil
}

func envOr(env map[string]string, key, def string) string {
	if v := strings.TrimSpace(env[key]); v != "" {
		return v
	}
	return def
}

func FetchOpenAI(env map[string]string) ([]Entry, error) {
	return fetchOpenAICompatModels(
		envOr(env, "OPENAI_BASE_URL", DefaultOpenAIBaseURL),
		env["OPENAI_API_KEY"], "Bearer",
	)
}

func FetchGrok(env map[string]string) ([]Entry, error) {
	key := strings.TrimSpace(env["XAI_API_KEY"])
	if key == "" {
		key = strings.TrimSpace(env["GROK_API_KEY"])
	}
	return fetchOpenAICompatModels(
		envOr(env, "GROK_BASE_URL", envOr(env, "XAI_BASE_URL", DefaultGrokBaseURL)),
		key, "Bearer",
	)
}

func FetchCanopyWave(env map[string]string) ([]Entry, error) {
	return fetchOpenAICompatModels(
		envOr(env, "CANOPYWAVE_BASE_URL", DefaultCanopyWaveBaseURL),
		env["CANOPYWAVE_API_KEY"], "Bearer",
	)
}

func FetchOpenCodeGo(env map[string]string) ([]Entry, error) {
	return fetchOpenAICompatModels(
		envOr(env, "OPENCODEGO_BASE_URL", DefaultOpenCodeGoBaseURL),
		env["OPENCODEGO_API_KEY"], "Bearer",
	)
}

func FetchOpenRouter(env map[string]string) ([]Entry, error) {
	apiKey := strings.TrimSpace(env["OPENROUTER_API_KEY"])
	if apiKey == "" {
		return nil, nil
	}
	baseURL := strings.TrimRight(envOr(env, "OPENROUTER_BASE_URL", DefaultOpenRouterBaseURL), "/")
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, baseURL+"/models", nil)
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "eyrie-model-catalog/1.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openrouter model fetch failed (%d)", resp.StatusCode)
	}
	var payload struct {
		Data []openRouterModel `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	var entries []Entry
	for _, raw := range payload.Data {
		id := strings.TrimSpace(raw.ID)
		if id == "" {
			continue
		}
		ctx := 128000
		if raw.ContextLength != nil {
			ctx = *raw.ContextLength
		} else if raw.TopProvider != nil && raw.TopProvider.ContextLength != nil {
			ctx = *raw.TopProvider.ContextLength
		}
		maxOut := 16384
		if raw.TopProvider != nil && raw.TopProvider.MaxCompletionTokens != nil {
			maxOut = *raw.TopProvider.MaxCompletionTokens
		}
		var inPrice, outPrice float64
		if raw.Pricing != nil {
			inPrice = asFloat(raw.Pricing.Prompt) * 1_000_000
			outPrice = asFloat(raw.Pricing.Completion) * 1_000_000
		}
		entries = append(entries, Entry{
			ID: id, InputPricePer1M: inPrice, OutputPricePer1M: outPrice,
			ContextWindow: ctx, MaxOutput: maxOut, DisplayName: id,
		})
	}
	return entries, nil
}

func FetchAnthropic(env map[string]string) ([]Entry, error) {
	apiKey := strings.TrimSpace(env["ANTHROPIC_API_KEY"])
	if apiKey == "" {
		return nil, nil
	}
	baseURL := strings.TrimRight(envOr(env, "ANTHROPIC_BASE_URL", "https://api.anthropic.com/v1"), "/")
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, baseURL+"/models", nil)
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "eyrie-model-catalog/1.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("anthropic model fetch failed (%d)", resp.StatusCode)
	}
	var payload struct {
		Data []struct {
			ID          string `json:"id"`
			DisplayName string `json:"display_name"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	var entries []Entry
	for _, raw := range payload.Data {
		id := strings.TrimSpace(raw.ID)
		if id == "" {
			continue
		}
		label := strings.TrimSpace(raw.DisplayName)
		if label == "" {
			label = id
		}
		entries = append(entries, Entry{
			ID: id, DisplayName: label, ContextWindow: 200000, MaxOutput: 8192,
		})
	}
	return entries, nil
}

func FetchGemini(env map[string]string) ([]Entry, error) {
	apiKey := strings.TrimSpace(env["GEMINI_API_KEY"])
	if apiKey == "" {
		return nil, nil
	}
	url := "https://generativelanguage.googleapis.com/v1beta/models?key=" + apiKey
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "eyrie-model-catalog/1.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gemini model fetch failed (%d)", resp.StatusCode)
	}
	var payload struct {
		Models []struct {
			Name                       string `json:"name"`
			DisplayName                string `json:"displayName"`
			InputTokenLimit            int    `json:"inputTokenLimit"`
			OutputTokenLimit           int    `json:"outputTokenLimit"`
			SupportedGenerationMethods []string `json:"supportedGenerationMethods"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	var entries []Entry
	for _, raw := range payload.Models {
		name := strings.TrimSpace(raw.Name)
		if name == "" {
			continue
		}
		supportsGen := false
		for _, m := range raw.SupportedGenerationMethods {
			if m == "generateContent" {
				supportsGen = true
				break
			}
		}
		if !supportsGen {
			continue
		}
		id := strings.TrimPrefix(name, "models/")
		label := strings.TrimSpace(raw.DisplayName)
		if label == "" {
			label = id
		}
		ctxWin := raw.InputTokenLimit
		if ctxWin <= 0 {
			ctxWin = 128000
		}
		maxOut := raw.OutputTokenLimit
		if maxOut <= 0 {
			maxOut = 8192
		}
		entries = append(entries, Entry{
			ID: id, DisplayName: label, ContextWindow: ctxWin, MaxOutput: maxOut,
		})
	}
	return entries, nil
}

func FetchOllama(env map[string]string) ([]Entry, error) {
	baseURL := strings.TrimSpace(env["OLLAMA_BASE_URL"])
	if baseURL == "" {
		return nil, nil
	}
	root := strings.TrimRight(baseURL, "/")
	if strings.HasSuffix(root, "/v1") {
		root = strings.TrimSuffix(root, "/v1")
	}
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, root+"/api/tags", nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "eyrie-model-catalog/1.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama model fetch failed (%d)", resp.StatusCode)
	}
	var payload struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	var entries []Entry
	for _, raw := range payload.Models {
		id := strings.TrimSpace(raw.Name)
		if id == "" {
			continue
		}
		entries = append(entries, Entry{
			ID: id, DisplayName: id, ContextWindow: 32000, MaxOutput: 4096,
		})
	}
	return entries, nil
}
