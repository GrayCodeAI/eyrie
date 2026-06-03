package xiaomi

import (
	"fmt"
	"strings"
	"testing"
)

func TestResolveAnthropicBase_MatchesOfficialPaths(t *testing.T) {
	// Payg + token plan bases; AnthropicClient posts to baseURL + "/v1/messages".
	payg, err := ResolveAnthropicBase(BillingPayAsYouGo, "")
	if err != nil || payg != PayAsYouGoAnthropicBase {
		t.Fatalf("payg anthropic = %q err=%v", payg, err)
	}
	want := payg + "/v1/messages"
	if want != "https://api.xiaomimimo.com/anthropic/v1/messages" {
		t.Fatalf("payg messages URL = %q", want)
	}
	cn, err := ResolveAnthropicBase(BillingTokenPlan, RegionCN)
	if err != nil || cn != TokenPlanCNAnthropicBase {
		t.Fatalf("cn anthropic = %q err=%v", cn, err)
	}
	if cn+"/v1/messages" != "https://token-plan-cn.xiaomimimo.com/anthropic/v1/messages" {
		t.Fatalf("cn messages URL mismatch")
	}
}

func TestResolveOpenAIBase(t *testing.T) {
	base, err := ResolveOpenAIBase(BillingPayAsYouGo, "", "")
	if err != nil || base != PayAsYouGoOpenAIBase {
		t.Fatalf("payg = %q err=%v", base, err)
	}
	base, err = ResolveOpenAIBase(BillingTokenPlan, RegionSGP, "")
	if err != nil || base != TokenPlanSGPOpenAIBase {
		t.Fatalf("sgp = %q err=%v", base, err)
	}
	override := "https://custom.example/v1"
	base, err = ResolveOpenAIBase(BillingTokenPlan, RegionCN, override)
	if err != nil || base != override {
		t.Fatalf("override = %q err=%v", base, err)
	}
}

func TestResolveOpenAIBasePreferRegion_IgnoresStaleOverride(t *testing.T) {
	staleCN := TokenPlanCNOpenAIBase
	got, err := ResolveOpenAIBasePreferRegion(BillingTokenPlan, RegionSGP, staleCN)
	if err != nil || got != TokenPlanSGPOpenAIBase {
		t.Fatalf("sgp prefer region = %q err=%v", got, err)
	}
	got, err = ResolveOpenAIBasePreferRegion(BillingTokenPlan, RegionSGP, "")
	if err != nil || got != TokenPlanSGPOpenAIBase {
		t.Fatalf("sgp empty override = %q err=%v", got, err)
	}
}

func TestAppendKeyMismatchHint(t *testing.T) {
	base := fmt.Errorf("credential probe failed: invalid API key (HTTP 401)")
	out := AppendKeyMismatchHint(base, ProviderPayAsYouGo, "tp-test")
	if out == nil || !strings.Contains(out.Error(), "Token Plan") {
		t.Fatalf("expected token plan hint, got %v", out)
	}
	if AppendKeyMismatchHint(base, ProviderPayAsYouGo, "sk-test") != base {
		t.Fatal("sk- on payg should not append hint")
	}
}

func TestKeyMismatchHint(t *testing.T) {
	if h := KeyMismatchHint(BillingPayAsYouGo, "tp-abc"); h == "" {
		t.Fatal("expected tp hint on payg")
	}
	if h := KeyMismatchHint(BillingTokenPlan, "sk-abc"); h == "" {
		t.Fatal("expected sk hint on token plan")
	}
}