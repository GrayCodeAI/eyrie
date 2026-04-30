package catalog

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	DefaultOpenRouterBaseURL  = "https://openrouter.ai/api/v1"
	DefaultCanopyWaveBaseURL  = "https://inference.canopywave.io/v1"
)

var catalogHTTPClient = &http.Client{Timeout: 30 * time.Second}

type openRouterModel struct {
	ID            string `json:"id"`
	ContextLength *int   `json:"context_length"`
	TopProvider   *struct {
		ContextLength      *int `json:"context_length"`
		MaxCompletionTokens *int `json:"max_completion_tokens"`
	} `json:"top_provider"`
	Pricing *struct {
		Prompt     interface{} `json:"prompt"`
		Completion interface{} `json:"completion"`
	} `json:"pricing"`
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

func asFloat(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case string:
		var f float64
		fmt.Sscanf(n, "%f", &f)
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

func fetchOpenRouterCatalog(env map[string]string) ([]ModelCatalogEntry, error) {
	apiKey := strings.TrimSpace(env["OPENROUTER_API_KEY"])
	if apiKey == "" {
		return nil, nil
	}
	baseURL := strings.TrimSpace(env["OPENROUTER_BASE_URL"])
	if baseURL == "" {
		baseURL = DefaultOpenRouterBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")

	req, _ := http.NewRequest("GET", baseURL+"/models", nil)
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "eyrie-model-catalog/1.0")

	resp, err := catalogHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("openrouter model fetch failed (%d)", resp.StatusCode)
	}

	var payload struct {
		Data []openRouterModel `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	var entries []ModelCatalogEntry
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
		entries = append(entries, ModelCatalogEntry{
			ID: id, InputPricePer1M: inPrice, OutputPricePer1M: outPrice,
			ContextWindow: ctx, MaxOutput: maxOut, DisplayName: id,
		})
	}
	return entries, nil
}

func fetchCanopyWaveCatalog(env map[string]string) ([]ModelCatalogEntry, error) {
	apiKey := strings.TrimSpace(env["CANOPYWAVE_API_KEY"])
	if apiKey == "" {
		return nil, nil
	}
	baseURL := strings.TrimSpace(env["CANOPYWAVE_BASE_URL"])
	if baseURL == "" {
		baseURL = DefaultCanopyWaveBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")

	req, _ := http.NewRequest("GET", baseURL+"/models", nil)
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "eyrie-model-catalog/1.0")

	resp, err := catalogHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("canopywave model fetch failed (%d)", resp.StatusCode)
	}

	var payload struct {
		Data []openAICompatModel `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	var entries []ModelCatalogEntry
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
		entries = append(entries, ModelCatalogEntry{
			ID: id, InputPricePer1M: inPrice, OutputPricePer1M: outPrice,
			ContextWindow: intOr(raw.ContextLength, 128000),
			MaxOutput:     intOr(raw.MaxCompletionTokens, 16384),
			DisplayName:   id,
		})
	}
	return entries, nil
}
