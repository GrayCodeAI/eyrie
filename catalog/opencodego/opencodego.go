// Package opencodego holds shared constants and helpers for the OpenCode Go gateway
// (https://opencode.ai/docs/go/). Models are discovered via GET /v1/models; chat
// routing picks /v1/chat/completions or /v1/messages per model family.
package opencodego

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// DefaultBaseURL is the OpenCode Go API root.
const DefaultBaseURL = "https://opencode.ai/zen/go/v1"

// NativeModelID strips OpenCode config prefixes (opencode-go/kimi-k2.6 → kimi-k2.6).
func NativeModelID(id string) string {
	id = strings.TrimSpace(id)
	lower := strings.ToLower(id)
	for _, prefix := range []string{"opencode-go/", "opencodego/"} {
		if strings.HasPrefix(lower, prefix) {
			return strings.TrimSpace(id[len(prefix):])
		}
	}
	if i := strings.LastIndex(id, "/"); i >= 0 {
		return strings.TrimSpace(id[i+1:])
	}
	return id
}

// ProtocolForModel derives the API protocol from model name patterns.
// Returns "anthropic" or "openai". This is the heuristic fallback when
// the live API doesn't include protocol metadata.
// MiniMax and Qwen3.x use Anthropic /v1/messages; everything else uses OpenAI /v1/chat/completions.
func ProtocolForModel(modelID string) string {
	id := strings.ToLower(NativeModelID(modelID))
	if id == "" {
		return "openai"
	}
	if strings.Contains(id, "minimax") || strings.HasPrefix(id, "qwen3.") {
		return "anthropic"
	}
	return "openai"
}

// Dynamic protocol map — populated from live fetch results.
// Falls back to ProtocolForModel heuristic when no live data is available.
var (
	protocolMapMu    sync.RWMutex
	protocolMap      = map[string]string{} // native model ID → "anthropic" | "openai"
	protocolMapValid = false
)

// UpdateProtocolMap refreshes the dynamic protocol map from live fetch entries.
// Called after every successful FetchOpenCodeGo in the discover pipeline.
func UpdateProtocolMap(entries []struct{ ID, Protocol string }) {
	protocolMapMu.Lock()
	defer protocolMapMu.Unlock()
	for _, e := range entries {
		id := strings.ToLower(NativeModelID(e.ID))
		if id == "" || e.Protocol == "" {
			continue
		}
		protocolMap[id] = e.Protocol
	}
	protocolMapValid = true
}

// UsesMessagesAPI reports whether a model should use Anthropic /v1/messages on
// OpenCode Go (see opencode.ai/docs/go endpoints table).
//
// Resolution order:
//  1. Dynamic protocol map (populated from live /v1/models response)
//  2. Heuristic fallback (model name pattern matching)
func UsesMessagesAPI(modelID string) bool {
	id := strings.ToLower(NativeModelID(modelID))
	if id == "" {
		return false
	}

	// Check dynamic map first (from live fetch).
	protocolMapMu.RLock()
	if protocolMapValid {
		if proto, ok := protocolMap[id]; ok {
			protocolMapMu.RUnlock()
			return proto == "anthropic"
		}
	}
	protocolMapMu.RUnlock()

	// Fallback to heuristic.
	return ProtocolForModel(modelID) == "anthropic"
}

// ProtocolMapSnapshot returns a copy of the current protocol map for testing/debugging.
func ProtocolMapSnapshot() map[string]string {
	protocolMapMu.RLock()
	defer protocolMapMu.RUnlock()
	out := make(map[string]string, len(protocolMap))
	for k, v := range protocolMap {
		out[k] = v
	}
	return out
}

// ResetProtocolMap clears the dynamic protocol map. For testing only.
func ResetProtocolMap() {
	protocolMapMu.Lock()
	defer protocolMapMu.Unlock()
	protocolMap = map[string]string{}
	protocolMapValid = false
}

// --- Usage Limit Tracking ---
// Per https://opencode.ai/docs/go/ limits are dollar-denominated:
//   - 5-hour: $12
//   - Weekly: $30
//   - Monthly: $60

// UsageLimits defines the dollar-denominated spending caps.
type UsageLimits struct {
	FiveHourLimit float64
	WeeklyLimit   float64
	MonthlyLimit  float64
}

// DefaultUsageLimits returns the limits from the OpenCode Go docs.
func DefaultUsageLimits() UsageLimits {
	return UsageLimits{
		FiveHourLimit: 12.0,
		WeeklyLimit:   30.0,
		MonthlyLimit:  60.0,
	}
}

// UsageRecord tracks a single spend event.
type UsageRecord struct {
	Timestamp    time.Time
	Model        string
	InputTokens  int
	OutputTokens int
	CostUSD      float64
}

// UsageTracker tracks cumulative spend with sliding time windows.
type UsageTracker struct {
	mu      sync.Mutex
	records []UsageRecord
	limits  UsageLimits
}

