package client

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
)

// ProviderFeatures tracks which capabilities each provider supports.
// Prevents sending unsupported features (thinking, tools, images) to providers
// that don't handle them, avoiding cryptic API errors.
type ProviderFeatures struct {
	mu       sync.RWMutex
	features map[string]FeatureSet
}

// FeatureSet describes what a provider supports.
type FeatureSet struct {
	Thinking   bool `json:"thinking"`
	ToolUse    bool `json:"tool_use"`
	Images     bool `json:"images"`
	Streaming  bool `json:"streaming"`
	Caching    bool `json:"caching"`
	JSON       bool `json:"json_mode"`
	Embeddings bool `json:"embeddings"`
	MaxContext int  `json:"max_context"`
}

// NewProviderFeatures creates a feature registry with known provider capabilities.
func NewProviderFeatures() *ProviderFeatures {
	pf := &ProviderFeatures{
		features: map[string]FeatureSet{
			"anthropic": {
				Thinking: true, ToolUse: true, Images: true,
				Streaming: true, Caching: true, JSON: true,
				Embeddings: false, MaxContext: 200000,
			},
			"openai": {
				Thinking: true, ToolUse: true, Images: true,
				Streaming: true, Caching: false, JSON: true,
				Embeddings: true, MaxContext: 128000,
			},
			"gemini": {
				Thinking: true, ToolUse: true, Images: true,
				Streaming: true, Caching: false, JSON: true,
				Embeddings: true, MaxContext: 1000000,
			},
			"ollama": {
				Thinking: false, ToolUse: true, Images: true,
				Streaming: true, Caching: false, JSON: true,
				Embeddings: true, MaxContext: 32000,
			},
			"openrouter": {
				Thinking: true, ToolUse: true, Images: true,
				Streaming: true, Caching: false, JSON: true,
				Embeddings: false, MaxContext: 200000,
			},
			"grok": {
				Thinking: false, ToolUse: true, Images: false,
				Streaming: true, Caching: false, JSON: true,
				Embeddings: false, MaxContext: 131072,
			},
		},
	}
	return pf
}

// Get returns features for a provider.
func (pf *ProviderFeatures) Get(provider string) FeatureSet {
	pf.mu.RLock()
	defer pf.mu.RUnlock()
	if f, ok := pf.features[strings.ToLower(provider)]; ok {
		return f
	}
	// Unknown provider: assume basic features
	return FeatureSet{ToolUse: true, Streaming: true, MaxContext: 32000}
}

// Supports checks if a provider supports a specific feature.
func (pf *ProviderFeatures) Supports(provider, feature string) bool {
	f := pf.Get(provider)
	switch strings.ToLower(feature) {
	case "thinking":
		return f.Thinking
	case "tools", "tool_use":
		return f.ToolUse
	case "images":
		return f.Images
	case "streaming":
		return f.Streaming
	case "caching", "cache":
		return f.Caching
	case "json", "json_mode":
		return f.JSON
	case "embeddings":
		return f.Embeddings
	default:
		return false
	}
}

// DeprecationChecker warns when using models approaching retirement.
type DeprecationChecker struct {
	mu           sync.RWMutex
	deprecations map[string]DeprecationInfo
}

// DeprecationInfo describes a model's deprecation status.
type DeprecationInfo struct {
	Model       string    `json:"model"`
	Deprecated  bool      `json:"deprecated"`
	RetireDate  time.Time `json:"retire_date"`
	Replacement string    `json:"replacement"`
	Message     string    `json:"message"`
}

