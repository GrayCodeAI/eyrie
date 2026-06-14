package live

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/GrayCodeAI/eyrie/catalog/opencodego"
	"github.com/GrayCodeAI/eyrie/catalog/xiaomi"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

const (
	DefaultOpenRouterBaseURL = "https://openrouter.ai/api/v1"
	DefaultCanopyWaveBaseURL = "https://inference.canopywave.io/v1"
	DefaultZAIBaseURL        = "https://api.z.ai/api/paas/v4"
	DefaultOpenAIBaseURL     = "https://api.openai.com/v1"
	DefaultGrokBaseURL       = "https://api.x.ai/v1"
	DefaultOpenCodeGoBaseURL = opencodego.DefaultBaseURL
	DefaultKimiBaseURL       = "https://api.moonshot.ai/v1"
	DefaultXiaomiMimoBaseURL = "https://api.xiaomimimo.com/v1"
	DefaultMiniMaxBaseURL    = "https://api.minimax.io/v1"
)

// FetchFunc lists models from a live provider API.
type FetchFunc func(env map[string]string) ([]Entry, error)

const DefaultDeepSeekBaseURL = "https://api.deepseek.com/v1"

// Registry maps fetcher keys to implementations.
var Registry = map[string]FetchFunc{
	"anthropic":              FetchAnthropic,
	"openai":                 FetchOpenAI,
	"azure":                  FetchAzure,
	"gemini":                 FetchGemini,
	"bedrock":                FetchBedrock,
	"vertex":                 FetchVertex,
	"openrouter":             FetchOpenRouter,
	"grok":                   FetchGrok,
	"z-ai":                   FetchZAI,
	"canopywave":             FetchCanopyWave,
	"opencodego":             FetchOpenCodeGo,
	"kimi":                   FetchKimi,
	"xiaomi_mimo_payg":       FetchXiaomiMimoPayg,
	"xiaomi_mimo_token_plan": FetchXiaomiMimoTokenPlan,
	"minimax_token_plan":     FetchMiniMaxTokenPlan,
	"minimax_payg":           FetchMiniMaxPayg,
	"ollama":                 FetchOllama,
	"deepseek":               FetchDeepSeek,
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
	// APIType captures protocol hints from the provider (e.g., "anthropic", "openai").
	// Not all providers return this; empty means unknown.
	APIType string `json:"api_type,omitempty"`
	Pricing *struct {
		Prompt      interface{} `json:"prompt"`
		Completion  interface{} `json:"completion"`
		CachedRead  interface{} `json:"cached_read,omitempty"`
		CachedWrite interface{} `json:"cached_write,omitempty"`
		Request     interface{} `json:"request,omitempty"`
		Image       interface{} `json:"image,omitempty"`
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

func fetchOpenAICompatModels(ctx context.Context, baseURL, apiKey, authHeader string) ([]Entry, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, nil
	}
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("live: missing base URL")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/models", nil)
	if err != nil {
		return nil, fmt.Errorf("live: create request: %w", err)
	}
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
	cachedRead, cachedWrite := cachedPricingFromListModel(m)
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
	// Derive protocol from api_type field or features/tags hints.
	protocol := protocolFromMetadata(m.APIType, features, m.Tags)
	return Entry{
		ID: id, DisplayName: label, Description: strings.TrimSpace(m.Description), OwnedBy: owner,
		InputPricePer1M: inPrice, OutputPricePer1M: outPrice,
		CachedReadPricePer1M:  cachedRead,
		CachedWritePricePer1M: cachedWrite,
		ContextWindow:         intOrFirst(0, m.ContextLength, m.ContextSize),
		MaxOutput:             intOrFirst(0, m.MaxCompletionTokens, m.MaxOutputTokens),
		Features:              features,
		Protocol:              protocol,
		RawJSON:               append(json.RawMessage(nil), raw...),
	}, true
}

// protocolFromMetadata derives the API protocol from available metadata.
// Returns "anthropic", "openai", or "" if unknown.
func protocolFromMetadata(apiType string, features, tags []string) string {
	apiType = strings.ToLower(strings.TrimSpace(apiType))
	if apiType == "anthropic" || apiType == "openai" {
		return apiType
	}
	// Check features and tags for protocol hints.
	for _, f := range append(features, tags...) {
		f = strings.ToLower(strings.TrimSpace(f))
		if f == "anthropic" || f == "anthropic-compat" || f == "anthropic-messages" {
			return "anthropic"
		}
		if f == "openai" || f == "openai-compat" || f == "openai-chat-completions" {
			return "openai"
		}
	}
	return ""
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

// cachedPricingFromListModel extracts cached read/write pricing from the model JSON.
// Returns (cachedRead, cachedWrite) per 1M tokens.
func cachedPricingFromListModel(m listModelJSON) (cachedRead, cachedWrite float64) {
	if m.Pricing == nil {
		return 0, 0
	}
	cachedRead = asFloat(m.Pricing.CachedRead) * 1_000_000
	cachedWrite = asFloat(m.Pricing.CachedWrite) * 1_000_000
	return cachedRead, cachedWrite
}

func envOr(env map[string]string, key, def string) string {
	if v := strings.TrimSpace(env[key]); v != "" {
		return v
	}
	return def
}

func firstEnv(env map[string]string, keys ...string) string {
	for _, key := range keys {
		if v := strings.TrimSpace(env[key]); v != "" {
			return v
		}
	}
	return ""
}

func FetchOpenAI(env map[string]string) ([]Entry, error) {
	entries, err := fetchOpenAICompatModels(
		context.Background(),
		envOr(env, "OPENAI_BASE_URL", DefaultOpenAIBaseURL),
		env["OPENAI_API_KEY"], "Bearer",
	)
	if err != nil {
		return nil, err
	}
	// Enrich with capabilities from OpenRouter (context window, pricing, supported parameters)
	enrichOpenAIWithOpenRouter(entries)
	return entries, nil
}

// enrichOpenAIWithOpenRouter fetches OpenRouter's model list and enriches OpenAI entries
// with context window, pricing, and capability data that OpenAI's own API doesn't return.
func enrichOpenAIWithOpenRouter(entries []Entry) {
	if len(entries) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, DefaultOpenRouterBaseURL+"/models", nil)
	if err != nil {
		return
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "eyrie-model-catalog/1.0")
	resp, err := httpClient.Do(req)
	if err != nil {
		return
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return
	}
	var payload struct {
		Data []openRouterModelEntry `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return
	}
	// Build lookup map: "gpt-4o" → openRouterModelEntry
	lookup := map[string]openRouterModelEntry{}
	for _, m := range payload.Data {
		// OpenRouter IDs are "openai/gpt-4o" — strip prefix
		nativeID := strings.TrimPrefix(m.ID, "openai/")
		if nativeID != m.ID {
			lookup[nativeID] = m
		}
	}
	// Enrich entries
	for i := range entries {
		or, ok := lookup[entries[i].ID]
		if !ok {
			continue
		}
		if or.ContextLength > 0 {
			entries[i].ContextWindow = or.ContextLength
			entries[i].MaxInputTokens = or.ContextLength
		}
		if or.TopProvider.MaxCompletionTokens > 0 {
			entries[i].MaxOutput = or.TopProvider.MaxCompletionTokens
		}
		if p, err := strconv.ParseFloat(or.Pricing.Prompt, 64); err == nil && p > 0 {
			entries[i].InputPricePer1M = p * 1_000_000
		}
		if p, err := strconv.ParseFloat(or.Pricing.Completion, 64); err == nil && p > 0 {
			entries[i].OutputPricePer1M = p * 1_000_000
		}
		// Map supported parameters to features
		features := map[string]bool{}
		for _, sp := range or.SupportedParameters {
			features[sp] = true
		}
		if features["tools"] || features["functions"] {
			entries[i].Features = append(entries[i].Features, "tools")
		}
		if features["reasoning_effort"] {
			entries[i].Features = append(entries[i].Features, "thinking:enabled")
			entries[i].ThinkingEnabled = true
		}
		if features["response_format"] {
			entries[i].Features = append(entries[i].Features, "structured_output")
			entries[i].StructuredOutput = true
		}
		if features["temperature"] {
			entries[i].Features = appendUnique(entries[i].Features, "temperature")
		}
		if features["presence_penalty"] || features["frequency_penalty"] {
			entries[i].Features = appendUnique(entries[i].Features, "penalties")
		}
		if or.ContextLength > 0 {
			entries[i].Features = appendUnique(entries[i].Features, fmt.Sprintf("context:%d", or.ContextLength))
		}
	}
}

// enrichFromOpenRouter fetches OpenRouter's model list and enriches entries
// with pricing and context data. prefix is the OpenRouter provider prefix (e.g., "moonshotai/").
func enrichFromOpenRouter(entries []Entry, prefix string) {
	if len(entries) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, DefaultOpenRouterBaseURL+"/models", nil)
	if err != nil {
		return
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "eyrie-model-catalog/1.0")
	resp, err := httpClient.Do(req)
	if err != nil {
		return
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return
	}
	var payload struct {
		Data []openRouterModelEntry `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return
	}
	// Build lookup map by stripping prefix
	lookup := map[string]openRouterModelEntry{}
	for _, m := range payload.Data {
		nativeID := strings.TrimPrefix(m.ID, prefix)
		if nativeID != m.ID {
			lookup[nativeID] = m
		}
	}
	// Enrich entries
	for i := range entries {
		or, ok := lookup[entries[i].ID]
		if !ok {
			continue
		}
		if or.ContextLength > 0 && entries[i].ContextWindow == 0 {
			entries[i].ContextWindow = or.ContextLength
		}
		if or.TopProvider.MaxCompletionTokens > 0 && entries[i].MaxOutput == 0 {
			entries[i].MaxOutput = or.TopProvider.MaxCompletionTokens
		}
		if p, err := strconv.ParseFloat(or.Pricing.Prompt, 64); err == nil && p > 0 && entries[i].InputPricePer1M == 0 {
			entries[i].InputPricePer1M = p * 1_000_000
		}
		if p, err := strconv.ParseFloat(or.Pricing.Completion, 64); err == nil && p > 0 && entries[i].OutputPricePer1M == 0 {
			entries[i].OutputPricePer1M = p * 1_000_000
		}
	}
}

type openRouterModelEntry struct {
	ID                  string   `json:"id"`
	ContextLength       int      `json:"context_length"`
	SupportedParameters []string `json:"supported_parameters"`
	TopProvider         struct {
		MaxCompletionTokens int `json:"max_completion_tokens"`
	} `json:"top_provider"`
	Pricing struct {
		Prompt     string `json:"prompt"`
		Completion string `json:"completion"`
	} `json:"pricing"`
}

func appendUnique(slice []string, s string) []string {
	for _, v := range slice {
		if v == s {
			return slice
		}
	}
	return append(slice, s)
}

func FetchMiniMaxTokenPlan(env map[string]string) ([]Entry, error) {
	return fetchOpenAICompatModels(
		context.Background(),
		envOr(env, "MINIMAX_TOKEN_PLAN_BASE_URL", DefaultMiniMaxBaseURL),
		env["MINIMAX_TOKEN_PLAN_API_KEY"], "Bearer",
	)
}

func FetchMiniMaxPayg(env map[string]string) ([]Entry, error) {
	return fetchOpenAICompatModels(
		context.Background(),
		envOr(env, "MINIMAX_PAYG_BASE_URL", DefaultMiniMaxBaseURL),
		env["MINIMAX_PAYG_API_KEY"], "Bearer",
	)
}

func FetchAzure(env map[string]string) ([]Entry, error) {
	if id := firstEnv(env, "AZURE_OPENAI_DEPLOYMENT", "AZURE_OPENAI_MODEL", "OPENAI_MODEL"); id != "" {
		return []Entry{{ID: id, DisplayName: id}}, nil
	}
	token := firstEnv(env, "AZURE_OPENAI_MANAGEMENT_TOKEN", "AZURE_ACCESS_TOKEN")
	subscriptionID := strings.TrimSpace(env["AZURE_SUBSCRIPTION_ID"])
	resourceGroup := strings.TrimSpace(env["AZURE_RESOURCE_GROUP"])
	accountName := firstEnv(env, "AZURE_OPENAI_ACCOUNT_NAME", "AZURE_OPENAI_ACCOUNT")
	if token == "" || subscriptionID == "" || resourceGroup == "" || accountName == "" {
		return nil, nil
	}
	apiVersion := envOr(env, "AZURE_OPENAI_MANAGEMENT_API_VERSION", "2024-10-01")
	path := fmt.Sprintf("https://management.azure.com/subscriptions/%s/resourceGroups/%s/providers/Microsoft.CognitiveServices/accounts/%s/deployments",
		url.PathEscape(subscriptionID), url.PathEscape(resourceGroup), url.PathEscape(accountName))
	reqURL := path + "?api-version=" + url.QueryEscape(apiVersion)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("live: create azure request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "eyrie-model-catalog/1.0")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("azure deployment fetch failed (%d)", resp.StatusCode)
	}
	var payload struct {
		Value []json.RawMessage `json:"value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	var entries []Entry
	for _, raw := range payload.Value {
		entry, ok := entryFromAzureDeploymentJSON(raw)
		if ok {
			entries = append(entries, entry)
		}
	}
	return entries, nil
}

func entryFromAzureDeploymentJSON(raw json.RawMessage) (Entry, bool) {
	var dep struct {
		Name       string `json:"name"`
		Properties struct {
			Model struct {
				Name    string `json:"name"`
				Format  string `json:"format"`
				Version string `json:"version"`
			} `json:"model"`
			ProvisioningState string `json:"provisioningState"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(raw, &dep); err != nil {
		return Entry{}, false
	}
	id := strings.TrimSpace(dep.Name)
	if id == "" || !strings.EqualFold(strings.TrimSpace(dep.Properties.ProvisioningState), "Succeeded") {
		return Entry{}, false
	}
	label := id
	if model := strings.TrimSpace(dep.Properties.Model.Name); model != "" {
		label = id + " (" + model + ")"
	}
	return Entry{ID: id, DisplayName: label, OwnedBy: "azure", RawJSON: append(json.RawMessage(nil), raw...)}, true
}

func FetchBedrock(env map[string]string) ([]Entry, error) {
	accessKeyID := strings.TrimSpace(env["AWS_ACCESS_KEY_ID"])
	secretAccessKey := strings.TrimSpace(env["AWS_SECRET_ACCESS_KEY"])
	region := firstEnv(env, "AWS_REGION", "AWS_DEFAULT_REGION")
	if accessKeyID == "" || secretAccessKey == "" || region == "" {
		return nil, nil
	}
	reqURL := fmt.Sprintf("https://bedrock.%s.amazonaws.com/foundation-models?byProvider=Anthropic", region)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("live: create bedrock request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "eyrie-model-catalog/1.0")
	signAWSV4(req, accessKeyID, secretAccessKey, strings.TrimSpace(env["AWS_SESSION_TOKEN"]), region, "bedrock", nil)
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bedrock model fetch failed (%d)", resp.StatusCode)
	}
	var payload struct {
		ModelSummaries []json.RawMessage `json:"modelSummaries"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	var entries []Entry
	for _, raw := range payload.ModelSummaries {
		entry, ok := entryFromBedrockModelJSON(raw)
		if ok {
			entries = append(entries, entry)
		}
	}
	return entries, nil
}

func entryFromBedrockModelJSON(raw json.RawMessage) (Entry, bool) {
	var m struct {
		ModelID                    string   `json:"modelId"`
		ModelName                  string   `json:"modelName"`
		ProviderName               string   `json:"providerName"`
		ResponseStreamingSupported bool     `json:"responseStreamingSupported"`
		InputModalities            []string `json:"inputModalities"`
		OutputModalities           []string `json:"outputModalities"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		return Entry{}, false
	}
	id := strings.TrimSpace(m.ModelID)
	if id == "" {
		return Entry{}, false
	}
	if provider := strings.TrimSpace(m.ProviderName); provider != "" && !strings.EqualFold(provider, "Anthropic") {
		return Entry{}, false
	}
	label := strings.TrimSpace(m.ModelName)
	if label == "" {
		label = id
	}
	features := append([]string(nil), m.InputModalities...)
	features = append(features, m.OutputModalities...)
	if m.ResponseStreamingSupported {
		features = append(features, "streaming")
	}
	return Entry{ID: id, DisplayName: label, OwnedBy: "anthropic", Features: features, RawJSON: append(json.RawMessage(nil), raw...)}, true
}

func FetchVertex(env map[string]string) ([]Entry, error) {
	projectID := strings.TrimSpace(env["VERTEX_PROJECT_ID"])
	region := strings.TrimSpace(env["VERTEX_REGION"])
	token := firstEnv(env, "VERTEX_ACCESS_TOKEN", "GOOGLE_OAUTH_ACCESS_TOKEN")
	if projectID == "" || region == "" || token == "" {
		return nil, nil
	}
	// Fetch Anthropic models from Vertex AI (not Google's own models)
	reqURL := fmt.Sprintf("https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/anthropic/models",
		region, url.PathEscape(projectID), url.PathEscape(region))
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("live: create vertex request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "eyrie-model-catalog/1.0")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("vertex model fetch failed (%d)", resp.StatusCode)
	}
	var payload struct {
		PublisherModels []json.RawMessage `json:"publisherModels"`
		Models          []json.RawMessage `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	rawModels := payload.PublisherModels
	if len(rawModels) == 0 {
		rawModels = payload.Models
	}
	var entries []Entry
	for _, raw := range rawModels {
		entry, ok := entryFromVertexModelJSON(raw)
		if ok {
			entries = append(entries, entry)
		}
	}
	return entries, nil
}

func entryFromVertexModelJSON(raw json.RawMessage) (Entry, bool) {
	var m struct {
		Name             string   `json:"name"`
		DisplayName      string   `json:"displayName"`
		Description      string   `json:"description"`
		VersionID        string   `json:"versionId"`
		Frameworks       []string `json:"frameworks"`
		SupportedActions []string `json:"supportedActions"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		return Entry{}, false
	}
	id := strings.TrimSpace(m.Name)
	if id == "" {
		return Entry{}, false
	}
	if i := strings.LastIndex(id, "/models/"); i >= 0 {
		id = id[i+len("/models/"):]
	}
	label := strings.TrimSpace(m.DisplayName)
	if label == "" {
		label = id
	}
	features := append([]string(nil), m.Frameworks...)
	// Tag supported actions as features
	for _, action := range m.SupportedActions {
		features = appendUnique(features, "action:"+action)
	}
	return Entry{
		ID:          id,
		DisplayName: label,
		Description: strings.TrimSpace(m.Description),
		OwnedBy:     "anthropic",
		Features:    features,
		RawJSON:     append(json.RawMessage(nil), raw...),
	}, true
}

func FetchGrok(env map[string]string) ([]Entry, error) {
	entries, err := fetchOpenAICompatModels(
		context.Background(),
		envOr(env, "XAI_BASE_URL", DefaultGrokBaseURL),
		env["XAI_API_KEY"], "Bearer",
	)
	if err != nil {
		return nil, err
	}
	enrichFromOpenRouter(entries, "x-ai/")
	return entries, nil
}

func FetchZAI(env map[string]string) ([]Entry, error) {
	entries, err := fetchOpenAICompatModels(
		context.Background(),
		envOr(env, "ZAI_BASE_URL", DefaultZAIBaseURL),
		env["ZAI_API_KEY"], "Bearer",
	)
	if err != nil {
		return nil, err
	}
	enrichFromOpenRouter(entries, "z-ai/")
	return entries, nil
}

func FetchCanopyWave(env map[string]string) ([]Entry, error) {
	entries, err := fetchOpenAICompatModels(
		context.Background(),
		envOr(env, "CANOPYWAVE_BASE_URL", DefaultCanopyWaveBaseURL),
		env["CANOPYWAVE_API_KEY"], "Bearer",
	)
	if err != nil {
		return nil, err
	}
	// CanopyWave returns pricing in cents per 1M tokens, not dollars.
	// Convert to dollars: 140 cents = $1.40.
	for i := range entries {
		if entries[i].InputPricePer1M > 0 {
			entries[i].InputPricePer1M = entries[i].InputPricePer1M / 100
		}
		if entries[i].OutputPricePer1M > 0 {
			entries[i].OutputPricePer1M = entries[i].OutputPricePer1M / 100
		}
	}
	return entries, nil
}

func FetchOpenCodeGo(env map[string]string) ([]Entry, error) {
	entries, err := fetchOpenAICompatModels(
		context.Background(),
		envOr(env, "OPENCODEGO_BASE_URL", DefaultOpenCodeGoBaseURL),
		env["OPENCODEGO_API_KEY"], "Bearer",
	)
	if err != nil {
		return nil, err
	}
	for i := range entries {
		entries[i].ID = opencodego.NativeModelID(entries[i].ID)
		// Merge with static metadata from docs (pricing, protocol, context windows).
		if meta, ok := opencodego.MetadataForModel(entries[i].ID); ok {
			entries[i] = enrichFromStaticMeta(entries[i], meta)
		} else {
			// Unknown model — derive protocol from name pattern.
			if entries[i].Protocol == "" {
				entries[i].Protocol = opencodego.ProtocolForModel(entries[i].ID)
			}
		}
	}
	return entries, nil
}

// enrichFromStaticMeta fills Entry fields from the static docs-based metadata.
// API-provided fields (like id, owned_by) are preserved; static metadata fills gaps.
func enrichFromStaticMeta(e Entry, meta opencodego.ModelMetadata) Entry {
	e.Protocol = meta.Protocol
	if e.InputPricePer1M == 0 {
		e.InputPricePer1M = meta.InputPer1M
	}
	if e.OutputPricePer1M == 0 {
		e.OutputPricePer1M = meta.OutputPer1M
	}
	if e.CachedReadPricePer1M == 0 {
		e.CachedReadPricePer1M = meta.CachedRead
	}
	if e.CachedWritePricePer1M == 0 {
		e.CachedWritePricePer1M = meta.CachedWrite
	}
	if e.ContextWindow == 0 {
		e.ContextWindow = meta.Context
	}
	if e.MaxOutput == 0 {
		e.MaxOutput = meta.MaxOutput
	}
	return e
}

func FetchKimi(env map[string]string) ([]Entry, error) {
	entries, err := fetchOpenAICompatModels(
		context.Background(),
		envOr(env, "MOONSHOT_BASE_URL", DefaultKimiBaseURL),
		env["MOONSHOT_API_KEY"], "Bearer",
	)
	if err != nil {
		return nil, err
	}
	// Enrich with pricing from OpenRouter (Kimi API doesn't return pricing).
	enrichFromOpenRouter(entries, "moonshotai/")
	return entries, nil
}

func FetchXiaomiMimoPayg(env map[string]string) ([]Entry, error) {
	return fetchMimoOpenAIModels(env, "XIAOMI_MIMO_PAYG_API_KEY", "XIAOMI_MIMO_PAYG_BASE_URL", DefaultXiaomiMimoBaseURL)
}

func FetchXiaomiMimoTokenPlan(env map[string]string) ([]Entry, error) {
	base := resolveTokenPlanOpenAIBase(env)
	if base != "" {
		env2 := make(map[string]string, len(env)+1)
		for k, v := range env {
			env2[k] = v
		}
		env2["XIAOMI_MIMO_TOKEN_PLAN_BASE_URL"] = base
		env = env2
	}
	return fetchMimoOpenAIModels(env, "XIAOMI_MIMO_TOKEN_PLAN_API_KEY", "XIAOMI_MIMO_TOKEN_PLAN_BASE_URL", "")
}

func resolveTokenPlanOpenAIBase(env map[string]string) string {
	region, err := xiaomi.NormalizeRegion(env["XIAOMI_MIMO_TOKEN_PLAN_REGION"])
	if err != nil {
		region = ""
	}
	override := strings.TrimSpace(env["XIAOMI_MIMO_TOKEN_PLAN_BASE_URL"])
	base, err := xiaomi.ResolveOpenAIBasePreferRegion(xiaomi.BillingTokenPlan, region, override)
	if err != nil {
		return ""
	}
	return base
}

func fetchMimoOpenAIModels(env map[string]string, keyEnv, baseEnv, defaultBase string) ([]Entry, error) {
	apiKey := strings.TrimSpace(env[keyEnv])
	if apiKey == "" {
		return nil, nil
	}
	base := strings.TrimSpace(env[baseEnv])
	if base == "" {
		base = defaultBase
	}
	if base == "" {
		return nil, fmt.Errorf("live: missing MiMo base URL (set %s)", baseEnv)
	}
	return fetchMimoModels(context.Background(), base, apiKey, env)
}

func fetchMimoModels(ctx context.Context, baseURL, apiKey string, env map[string]string) ([]Entry, error) {
	raw, err := xiaomi.FetchOpenAIModelsJSON(ctx, baseURL, apiKey)
	if err != nil {
		return nil, err
	}
	platform, _ := xiaomi.FetchPlatformModelsIndex(ctx, xiaomi.PlatformModelsURLFromEnv(env))
	var entries []Entry
	for _, r := range raw {
		entry, ok := entryFromOpenAICompatJSON(r)
		if ok {
			entries = append(entries, enrichMimoEntry(entry, platform))
		}
	}
	return entries, nil
}

func FetchOpenRouter(env map[string]string) ([]Entry, error) {
	apiKey := strings.TrimSpace(env["OPENROUTER_API_KEY"])
	if apiKey == "" {
		return nil, nil
	}
	baseURL := strings.TrimRight(envOr(env, "OPENROUTER_BASE_URL", DefaultOpenRouterBaseURL), "/")
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, baseURL+"/models", nil)
	if err != nil {
		return nil, fmt.Errorf("live: create request: %w", err)
	}
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
		ctx := 0
		if m.ContextLength != nil {
			ctx = *m.ContextLength
		} else if m.TopProvider != nil && m.TopProvider.ContextLength != nil {
			ctx = *m.TopProvider.ContextLength
		}
		maxOut := 0
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

// anthropicModelEntry represents one model from the Anthropic GET /v1/models response.
type anthropicModelEntry struct {
	ID             string `json:"id"`
	DisplayName    string `json:"display_name"`
	MaxInputTokens int    `json:"max_input_tokens"`
	MaxTokens      int    `json:"max_tokens"`
	Capabilities   struct {
		Batch struct {
			Supported bool `json:"supported"`
		} `json:"batch"`
		Citations struct {
			Supported bool `json:"supported"`
		} `json:"citations"`
		CodeExecution struct {
			Supported bool `json:"supported"`
		} `json:"code_execution"`
		Effort struct {
			Supported bool `json:"supported"`
			Low       struct {
				Supported bool `json:"supported"`
			} `json:"low"`
			Medium struct {
				Supported bool `json:"supported"`
			} `json:"medium"`
			High struct {
				Supported bool `json:"supported"`
			} `json:"high"`
			XHigh struct {
				Supported bool `json:"supported"`
			} `json:"xhigh"`
			Max struct {
				Supported bool `json:"supported"`
			} `json:"max"`
		} `json:"effort"`
		ImageInput struct {
			Supported bool `json:"supported"`
		} `json:"image_input"`
		PDFInput struct {
			Supported bool `json:"supported"`
		} `json:"pdf_input"`
		StructuredOutputs struct {
			Supported bool `json:"supported"`
		} `json:"structured_outputs"`
		Thinking struct {
			Supported bool `json:"supported"`
			Types     struct {
				Enabled struct {
					Supported bool `json:"supported"`
				} `json:"enabled"`
				Adaptive struct {
					Supported bool `json:"supported"`
				} `json:"adaptive"`
			} `json:"types"`
		} `json:"thinking"`
	} `json:"capabilities"`
}

func FetchAnthropic(env map[string]string) ([]Entry, error) {
	apiKey := strings.TrimSpace(env["ANTHROPIC_API_KEY"])
	if apiKey == "" {
		return nil, nil
	}
	baseURL := strings.TrimRight(envOr(env, "ANTHROPIC_BASE_URL", "https://api.anthropic.com/v1"), "/")
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, baseURL+"/models", nil)
	if err != nil {
		return nil, fmt.Errorf("live: create request: %w", err)
	}
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
		var m anthropicModelEntry
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
		entry := Entry{
			ID: id, DisplayName: label,
			ContextWindow:  m.MaxInputTokens, // maps to ModelCatalogEntry.ContextWindow via LiveEntriesToCatalog
			MaxInputTokens: m.MaxInputTokens,
			MaxOutput:      m.MaxTokens,
			RawJSON:        append(json.RawMessage(nil), raw...),
		}
		// Extract capabilities
		entry.ThinkingEnabled = m.Capabilities.Thinking.Types.Enabled.Supported
		entry.ThinkingAdaptive = m.Capabilities.Thinking.Types.Adaptive.Supported
		if m.Capabilities.Effort.Supported {
			entry.EffortSupported = true
			var levels []string
			for _, lvl := range []string{"low", "medium", "high", "xhigh", "max"} {
				switch lvl {
				case "low":
					if m.Capabilities.Effort.Low.Supported {
						levels = append(levels, lvl)
					}
				case "medium":
					if m.Capabilities.Effort.Medium.Supported {
						levels = append(levels, lvl)
					}
				case "high":
					if m.Capabilities.Effort.High.Supported {
						levels = append(levels, lvl)
					}
				case "xhigh":
					if m.Capabilities.Effort.XHigh.Supported {
						levels = append(levels, lvl)
					}
				case "max":
					if m.Capabilities.Effort.Max.Supported {
						levels = append(levels, lvl)
					}
				}
			}
			entry.EffortLevels = strings.Join(levels, ",")
		}
		entry.StructuredOutput = m.Capabilities.StructuredOutputs.Supported
		entry.CodeExecution = m.Capabilities.CodeExecution.Supported
		entry.CitationsSupported = m.Capabilities.Citations.Supported
		entry.PDFInput = m.Capabilities.PDFInput.Supported
		entry.ImageInput = m.Capabilities.ImageInput.Supported
		// Populate Features list for downstream catalog pipeline
		if entry.ThinkingEnabled {
			entry.Features = append(entry.Features, "thinking:enabled")
		}
		if entry.ThinkingAdaptive {
			entry.Features = append(entry.Features, "thinking:adaptive")
		}
		if entry.EffortSupported {
			entry.Features = append(entry.Features, "effort")
			if entry.EffortLevels != "" {
				entry.Features = append(entry.Features, "effort:"+entry.EffortLevels)
			}
		}
		if entry.StructuredOutput {
			entry.Features = append(entry.Features, "structured_output")
		}
		if entry.CodeExecution {
			entry.Features = append(entry.Features, "code_execution")
		}
		if entry.CitationsSupported {
			entry.Features = append(entry.Features, "citations")
		}
		if entry.PDFInput {
			entry.Features = append(entry.Features, "pdf_input")
		}
		if entry.ImageInput {
			entry.Features = append(entry.Features, "image_input")
		}
		entries = append(entries, entry)
	}
	// Enrich with pricing from OpenRouter (Anthropic API doesn't return pricing).
	enrichFromOpenRouter(entries, "anthropic/")
	return entries, nil
}

