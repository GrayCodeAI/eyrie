package xiaomi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchPlatformModelsIndex_Mock(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	defer srv.Close()

	idx, err := FetchPlatformModelsIndex(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	pm, ok := idx["mimo-v2.5"]
	if !ok {
		t.Fatal("missing mimo-v2.5")
	}
	if pm.ContextLength != 1_048_576 {
		t.Fatalf("ctx=%d", pm.ContextLength)
	}
	if pm.InputPricePer1M != 0.14 || pm.OutputPricePer1M != 0.28 {
		t.Fatalf("price=%v/%v", pm.InputPricePer1M, pm.OutputPricePer1M)
	}
}

func TestApplyPlatformMetadata_MergesAndUsesPlatformRaw(t *testing.T) {
	rawInf := json.RawMessage(`{"id":"mimo-v2.5","object":"model"}`)
	platform := map[string]PlatformModel{
		"mimo-v2.5": {
			ID: "xiaomi/mimo-v2.5", Name: "MiMo V2.5", ContextLength: 1_000_000,
			InputPricePer1M: 0.14, OutputPricePer1M: 0.28,
			Raw: json.RawMessage(`{"id":"xiaomi/mimo-v2.5","context_length":1000000}`),
		},
	}
	_, _, ctx, _, in, out, meta := ApplyPlatformMetadata(
		"mimo-v2.5", "", "", 0, 0, 0, 0, rawInf, platform,
	)
	if ctx != 1_000_000 || in != 0.14 || out != 0.28 {
		t.Fatalf("merged=%d %v/%v", ctx, in, out)
	}
	if string(meta) != `{"id":"xiaomi/mimo-v2.5","context_length":1000000}` {
		t.Fatalf("meta=%s", meta)
	}
}

func TestNativeModelID(t *testing.T) {
	if NativeModelID("xiaomi/mimo-v2.5-pro") != "mimo-v2.5-pro" {
		t.Fatal()
	}
}