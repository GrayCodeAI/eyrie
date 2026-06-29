package runtime

import (
	"context"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/GrayCodeAI/eyrie/catalog"
	"github.com/GrayCodeAI/eyrie/config"
	"github.com/GrayCodeAI/eyrie/credentials"
)

var chatProviderPreferenceOrder = []string{
	"openai",
	"anthropic",
	"openrouter",
	"grok",
	"gemini",
	"vertex",
	"bedrock",
	"zai_coding",
	"zai_payg",
	"canopywave",
	"deepseek",
	"azure",
	"opencodego",
	"kimi",
	"xiaomi_mimo_payg",
	"xiaomi_mimo_token_plan",
	"minimax_token_plan",
	"minimax_payg",
	"ollama",
}

// DefaultModelProviderFilter returns the catalog provider id to use when listing models
// with no explicit provider (e.g. /config model picker after paste-key).
// Order: provider.json default → first configured deployment (stable sort by id).
func DefaultModelProviderFilter(ctx context.Context) string {
	rt, err := Load(ctx)
	if err != nil || rt == nil {
		return ""
	}
	if rt.Provider != nil {
		if p := config.DefaultProviderFromConfig(rt.Provider); p != "" {
			return catalog.CanonicalProviderID(p)
		}
	}
	rows, err := rt.DeploymentRows()
	if err != nil || len(rows) == 0 {
		return ""
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	for _, row := range rows {
		if row.Configured {
			if p := catalog.CanonicalProviderID(row.ProviderID); p != "" {
				return p
			}
		}
	}
	return ""
}

// PreferredProvider returns the runtime-owned provider choice when a host has
// not pinned one explicitly. Active selection wins first, then inferred model
// ownership, then configured providers ordered by runtime preference, and
// finally credential detection as a last resort.
func PreferredProvider(ctx context.Context) string {
	if ctx == nil {
		ctx = context.Background()
	}
	if provider := normalizeRuntimeProviderID(ActiveProvider(ctx)); provider != "" && providerConfigured(ctx, provider) {
		return provider
	}
	if model := ActiveModel(ctx); model != "" {
		if provider := inferProviderForModel(ctx, model); provider != "" && providerConfigured(ctx, provider) {
			return provider
		}
	}
	if provider := preferredConfiguredProvider(ctx); provider != "" {
		return provider
	}
	return preferredDetectedProvider()
}

func preferredConfiguredProvider(ctx context.Context) string {
	rt, err := Load(ctx)
	if err != nil || rt == nil {
		return ""
	}
	rows, err := rt.DeploymentRows()
	if err != nil || len(rows) == 0 {
		return ""
	}
	configured := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		if !row.Configured {
			continue
		}
		if provider := catalog.CanonicalProviderID(row.ProviderID); provider != "" {
			configured[provider] = struct{}{}
		}
	}
	for _, provider := range chatProviderPreferenceOrder {
		if _, ok := configured[provider]; ok {
			return provider
		}
	}

	ordered := make([]string, 0, len(configured))
	for provider := range configured {
		ordered = append(ordered, provider)
	}
	sort.Strings(ordered)
	if len(ordered) == 0 {
		return ""
	}
	return ordered[0]
}

func preferredDetectedProvider() string {
	for _, provider := range chatProviderPreferenceOrder {
		switch provider {
		case "ollama":
			if runtimeEnvValue("OLLAMA_BASE_URL") != "" {
				return provider
			}
		default:
			profile, ok := runtimeProfileForProvider(provider)
			if !ok {
				continue
			}
			ready := true
			for _, envKey := range profile.DetectionEnv {
				if runtimeEnvValue(envKey) == "" {
					ready = false
					break
				}
			}
			if ready {
				return provider
			}
		}
	}
	return ""
}

func runtimeProfileForProvider(provider string) (config.RuntimeProviderProfile, bool) {
	switch provider {
	case "anthropic":
		return config.AnthropicRuntimeProfile, true
	case "openai":
		return config.OpenAIRuntimeProfile, true
	case "openrouter":
		return config.OpenRouterRuntimeProfile, true
	case "grok":
		return config.GrokRuntimeProfile, true
	case "gemini":
		return config.GeminiRuntimeProfile, true
	case "vertex":
		return config.VertexRuntimeProfile, true
	case "bedrock":
		return config.BedrockRuntimeProfile, true
	case "zai_coding":
		return config.ZAICodingRuntimeProfile, true
	case "zai_payg":
		return config.ZAIPaygRuntimeProfile, true
	case "canopywave":
		return config.CanopyWaveRuntimeProfile, true
	case "deepseek":
		return config.DeepSeekRuntimeProfile, true
	case "azure":
		return config.AzureRuntimeProfile, true
	case "opencodego":
		return config.OpenCodeGoRuntimeProfile, true
	case "kimi":
		return config.KimiRuntimeProfile, true
	case "xiaomi_mimo_payg":
		return config.XiaomiPaygRuntimeProfile, true
	case "xiaomi_mimo_token_plan":
		return config.XiaomiTokenPlanRuntimeProfile, true
	case "minimax_token_plan":
		return config.MiniMaxTokenPlanRuntimeProfile, true
	case "minimax_payg":
		return config.MiniMaxPaygRuntimeProfile, true
	default:
		return config.RuntimeProviderProfile{}, false
	}
}

func runtimeEnvValue(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if value := credentials.LookupSecret(ctx, key); value != "" {
		return value
	}
	return strings.TrimSpace(os.Getenv(key))
}
