// Dump full MiMo inference + platform catalog JSON.
package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/GrayCodeAI/eyrie/catalog/xiaomi"
	eyriecfg "github.com/GrayCodeAI/eyrie/config"
)

func main() {
	ctx := context.Background()
	env := eyriecfg.DiscoveryEnvMap(ctx)

	out := map[string]any{}

	platformURL := xiaomi.PlatformModelsURLFromEnv(env)
	platBody, platStatus, platErr := httpGet(ctx, platformURL, "")
	out["platform_catalog"] = map[string]any{
		"url":         platformURL,
		"http_status": platStatus,
		"error":       errStr(platErr),
		"body":        parseJSONBody(platBody),
	}

	key := strings.TrimSpace(env["XIAOMI_MIMO_TOKEN_PLAN_API_KEY"])
	region, _ := xiaomi.NormalizeRegion(env["XIAOMI_MIMO_TOKEN_PLAN_REGION"])
	base, baseErr := xiaomi.ResolveOpenAIBasePreferRegion(
		xiaomi.BillingTokenPlan, region, env["XIAOMI_MIMO_TOKEN_PLAN_BASE_URL"],
	)
	inf := map[string]any{
		"region":  env["XIAOMI_MIMO_TOKEN_PLAN_REGION"],
		"key_set": key != "",
	}
	if baseErr != nil {
		inf["error"] = baseErr.Error()
	} else {
		infURL := strings.TrimRight(base, "/") + "/models"
		inf["url"] = infURL
		body, status, err := httpGet(ctx, infURL, key)
		inf["http_status"] = status
		inf["error"] = errStr(err)
		inf["body"] = parseJSONBody(body)
	}
	out["inference_models"] = inf

	if baseErr == nil && key != "" {
		raw, err := xiaomi.FetchOpenAIModelsJSON(ctx, base, key)
		if err != nil {
			out["eyrie_merged_rows_error"] = err.Error()
		} else {
			platform, _ := xiaomi.FetchPlatformModelsIndex(ctx, platformURL)
			rows := make([]map[string]any, 0, len(raw))
			for _, r := range raw {
				var stub struct {
					ID string `json:"id"`
				}
				_ = json.Unmarshal(r, &stub)
				display, desc, ctxWin, maxOut, in, out, meta := xiaomi.ApplyPlatformMetadata(
					stub.ID, "", "", 0, 0, 0, 0, r, platform,
				)
				rows = append(rows, map[string]any{
					"id":                 stub.ID,
					"display_name":       display,
					"description":        desc,
					"context_window":     ctxWin,
					"max_output":         maxOut,
					"input_price_per_1m": in,
					"output_price_per_1m": out,
					"live_metadata":      parseJSONBody(meta),
					"inference_raw":      parseJSONBody(r),
				})
			}
			out["eyrie_merged_rows"] = rows
		}
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(out)
}

func httpGet(ctx context.Context, url, apiKey string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "eyrie-model-catalog/1.0")
	if apiKey != "" {
		req.Header.Set("api-key", apiKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	return body, resp.StatusCode, err
}

func parseJSONBody(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	var v any
	if json.Unmarshal(b, &v) == nil {
		return v
	}
	return string(b)
}

func errStr(err error) any {
	if err == nil {
		return nil
	}
	return err.Error()
}