// NewUsageTracker creates a tracker with the given limits.
func NewUsageTracker(limits UsageLimits) *UsageTracker {
	return &UsageTracker{limits: limits}
}

// Record adds a spend event.
func (t *UsageTracker) Record(r UsageRecord) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.records = append(t.records, r)
	// Prune records older than 30 days to bound memory.
	cutoff := time.Now().Add(-30 * 24 * time.Hour)
	i := 0
	for _, rec := range t.records {
		if rec.Timestamp.After(cutoff) {
			t.records[i] = rec
			i++
		}
	}
	t.records = t.records[:i]
}

// SpendInWindow returns total USD spent in the given duration window.
func (t *UsageTracker) SpendInWindow(window time.Duration) float64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	cutoff := time.Now().Add(-window)
	var total float64
	for _, r := range t.records {
		if r.Timestamp.After(cutoff) {
			total += r.CostUSD
		}
	}
	return total
}

// UsageStatus returns the current spend vs limits.
type UsageStatus struct {
	FiveHourSpend float64
	FiveHourLimit float64
	WeeklySpend   float64
	WeeklyLimit   float64
	MonthlySpend  float64
	MonthlyLimit  float64
}

// Status returns current spend vs all three limits.
func (t *UsageTracker) Status() UsageStatus {
	return UsageStatus{
		FiveHourSpend: t.SpendInWindow(5 * time.Hour),
		FiveHourLimit: t.limits.FiveHourLimit,
		WeeklySpend:   t.SpendInWindow(7 * 24 * time.Hour),
		WeeklyLimit:   t.limits.WeeklyLimit,
		MonthlySpend:  t.SpendInWindow(30 * 24 * time.Hour),
		MonthlyLimit:  t.limits.MonthlyLimit,
	}
}

// WouldExceedLimit checks if adding `additionalCost` would exceed any limit.
// Returns nil if OK, or an error describing which limit would be exceeded.
func (t *UsageTracker) WouldExceedLimit(additionalCost float64) error {
	s := t.Status()
	if s.FiveHourSpend+additionalCost > s.FiveHourLimit {
		return fmt.Errorf("opencodego: 5-hour limit $%.2f would be exceeded (current: $%.2f, adding: $%.2f)",
			s.FiveHourLimit, s.FiveHourSpend, additionalCost)
	}
	if s.WeeklySpend+additionalCost > s.WeeklyLimit {
		return fmt.Errorf("opencodego: weekly limit $%.2f would be exceeded (current: $%.2f, adding: $%.2f)",
			s.WeeklyLimit, s.WeeklySpend, additionalCost)
	}
	if s.MonthlySpend+additionalCost > s.MonthlyLimit {
		return fmt.Errorf("opencodego: monthly limit $%.2f would be exceeded (current: $%.2f, adding: $%.2f)",
			s.MonthlyLimit, s.MonthlySpend, additionalCost)
	}
	return nil
}

// EstimateCost estimates the USD cost for a request given token counts and model pricing.
func EstimateCost(model string, inputTokens, outputTokens int, inputPricePer1M, outputPricePer1M float64) float64 {
	return (float64(inputTokens)*inputPricePer1M + float64(outputTokens)*outputPricePer1M) / 1_000_000
}

// PricingTier holds pricing for a single tier (base or over-threshold).
type PricingTier struct {
	InputPer1M  float64
	OutputPer1M float64
}

// EstimateCostTiered estimates USD cost with tiered pricing support.
// When tierThreshold > 0 and total input tokens exceed it, the higher rate applies
// to the portion above the threshold. Same for output tokens.
func EstimateCostTiered(inputTokens, outputTokens int, base, tiered PricingTier, tierThreshold int) float64 {
	if tierThreshold <= 0 {
		// No tiering.
		return EstimateCost("", inputTokens, outputTokens, base.InputPer1M, base.OutputPer1M)
	}
	var cost float64
	// Input: base rate for first tierThreshold tokens, tiered rate for the rest.
	if inputTokens <= tierThreshold {
		cost += float64(inputTokens) * base.InputPer1M / 1_000_000
	} else {
		cost += float64(tierThreshold) * base.InputPer1M / 1_000_000
		cost += float64(inputTokens-tierThreshold) * tiered.InputPer1M / 1_000_000
	}
	// Output: same logic.
	if outputTokens <= tierThreshold {
		cost += float64(outputTokens) * base.OutputPer1M / 1_000_000
	} else {
		cost += float64(tierThreshold) * base.OutputPer1M / 1_000_000
		cost += float64(outputTokens-tierThreshold) * tiered.OutputPer1M / 1_000_000
	}
	return cost
}

// FormatStatus returns a human-readable spend summary.
func FormatStatus(s UsageStatus) string {
	return fmt.Sprintf(
		"5hr: $%.2f/$%.2f | weekly: $%.2f/$%.2f | monthly: $%.2f/$%.2f",
		s.FiveHourSpend, s.FiveHourLimit,
		s.WeeklySpend, s.WeeklyLimit,
		s.MonthlySpend, s.MonthlyLimit,
	)
}
