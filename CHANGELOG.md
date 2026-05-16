# Changelog

All notable changes to eyrie are documented here.  
Format: [Keep a Changelog](https://keepachangelog.com/en/1.0.0/) · Versioning: [SemVer](https://semver.org/)

---

## [Unreleased]

### Changed
- **Version re-baselined to `0.2.0`** in `eyrie.go` (`const Version`) and
  `client/client.go` (`var Version`, used in the `User-Agent` header).
  Aligns eyrie with the rest of the hawk-eco ecosystem (`hawk`, `tok`,
  `yaad`, `sight`, `inspect`).

### Added — Production Hardening (top-50 OSS parity)
- Same-style hardening pass already on this branch:
  strict `golangci-lint` v2 config, unchecked-error fixes across
  `observability.go`, `sdk/go/client.go`, `storage/dag.go`,
  `storage/sqlite.go`, dead-code removal, and gofmt cleanup of the
  residual blank-line drift in `client/client.go`.
- `CONTRIBUTING.md` — development setup, branch flow, conventional
  commits, test/lint requirements.
- `CODE_OF_CONDUCT.md` — Contributor Covenant 2.1.
- `.gitattributes` — LF line-ending normalization, binary detection,
  GitHub linguist hints.
- `.github/dependabot.yml` — weekly `gomod` + `github-actions` updates.
- `.github/PULL_REQUEST_TEMPLATE.md` — Summary / Changes / Testing /
  Checklist.
- `.github/ISSUE_TEMPLATE/bug_report.yml` — structured bug report.
- `.github/ISSUE_TEMPLATE/feature_request.yml` — feature request with
  solo-developer fit checks.
- `.github/ISSUE_TEMPLATE/config.yml` — routes security reports to
  GitHub Security Advisories, questions to Discussions, blocks blank
  issues.

## [0.1.0] — 2026-05-12

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
- Fallback provider chains with automatic failover on retriable errors
- Provider health checking and success/failure stat tracking

**Caching**
- `AddCacheBreakpoints` — Anthropic `cache_control` on system prompt and conversation prefix
- Semantic caching for repeated queries
- Cache analytics (hit/miss stats)

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
- Provider profile management

**Advanced**
- Call metrics collector for provider usage tracking
- Role merge utility for message optimization
- Dynamic provider registration with registry freezing
- Weighted provider selection for load balancing
- Batch API support (Anthropic Message Batches)
- Cost estimator for pre-call cost prediction
- Embedding support

**Quality**
- `User-Agent: eyrie/0.1.0` on all HTTP requests
- Request ID captured from response headers
- 4 KB error body cap — prevents OOM on large error responses
- `"eyrie: "` prefix on all errors with `%w` wrapping
- Zero external dependencies · Go 1.26
