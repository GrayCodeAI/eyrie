package live

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchXiaomiPayg_MockHTTPServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("api-key") != "test-xiaomi-key" && r.Header.Get("Authorization") != "Bearer test-xiaomi-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		resp := struct {
			Data []json.RawMessage `json:"data"`
		}{
			Data: []json.RawMessage{
				json.RawMessage(`{"id":"mimo/model-x","display_name":"MiMo Model X","status":1}`),
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	platformSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	}))
	defer platformSrv.Close()

	entries, err := FetchXiaomiPayg(map[string]string{
		"XIAOMI_MIMO_PAYG_API_KEY":        "test-xiaomi-key",
		"XIAOMI_MIMO_PAYG_BASE_URL":       srv.URL,
		"XIAOMI_MIMO_PLATFORM_MODELS_URL": platformSrv.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 model, got %d", len(entries))
	}
	if entries[0].ID != "mimo/model-x" {
		t.Fatalf("id = %q", entries[0].ID)
	}
}

func TestFetchXiaomiPayg_EnrichesFromPlatformAPI(t *testing.T) {
	inferenceSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(struct {
			Data []json.RawMessage `json:"data"`
		}{Data: []json.RawMessage{json.RawMessage(`{"id":"mimo-v2.5","object":"model","owned_by":"xiaomi"}`)}})
	}))
	defer inferenceSrv.Close()

	platformSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []json.RawMessage{
				json.RawMessage(`{
					"id":"xiaomi/mimo-v2.5",
					"name":"Xiaomi MiMo:mimo-v2.5",
					"context_length":1048576,
					"max_output_length":131072,
					"pricing":{"prompt":"0.00000014","completion":"0.00000028"}
				}`),
			},
		})
	}))
	defer platformSrv.Close()

	entries, err := FetchXiaomiPayg(map[string]string{
		"XIAOMI_MIMO_PAYG_API_KEY":        "sk-test-key",
		"XIAOMI_MIMO_PAYG_BASE_URL":       inferenceSrv.URL,
		"XIAOMI_MIMO_PLATFORM_MODELS_URL": platformSrv.URL,
	})
	if err != nil || len(entries) != 1 {
		t.Fatalf("entries=%+v err=%v", entries, err)
	}
	if entries[0].ContextWindow != 1_048_576 {
		t.Fatalf("context=%d", entries[0].ContextWindow)
	}
	if entries[0].InputPricePer1M != 0.14 || entries[0].OutputPricePer1M != 0.28 {
		t.Fatalf("pricing=%v/%v", entries[0].InputPricePer1M, entries[0].OutputPricePer1M)
	}
	if !json.Valid(entries[0].RawJSON) || len(entries[0].RawJSON) < 40 {
		t.Fatalf("expected platform live_metadata, got %s", entries[0].RawJSON)
	}
}

func TestFetchXiaomiTokenPlan_RegionResolvesBase(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(struct {
			Data []json.RawMessage `json:"data"`
		}{Data: []json.RawMessage{json.RawMessage(`{"id":"mimo/tp-model"}`)}})
	}))
	defer srv.Close()

	platformSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	}))
	defer platformSrv.Close()

	entries, err := FetchXiaomiTokenPlan(map[string]string{
		"XIAOMI_MIMO_TOKEN_PLAN_API_KEY":  "tp-test-key",
		"XIAOMI_MIMO_TOKEN_PLAN_BASE_URL": srv.URL,
		"XIAOMI_MIMO_PLATFORM_MODELS_URL": platformSrv.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].ID != "mimo/tp-model" {
		t.Fatalf("entries = %+v err=%v", entries, err)
	}
}

func TestResolveTokenPlanOpenAIBase_StaleOverrideUsesRegion(t *testing.T) {
	base := resolveTokenPlanOpenAIBase(map[string]string{
		"XIAOMI_MIMO_TOKEN_PLAN_REGION":   "sgp",
		"XIAOMI_MIMO_TOKEN_PLAN_BASE_URL": "https://token-plan-cn.xiaomimimo.com/v1",
	})
	want := "https://token-plan-sgp.xiaomimimo.com/v1"
	if base != want {
		t.Fatalf("base = %q, want %s", base, want)
	}
}

func TestFetchXiaomiPayg_NoKey(t *testing.T) {
	entries, err := FetchXiaomiPayg(map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries with no key, got %d", len(entries))
	}
}
