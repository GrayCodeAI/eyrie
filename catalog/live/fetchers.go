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
	DefaultOpenRouterBaseURL = "https://openrouter.ai/api/v1"
	DefaultCanopyWaveBaseURL = "https://inference.canopywave.io/v1"
	DefaultZAIBaseURL        = "https://api.z.ai/api/paas/v4"
	DefaultOpenAIBaseURL     = "https://api.openai.com/v1"
	DefaultGrokBaseURL       = "https://api.x.ai/v1"
	DefaultOpenCodeGoBaseURL = "https://api.opencodego.ai/v1"
	DefaultKimiBaseURL       = "https://api.moonshot.ai/v1"
	DefaultXiaomiBaseURL     = "https://api.xiaomimimo.com/v1"
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
	"z-ai":       FetchZAI,
	"canopywave": FetchCanopyWave,
	"opencodego": FetchOpenCodeGo,
	"kimi":       FetchKimi,
	"xiaomi":     FetchXiaomi,
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

type listModelJSON struct {
	ID                   string   `json:"id"`
	Name                 string   `json:"name"`
	Title                string   `json:"title"`
	DisplayName          string   `json:"display_name"`
	Description          string   `json:"description"`
	ContextLength        *int     `json:"context_length"`
	ContextSize          *int     `json:"context_size"`
	MaxCompletionTokens  *int     `json:"max_completion_tokens"`
	MaxOutputTokens      *int     `json:"max_output_tokens"`
	InputTokenPricePerM  *float64 `json:"input_token_price_per_m"`
	OutputTokenPricePerM *float64 `json:"output_token_price_per_m"`
	Features             []string `json:"features"`
	Tags                 []string `json:"tags"`
	OwnedBy              string   `json:"owned_by"`
	Status               *int     `json:"status"`
	Pricing              *struct {
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

func ownerFromModelID(id string) string {
	id = strings.TrimSpace(id)
	if i := strings.Index(id, "/"); i > 0 {
		return id[:i]
	}
	return ""
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
		Data []json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	var entries []Entry
	for _, raw := range payload.Data {
		entry, ok := entryFromOpenAICompatJSON(raw)
		if !ok {
			continue
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func intOrFirst(def int, vals ...*int) int {
	for _, p := range vals {
		if p != nil && *p > 0 {
			return *p
		}
	}
	return def
}

func entryFromOpenAICompatJSON(raw json.RawMessage) (Entry, bool) {
	var m listModelJSON
	if err := json.Unmarshal(raw, &m); err != nil {
		return Entry{}, false
	}
	id := strings.TrimSpace(m.ID)
	if id == "" {
		return Entry{}, false
	}
	if m.Status != nil && *m.Status <= 0 {
		return Entry{}, false
	}
	inPrice, outPrice := pricingFromListModel(m)
	label := strings.TrimSpace(m.DisplayName)
	if label == "" {
		label = strings.TrimSpace(m.Title)
	}
	if label == "" {
		label = strings.TrimSpace(m.Name)
	}
	if label == "" {
		label = id
	}
	features := append([]string(nil), m.Features...)
	owner := strings.TrimSpace(m.OwnedBy)
	if owner == "" {
		owner = ownerFromModelID(id)
	}
	return Entry{
		ID: id, DisplayName: label, Description: strings.TrimSpace(m.Description), OwnedBy: owner,
		InputPricePer1M: inPrice, OutputPricePer1M: outPrice,
		ContextWindow: intOrFirst(0, m.ContextLength, m.ContextSize),
		MaxOutput:     intOrFirst(0, m.MaxCompletionTokens, m.MaxOutputTokens),
		Features:      features,
		RawJSON:       append(json.RawMessage(nil), raw...),
	}, true
}

func pricingFromListModel(m listModelJSON) (inPrice, outPrice float64) {
	if m.InputTokenPricePerM != nil {
		inPrice = *m.InputTokenPricePerM
	}
	if m.OutputTokenPricePerM != nil {
		outPrice = *m.OutputTokenPricePerM
	}
	if inPrice > 0 || outPrice > 0 {
		return inPrice, outPrice
	}
	if m.Pricing != nil {
		inPrice = asFloat(m.Pricing.Prompt) * 1_000_000
		outPrice = asFloat(m.Pricing.Completion) * 1_000_000
	}
	return inPrice, outPrice
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

func FetchZAI(env map[string]string) ([]Entry, error) {
	return fetchOpenAICompatModels(
		envOr(env, "ZAI_BASE_URL", DefaultZAIBaseURL),
		env["ZAI_API_KEY"], "Bearer",
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

func FetchKimi(env map[string]string) ([]Entry, error) {
	key := strings.TrimSpace(env["MOONSHOT_API_KEY"])
	if key == "" {
		key = strings.TrimSpace(env["KIMI_API_KEY"])
	}
	return fetchOpenAICompatModels(
		envOr(env, "MOONSHOT_BASE_URL", envOr(env, "KIMI_BASE_URL", DefaultKimiBaseURL)),
		key, "Bearer",
	)
}

func FetchXiaomi(env map[string]string) ([]Entry, error) {
	key := strings.TrimSpace(env["XIAOMI_API_KEY"])
	if key == "" {
		key = strings.TrimSpace(env["MIMO_API_KEY"])
	}
	return fetchOpenAICompatModels(
		envOr(env, "XIAOMI_BASE_URL", envOr(env, "MIMO_BASE_URL", DefaultXiaomiBaseURL)),
		key, "Bearer",
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
		Data []json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	var entries []Entry
	for _, raw := range payload.Data {
		var m openRouterModel
		if err := json.Unmarshal(raw, &m); err != nil {
			continue
		}
		id := strings.TrimSpace(m.ID)
		if id == "" {
			continue
		}
		ctx := 128000
		if m.ContextLength != nil {
			ctx = *m.ContextLength
		} else if m.TopProvider != nil && m.TopProvider.ContextLength != nil {
			ctx = *m.TopProvider.ContextLength
		}
		maxOut := 16384
		if m.TopProvider != nil && m.TopProvider.MaxCompletionTokens != nil {
			maxOut = *m.TopProvider.MaxCompletionTokens
		}
		var inPrice, outPrice float64
		if m.Pricing != nil {
			inPrice = asFloat(m.Pricing.Prompt) * 1_000_000
			outPrice = asFloat(m.Pricing.Completion) * 1_000_000
		}
		entries = append(entries, Entry{
			ID: id, InputPricePer1M: inPrice, OutputPricePer1M: outPrice,
			ContextWindow: ctx, MaxOutput: maxOut, DisplayName: id,
			RawJSON: append(json.RawMessage(nil), raw...),
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
		Data []json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	var entries []Entry
	for _, raw := range payload.Data {
		var m struct {
			ID          string `json:"id"`
			DisplayName string `json:"display_name"`
		}
		if err := json.Unmarshal(raw, &m); err != nil {
			continue
		}
		id := strings.TrimSpace(m.ID)
		if id == "" {
			continue
		}
		label := strings.TrimSpace(m.DisplayName)
		if label == "" {
			label = id
		}
		entries = append(entries, Entry{
			ID: id, DisplayName: label,
			RawJSON: append(json.RawMessage(nil), raw...),
		})
	}
	return entries, nil
}

func FetchGemini(env map[string]string) ([]Entry, error) {
	apiKey := strings.TrimSpace(env["GEMINI_API_KEY"])
	if apiKey == "" {
		return nil, nil
	}
	base := strings.TrimRight(envOr(env, "GEMINI_BASE_URL", "https://generativelanguage.googleapis.com"), "/")
	url := base + "/v1beta/models?key=" + apiKey
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
		Models []json.RawMessage `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	var entries []Entry
	for _, raw := range payload.Models {
		var m struct {
			Name                       string   `json:"name"`
			DisplayName                string   `json:"displayName"`
			InputTokenLimit            int      `json:"inputTokenLimit"`
			OutputTokenLimit           int      `json:"outputTokenLimit"`
			SupportedGenerationMethods []string `json:"supportedGenerationMethods"`
		}
		if err := json.Unmarshal(raw, &m); err != nil {
			continue
		}
		name := strings.TrimSpace(m.Name)
		if name == "" {
			continue
		}
		supportsGen := false
		for _, method := range m.SupportedGenerationMethods {
			if method == "generateContent" {
				supportsGen = true
				break
			}
		}
		if !supportsGen {
			continue
		}
		id := strings.TrimPrefix(name, "models/")
		label := strings.TrimSpace(m.DisplayName)
		if label == "" {
			label = id
		}
		ctxWin := m.InputTokenLimit
		if ctxWin <= 0 {
			ctxWin = 128000
		}
		maxOut := m.OutputTokenLimit
		if maxOut <= 0 {
			maxOut = 8192
		}
		entries = append(entries, Entry{
			ID: id, DisplayName: label, ContextWindow: ctxWin, MaxOutput: maxOut,
			RawJSON: append(json.RawMessage(nil), raw...),
		})
	}
	return entries, nil
}

func FetchOllama(env map[string]string) ([]Entry, error) {
	baseURL := strings.TrimSpace(env["OLLAMA_BASE_URL"])
	if baseURL == "" {
		return nil, nil
	}
	root := strings.TrimSuffix(strings.TrimRight(baseURL, "/"), "/v1")
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
		Models []json.RawMessage `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	var entries []Entry
	for _, raw := range payload.Models {
		var m struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(raw, &m); err != nil {
			continue
		}
		id := strings.TrimSpace(m.Name)
		if id == "" {
			continue
		}
		entries = append(entries, Entry{
			ID: id, DisplayName: id,
			RawJSON: append(json.RawMessage(nil), raw...),
		})
	}
	return entries, nil
}
