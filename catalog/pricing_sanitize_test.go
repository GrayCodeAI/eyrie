package catalog

import (
	"testing"
	"time"
)

func TestSanitizePricingV1_negativeInputRemoved(t *testing.T) {
	t.Parallel()
	p := sanitizePricingV1(PricingV1{
		Status:     PricingKnown,
		Currency:   "USD",
		RatesPer1M: map[string]float64{"input_tokens": -1, "output_tokens": 2},
	})
	if _, ok := p.RatesPer1M["input_tokens"]; ok {
		t.Fatal("negative input_tokens should be removed")
	}
	if p.RatesPer1M["output_tokens"] != 2 {
		t.Fatalf("output_tokens = %v, want 2", p.RatesPer1M["output_tokens"])
	}
}

func TestPricingFromLegacy_negativeBecomesUnknown(t *testing.T) {
	t.Parallel()
	p := pricingFromLegacy(ModelCatalogEntry{
		ID:               "openrouter/auto",
		InputPricePer1M:  -5,
		OutputPricePer1M: 1,
	}, time.Now().UTC(), "test")
	if p.Status != PricingUnknown {
		t.Fatalf("status = %q, want unknown", p.Status)
	}
	if len(p.RatesPer1M) != 0 {
		t.Fatalf("expected no rates, got %v", p.RatesPer1M)
	}
}
