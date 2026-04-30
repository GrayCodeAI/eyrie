# Changelog

All notable changes to eyrie are documented here.  
Format: [Keep a Changelog](https://keepachangelog.com/en/1.0.0/) · Versioning: [SemVer](https://semver.org/)

---

## [0.0.1] — 2026-04-30

> Initial release.

### Added

**Clients**
- `Provider` interface — `Chat`, `StreamChat`, `Ping`, `Name` — composable and mockable
- `AnthropicClient` — Anthropic Messages API with full content block support
- `OpenAIClient` — OpenAI and all OpenAI-compatible providers
- `MockProvider` — testing without API keys (echo / fixed / tool_use / error / max_tokens modes)
- `EyrieClient` — thread-safe universal client with cached provider instances

**Streaming**
- SSE parser for Anthropic and OpenAI formats
- Tool call streaming — `input_json_delta` accumulation (Anthropic), index-based accumulation (OpenAI)
- Thinking block streaming (`thinking_delta`)
- `StreamResult` with `Close()` — no goroutine leaks
- Buffered channels (64) with goroutine-owned close

**Reliability**
- Retry — exponential backoff, full jitter, `Retry-After` header support
- `ChatWithContinuation` — auto-retry on `max_tokens` (up to 3 continuations)
- `WithRateLimit` — token bucket rate limiter decorator

**Caching**
- `AddCacheBreakpoints` — Anthropic `cache_control` on system prompt and conversation prefix

**Catalog**
- Embedded model catalog — pricing, context windows, max output for 8 providers
- Live fetch from OpenRouter and CanopyWave
- Model tier resolution — opus / sonnet / haiku → concrete model IDs
- Model name canonicalization and marketing display names
- Deprecation warnings per provider

**Config**
- Provider detection from env vars (priority order)
- `~/.hawk/provider.json` config file I/O
- `ApplyProviderConfigToEnv` — applies config to `os.Environ`
- OpenAI-compatible runtime resolution

**Quality**
- `User-Agent: eyrie/0.0.1` on all HTTP requests
- Request ID captured from response headers
- 4 KB error body cap — prevents OOM on large error responses
- `"eyrie: "` prefix on all errors with `%w` wrapping
- 43 tests · zero external dependencies · Go 1.26