// NewDeprecationChecker creates a checker with known deprecations.
func NewDeprecationChecker() *DeprecationChecker {
	dc := &DeprecationChecker{
		deprecations: map[string]DeprecationInfo{
			"claude-3-opus-20240229": {
				Model: "claude-3-opus-20240229", Deprecated: true,
				Replacement: "claude-opus-4-6", Message: "Claude 3 Opus is deprecated. Use Claude Opus 4.6.",
			},
			"claude-3-sonnet-20240229": {
				Model: "claude-3-sonnet-20240229", Deprecated: true,
				Replacement: "claude-sonnet-4-6", Message: "Claude 3 Sonnet is deprecated. Use Claude Sonnet 4.6.",
			},
			"claude-3-haiku-20240307": {
				Model: "claude-3-haiku-20240307", Deprecated: true,
				Replacement: "claude-haiku-4-5-20251001", Message: "Claude 3 Haiku is deprecated. Use Claude Haiku 4.5.",
			},
			"gpt-4-turbo": {
				Model: "gpt-4-turbo", Deprecated: true,
				Replacement: "gpt-4.1", Message: "GPT-4 Turbo is deprecated. Use GPT-4.1.",
			},
			"gpt-3.5-turbo": {
				Model: "gpt-3.5-turbo", Deprecated: true,
				Replacement: "gpt-4.1-mini", Message: "GPT-3.5 Turbo is deprecated. Use GPT-4.1 Mini.",
			},
		},
	}
	return dc
}

// Check returns deprecation info if the model is deprecated.
func (dc *DeprecationChecker) Check(model string) *DeprecationInfo {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	if info, ok := dc.deprecations[model]; ok && info.Deprecated {
		return &info
	}
	return nil
}

// Warn logs a deprecation warning if applicable.
func (dc *DeprecationChecker) Warn(model string) {
	if info := dc.Check(model); info != nil {
		slog.Warn("deprecated model", "model", model, "replacement", info.Replacement, "message", info.Message)
	}
}

// RequestLogger logs all API requests/responses for debugging.
type RequestLogger struct {
	mu      sync.Mutex
	enabled bool
	entries []RequestLogEntry
	maxSize int
}

// RequestLogEntry is a single logged API call.
type RequestLogEntry struct {
	Timestamp    time.Time `json:"timestamp"`
	Provider     string    `json:"provider"`
	Model        string    `json:"model"`
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
	LatencyMs    int64     `json:"latency_ms"`
	Status       string    `json:"status"` // "success" or "error"
	Error        string    `json:"error,omitempty"`
	CacheHit     bool      `json:"cache_hit"`
}

// NewRequestLogger creates a logger.
func NewRequestLogger(enabled bool) *RequestLogger {
	return &RequestLogger{
		enabled: enabled,
		entries: make([]RequestLogEntry, 0, 100),
		maxSize: 500,
	}
}

// Log records a request.
func (rl *RequestLogger) Log(entry RequestLogEntry) {
	if !rl.enabled {
		return
	}
	rl.mu.Lock()
	defer rl.mu.Unlock()
	entry.Timestamp = time.Now()
	rl.entries = append(rl.entries, entry)
	if len(rl.entries) > rl.maxSize {
		rl.entries = rl.entries[len(rl.entries)-rl.maxSize:]
	}
}

// Recent returns the last N log entries.
func (rl *RequestLogger) Recent(n int) []RequestLogEntry {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if n > len(rl.entries) {
		n = len(rl.entries)
	}
	result := make([]RequestLogEntry, n)
	copy(result, rl.entries[len(rl.entries)-n:])
	return result
}

// Summary returns aggregate stats from the log.
func (rl *RequestLogger) Summary() string {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if len(rl.entries) == 0 {
		return "No API calls logged."
	}

	total := len(rl.entries)
	var errors, cacheHits int
	var totalLatency int64
	var totalIn, totalOut int

	for _, e := range rl.entries {
		if e.Status == "error" {
			errors++
		}
		if e.CacheHit {
			cacheHits++
		}
		totalLatency += e.LatencyMs
		totalIn += e.InputTokens
		totalOut += e.OutputTokens
	}

	avgLatency := totalLatency / int64(total)
	return fmt.Sprintf("API calls: %d (errors: %d, cache hits: %d, avg latency: %dms, tokens: %d in / %d out)",
		total, errors, cacheHits, avgLatency, totalIn, totalOut)
}
