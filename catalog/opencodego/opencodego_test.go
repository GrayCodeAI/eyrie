package opencodego

import (
	"testing"
	"time"
)

func TestNativeModelID(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"kimi-k2.6", "kimi-k2.6"},
		{"opencode-go/kimi-k2.7-code", "kimi-k2.7-code"},
		{"opencodego/glm-5.1", "glm-5.1"},
		{"  opencode-go/deepseek-v4-flash  ", "deepseek-v4-flash"},
	}
	for _, tc := range tests {
		if got := NativeModelID(tc.in); got != tc.want {
			t.Errorf("NativeModelID(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestProtocolForModel(t *testing.T) {
	tests := []struct {
		model string
		want  string
	}{
		{"kimi-k2.6", "openai"},
		{"glm-5.1", "openai"},
		{"deepseek-v4-flash", "openai"},
		{"mimo-v2.5", "openai"},
		{"minimax-m2.7", "anthropic"},
		{"minimax-m3", "anthropic"},
		{"qwen3.7-max", "anthropic"},
		{"qwen3.6-plus", "anthropic"},
		{"qwen3.5-plus", "anthropic"},
		{"opencode-go/kimi-k2.6", "openai"},
		{"opencodego/minimax-m2.5", "anthropic"},
	}
	for _, tc := range tests {
		if got := ProtocolForModel(tc.model); got != tc.want {
			t.Errorf("ProtocolForModel(%q) = %q, want %q", tc.model, got, tc.want)
		}
	}
}

func TestUsesMessagesAPI_HeuristicFallback(t *testing.T) {
	// Reset map so we test the heuristic fallback.
	ResetProtocolMap()
	tests := []struct {
		model string
		want  bool
	}{
		{"kimi-k2.6", false},
		{"glm-5.1", false},
		{"deepseek-v4-flash", false},
		{"mimo-v2.5", false},
		{"minimax-m2.7", true},
		{"minimax-m3", true},
		{"qwen3.7-max", true},
		{"qwen3.6-plus", true},
		{"qwen3.5-plus", true},
		{"opencode-go/kimi-k2.6", false},
		{"opencodego/minimax-m2.5", true},
	}
	for _, tc := range tests {
		if got := UsesMessagesAPI(tc.model); got != tc.want {
			t.Errorf("UsesMessagesAPI(%q) = %v, want %v", tc.model, got, tc.want)
		}
	}
}

func TestUsesMessagesAPI_DynamicMapOverrides(t *testing.T) {
	ResetProtocolMap()
	// Simulate live fetch returning protocol data.
	UpdateProtocolMap([]struct{ ID, Protocol string }{
		{"kimi-k2.6", "openai"},
		{"minimax-m3", "anthropic"},
		{"new-unknown-model", "anthropic"}, // not in heuristic
	})

	// kimi-k2.6 should be openai (from map).
	if UsesMessagesAPI("kimi-k2.6") {
		t.Error("kimi-k2.6 should be openai")
	}
	// minimax-m3 should be anthropic (from map).
	if !UsesMessagesAPI("minimax-m3") {
		t.Error("minimax-m3 should be anthropic")
	}
	// new-unknown-model should be anthropic (from map, not heuristic).
	if !UsesMessagesAPI("new-unknown-model") {
		t.Error("new-unknown-model should be anthropic from dynamic map")
	}
	// Completely unknown model falls back to heuristic.
	if UsesMessagesAPI("totally-new-model") {
		t.Error("totally-new-model should default to openai (heuristic fallback)")
	}

	ResetProtocolMap()
}

func TestProtocolMapSnapshot(t *testing.T) {
	ResetProtocolMap()
	UpdateProtocolMap([]struct{ ID, Protocol string }{
		{"kimi-k2.6", "openai"},
		{"minimax-m3", "anthropic"},
	})
	snap := ProtocolMapSnapshot()
	if snap["kimi-k2.6"] != "openai" {
		t.Errorf("snapshot kimi-k2.6 = %q, want openai", snap["kimi-k2.6"])
	}
	if snap["minimax-m3"] != "anthropic" {
		t.Errorf("snapshot minimax-m3 = %q, want anthropic", snap["minimax-m3"])
	}
	ResetProtocolMap()
}

func TestUsageTracker_RecordAndSpend(t *testing.T) {
	tracker := NewUsageTracker(DefaultUsageLimits())
	now := time.Now()

	tracker.Record(UsageRecord{Timestamp: now, CostUSD: 5.0})
	tracker.Record(UsageRecord{Timestamp: now.Add(-1 * time.Hour), CostUSD: 3.0})
	tracker.Record(UsageRecord{Timestamp: now.Add(-3 * 24 * time.Hour), CostUSD: 10.0}) // > 5hr, within week

	s := tracker.Status()
	if s.FiveHourSpend != 8.0 {
		t.Errorf("5hr spend = %.2f, want 8.0", s.FiveHourSpend)
	}
	if s.WeeklySpend != 18.0 {
		t.Errorf("weekly spend = %.2f, want 18.0", s.WeeklySpend)
	}
	if s.MonthlySpend != 18.0 {
		t.Errorf("monthly spend = %.2f, want 18.0", s.MonthlySpend)
	}
}

func TestUsageTracker_WouldExceedLimit(t *testing.T) {
	tracker := NewUsageTracker(UsageLimits{
		FiveHourLimit: 12.0,
		WeeklyLimit:   30.0,
		MonthlyLimit:  60.0,
	})
	now := time.Now()

	// Add $10 within 5 hours.
	tracker.Record(UsageRecord{Timestamp: now, CostUSD: 10.0})

	// Adding $1 should be OK ($11 < $12).
	if err := tracker.WouldExceedLimit(1.0); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Adding $3 should exceed 5-hour limit ($13 > $12).
	if err := tracker.WouldExceedLimit(3.0); err == nil {
		t.Error("expected error for 5-hour limit exceeded")
	}
}

func TestEstimateCost(t *testing.T) {
	// $1.40/1M input, $4.40/1M output (GLM-5.1 pricing)
	cost := EstimateCost("glm-5.1", 1000, 500, 1.40, 4.40)
	// (1000 * 1.40 + 500 * 4.40) / 1_000_000 = (1400 + 2200) / 1_000_000 = 0.0036
	if cost < 0.0035 || cost > 0.0037 {
		t.Errorf("cost = %.6f, want ~0.0036", cost)
	}
}

func TestFormatStatus(t *testing.T) {
	s := UsageStatus{
		FiveHourSpend: 5.50, FiveHourLimit: 12.0,
		WeeklySpend: 15.25, WeeklyLimit: 30.0,
		MonthlySpend: 42.00, MonthlyLimit: 60.0,
	}
	got := FormatStatus(s)
	want := "5hr: $5.50/$12.00 | weekly: $15.25/$30.00 | monthly: $42.00/$60.00"
	if got != want {
		t.Errorf("FormatStatus = %q, want %q", got, want)
	}
}

func TestEstimateCostTiered_NoThreshold(t *testing.T) {
	// No tiering — should match simple EstimateCost.
	base := PricingTier{InputPer1M: 0.50, OutputPer1M: 1.50}
	cost := EstimateCostTiered(1000, 500, base, PricingTier{}, 0)
	expected := EstimateCost("", 1000, 500, 0.50, 1.50)
	if cost != expected {
		t.Errorf("cost = %.6f, want %.6f", cost, expected)
	}
}

func TestEstimateCostTiered_WithinThreshold(t *testing.T) {
	// Qwen3.7 Plus: $0.40/1M input, $1.60/1M output below 256K
	//               $1.20/1M input, $4.80/1M output above 256K
	base := PricingTier{InputPer1M: 0.40, OutputPer1M: 1.60}
	tiered := PricingTier{InputPer1M: 1.20, OutputPer1M: 4.80}
	threshold := 256000

	// 100K input, 50K output — all within threshold.
	cost := EstimateCostTiered(100000, 50000, base, tiered, threshold)
	// (100000 * 0.40 + 50000 * 1.60) / 1_000_000 = (40000 + 80000) / 1_000_000 = 0.12
	if cost < 0.119 || cost > 0.121 {
		t.Errorf("cost = %.6f, want ~0.12", cost)
	}
}

func TestEstimateCostTiered_AboveThreshold(t *testing.T) {
	base := PricingTier{InputPer1M: 0.40, OutputPer1M: 1.60}
	tiered := PricingTier{InputPer1M: 1.20, OutputPer1M: 4.80}
	threshold := 256000

	// 300K input, 100K output — input exceeds threshold.
	cost := EstimateCostTiered(300000, 100000, base, tiered, threshold)
	// Input: 256000 * 0.40 + 44000 * 1.20 = 102400 + 52800 = 155200 / 1_000_000 = 0.1552
	// Output: 100000 * 1.60 = 160000 / 1_000_000 = 0.16
	// Total: 0.3152
	if cost < 0.314 || cost > 0.316 {
		t.Errorf("cost = %.6f, want ~0.3152", cost)
	}
}