func FetchGemini(env map[string]string) ([]Entry, error) {
	apiKey := strings.TrimSpace(env["GEMINI_API_KEY"])
	if apiKey == "" {
		return nil, nil
	}
	base := strings.TrimRight(envOr(env, "GEMINI_BASE_URL", "https://generativelanguage.googleapis.com"), "/")
	url := base + "/v1beta/models?key=" + apiKey
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("live: create request: %w", err)
	}
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
		entries = append(entries, Entry{
			ID: id, DisplayName: label, ContextWindow: m.InputTokenLimit, MaxOutput: m.OutputTokenLimit,
			RawJSON: append(json.RawMessage(nil), raw...),
		})
	}
	// Enrich with pricing from OpenRouter (Gemini API doesn't return pricing).
	enrichFromOpenRouter(entries, "google/")
	return entries, nil
}

func FetchOllama(env map[string]string) ([]Entry, error) {
	baseURL := strings.TrimSpace(env["OLLAMA_BASE_URL"])
	if baseURL == "" {
		return nil, nil
	}
	root := strings.TrimSuffix(strings.TrimRight(baseURL, "/"), "/v1")
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, root+"/api/tags", nil)
	if err != nil {
		return nil, fmt.Errorf("live: create request: %w", err)
	}
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

// FetchDeepSeek lists models from the DeepSeek OpenAI-compatible API.
func FetchDeepSeek(env map[string]string) ([]Entry, error) {
	entries, err := fetchOpenAICompatModels(
		context.Background(),
		envOr(env, "DEEPSEEK_BASE_URL", DefaultDeepSeekBaseURL),
		env["DEEPSEEK_API_KEY"], "Bearer",
	)
	if err != nil {
		return nil, err
	}
	enrichFromOpenRouter(entries, "deepseek/")
	return entries, nil
}

