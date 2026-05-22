# Credential setup: hawk (face) + eyrie (brain)

**Principle:** Hawk renders UI only. Eyrie owns providers, keys, validation, catalog, models.

See also: [DYNAMIC-MODEL-DISCOVERY.md](./DYNAMIC-MODEL-DISCOVERY.md)

## Supported providers (11 — from `catalog/registry`)

| Provider | Credential | Model strategy |
|----------|------------|----------------|
| Anthropic | `ANTHROPIC_API_KEY` | remote + live `/models` |
| OpenAI | `OPENAI_API_KEY` | remote + live `/models` |
| Google Gemini | `GEMINI_API_KEY` | remote + live API |
| OpenRouter | `OPENROUTER_API_KEY` | live-only (fallback remote) |
| xAI (Grok) | `XAI_API_KEY` | remote + live `/models` |
| Z.AI | `ZAI_API_KEY` | live-only |
| CanopyWave | `CANOPYWAVE_API_KEY` | live-only |
| OpenCode Go | `OPENCODEGO_API_KEY` | remote + live `/models` |
| Kimi (Moonshot) | `MOONSHOT_API_KEY` | live-only |
| Xiaomi (MiMo) | `XIAOMI_API_KEY` | live-only |
| Ollama (local) | `OLLAMA_BASE_URL` | live-only (`/api/tags`) |

Single source: `eyrie/catalog/registry/providers.go`

## Flow

```
/config hub
  → Paste API key → ResolveCredential → pick provider → SaveCredential → Discover → ListModels → pick model
  → Ollama local  → URL → SaveCredential → Discover → ListModels (live) → pick model
  → Pick model    → ListModels (auto) when credentials exist
```

## Host API (hawk uses `internal/eyrieclient` only)

- `ResolveCredentialForHost` / `SaveCredentialForHost`
- `ApplyEyrieCredentials`
- `ListModelsForProvider` — registry-driven live vs cache
- `LocalCredentialInference("ollama")`
- `FormatSetupError(provider, err)`

## Adding a new provider

1. Add one `ProviderSpec` row in `catalog/registry/providers.go`
2. If live list API exists: implement fetcher in `catalog/live/fetchers.go` and register key
3. Ensure models exist in remote catalog JSON (unless live-only)
4. No hawk changes