func signAWSV4(req *http.Request, accessKeyID, secretAccessKey, sessionToken, region, service string, body []byte) {
	now := time.Now().UTC()
	dateStamp := now.Format("20060102")
	amzDate := now.Format("20060102T150405Z")
	payloadHash := sha256Hex(body)
	req.Header.Set("X-Amz-Date", amzDate)
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)
	if sessionToken != "" {
		req.Header.Set("X-Amz-Security-Token", sessionToken)
	}
	req.Header.Set("Host", req.URL.Host)
	canonicalHeaders, signedHeaders := canonicalAWSHeaders(req.Header)
	canonicalQuery := req.URL.Query().Encode()
	canonicalRequest := strings.Join([]string{
		req.Method,
		req.URL.EscapedPath(),
		canonicalQuery,
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")
	scope := strings.Join([]string{dateStamp, region, service, "aws4_request"}, "/")
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		scope,
		sha256Hex([]byte(canonicalRequest)),
	}, "\n")
	signature := hex.EncodeToString(hmacSHA256(awsSigningKey(secretAccessKey, dateStamp, region, service), []byte(stringToSign)))
	req.Header.Set("Authorization", fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		accessKeyID, scope, signedHeaders, signature))
}

func canonicalAWSHeaders(headers http.Header) (string, string) {
	var names []string
	for name := range headers {
		names = append(names, strings.ToLower(name))
	}
	sort.Strings(names)
	var b strings.Builder
	for _, name := range names {
		values := headers.Values(name)
		for i, v := range values {
			values[i] = strings.Join(strings.Fields(v), " ")
		}
		b.WriteString(name)
		b.WriteByte(':')
		b.WriteString(strings.Join(values, ","))
		b.WriteByte('\n')
	}
	return b.String(), strings.Join(names, ";")
}

func awsSigningKey(secret, dateStamp, region, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secret), []byte(dateStamp))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte(service))
	return hmacSHA256(kService, []byte("aws4_request"))
}

func hmacSHA256(key, data []byte) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(data)
	return mac.Sum(nil)
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
