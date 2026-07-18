# AGENTS.md — Eyrie

Universal LLM provider runtime. One interface for every model. Authentication, routing, streaming, retries, caching — handled.

## Development workflow

When starting any new work (feature, fix, refactor, chore), always create a feature branch from `main` first. Never commit directly to `main`. Use branch naming conventions like `feat/<description>`, `fix/<description>`, or `chore/<description>`. Open a PR, ensure CI is green, then merge.

## Design Principles

- **Model-agnostic** — single interface for 75+ LLM providers
- **Host-neutral engine** — Eyrie owns provider routing, transport, caching,
  retry/fallback, and normalized telemetry; hosts own product UX and semantics
- **Streaming-first** — all responses are streamed; blocking is opt-in

## Observability

See [hawk/docs/OTEL-CONVENTIONS.md](https://github.com/GrayCodeAI/hawk/blob/main/docs/OTEL-CONVENTIONS.md) for the shared OpenTelemetry attribute vocabulary (`gen_ai.*`, `cost.usd`, etc.) used across all GrayCodeAI repos.

## Build & Test

```bash
go test ./...                    # Run all tests
go test -race ./...              # Race detector
go test -coverprofile=c.out ./... # Coverage
go vet ./...                     # Static analysis
gofumpt -w .                     # Format
make ci                          # Full CI suite
```

## Architecture

- `engine/` — stable host-facing provider engine facade and DTO contract
- `client/core/` — provider-neutral wire types, transport, stream, and retry primitives
- `client/adapters/` — provider protocol adapters and construction registry
- `client/` — backwards-compatible public facade, middleware, and caches
- `credentials/` — API key storage, lookup, and safe status projection
- `catalog/` — model catalog, discovery, capabilities, and pricing
- `router/` and `runtime/` — route policy and runtime resolution

## Conventions

- Go 1.26+, pure Go, no CGO
- Table-driven tests
- Conventional Commits: `feat:`, `fix:`, `docs:`, `refactor:`, `test:`
- No `Co-authored-by:` trailers (auto-stripped by githook)
- `gofumpt` formatting enforced in CI
- Credential setup flow documented in `docs/guides/CREDENTIAL-SETUP-FLOW.md`

## Common Pitfalls

- `engine` is Hawk's product boundary; Hawk must not assemble lower-level
  `client`, `catalog`, `config`, `credentials`, `router`, or `runtime` packages
- `client.Provider` remains the lower-level compatibility boundary for other
  consumers; preserve its method set and the facade's type identity
- Streaming tests need careful goroutine management
- `go.work` here should stay minimal; hawk's own `go.work` adds an `external/eyrie` replace so hawk can develop against a local eyrie checkout. Do not add extra local `replace` directives here without coordinating with hawk's workspace.

## Naming Conventions

- **Provider interface**: `client.Provider` with `Chat()`, `StreamChat()`, `Ping()`, `Name()` — implemented per LLM vendor
- **Client types**: `EyrieClient`, `EyrieMessage`, `EyrieResponse`, `EyrieTool`, `EyrieUsage` — `Eyrie` prefix for public types
- **Config struct**: `EyrieConfig` with `Provider`, `APIKey`, `BaseURL`, `Model`, `MaxRetries` fields
- **Provider implementations**: `AnthropicClient`, `OpenAIClient`, `GeminiClient`, `BedrockClient`, etc. — in `client/` package
- **Compatibility configs**: `OpenAICompat`, `GrokCompat`, `OpenRouterCompat` — `Compat` suffix for provider quirks
- **Error type**: `EyrieError` with `Provider`, `Op`, `StatusCode`, `RequestID`, `Message`, `Err` fields
- **Stream types**: `StreamResult`, `SSEEvent`, `StreamEvent` — streaming is SSE-based
- **Retry config**: `RetryConfig` embeds `types.RetryConfig` + adds `RetryOn []int` for HTTP status codes
- **Version wiring**: `client.Version` set via `SetVersion()` from root package — avoids circular import

## API Patterns

- **Provider auto-detection**: `DetectProvider()` checks env vars in priority order (ANTHROPIC_API_KEY, OPENAI_API_KEY, etc.)
- **Client creation**: `client.NewEyrieClient(&EyrieConfig{...})` or `client.Client(&EyrieConfig{...})` — both work
- **Chat method**: `c.Chat(ctx, messages, opts)` — non-streaming, returns `*EyrieResponse`
- **Stream method**: `c.StreamChat(ctx, messages, opts)` — returns `*StreamResult`, caller must `defer sr.Close()`
- **Auto-continuation**: `StreamChatContinue()` transparently retries when `stop_reason == max_tokens`
- **Provider fallback**: `fallback.go` implements fallback chains across providers
- **Rate limiting**: `ratelimit.go` implements token bucket per provider — prevents hitting API limits
- **Semantic caching**: `semantic_cache.go` caches similar prompts — optional, configurable TTL
- **Retry with backoff**: `retry.go` — exponential backoff + jitter, respects `Retry-After` header, retries on 429/500/502/503/529
- **Error hierarchy**: `EyrieError` has `IsRetriable()`, `IsAuthError()`, `IsRateLimited()` methods for programmatic handling
- **SSE parsing**: `parseSSEStream()` reads `bufio.Scanner` with 2MB buffer, emits `SSEEvent` to channel

## Testing Patterns

- **httptest.NewServer for provider tests**: mock LLM API responses with realistic JSON
- **Provider-specific test files**: `anthropic_test.go`, `openai_test.go`, `gemini_test.go` — each provider tested independently
- **Env var manipulation**: `os.Setenv`/`os.Unsetenv` in tests with `defer` cleanup — test provider detection
- **Credentials store**: `credentials.MapStore{}` for testing — `credentials.SetDefaultStore(store)` + `t.Cleanup(func() { ... })`
- **Client test structure**: create mock server, create client with server URL, call Chat/StreamChat, assert response
- **Header assertions**: verify `X-Api-Key`, `Anthropic-Version`, `User-Agent` headers are sent correctly
- **Stream tests**: parse SSE events from mock server, verify content/tool_call/done event types
- **Retry tests**: mock server returning 429/500, verify retry count and backoff behavior
- **Cache tests**: call same prompt twice, verify second call hits cache (no second HTTP request)
- **Fuzz tests**: `fuzz_test.go` for input parsing robustness

## Refactoring Guidelines

- **Safe to refactor**: `retry.go`, `ratelimit.go`, `cache.go` — internal infrastructure, no public API changes
- **Safe to refactor**: `stream.go` SSE parsing — internal implementation detail
- **Safe to refactor**: `fallback.go`, `weighted.go` — routing strategies, extend with new strategies
- **Safe to refactor**: `cost_estimator.go`, `cache_analytics.go` — metrics and tracking
- **Do not touch**: `Provider` interface (`Chat`, `StreamChat`, `Ping`, `Name`) — breaking change for all implementations
- **Do not touch**: `EyrieMessage`, `EyrieResponse`, `ChatOptions` struct field names — serialization contract
- **Do not touch**: `EyrieError` struct — used by consumers for error type assertions
- **Do not touch**: `client.EyrieConfig` — constructor contract for all consumers
- **Safe to extend**: add new provider implementations, new SSE event types, new cache strategies
- **When adding a provider**: create `client/<provider>.go`, implement `Provider` interface, register in `provider_registry.go`

## Key File Locations

| What | Where |
|---|---|
| Provider interface | `client/client.go` (`Provider`, `EyrieConfig`, `EyrieMessage`, `ContentPart`) |
| Chat implementation | `client/chat.go` (`Chat()`, `StreamChat()`, `StreamChatContinue()`) |
| Host-facing engine facade | `engine/` |
| Provider-neutral core | `client/core/` |
| Anthropic provider | `client/adapters/anthropic.go` |
| OpenAI provider | `client/adapters/openai.go` |
| Gemini provider | `client/adapters/gemini.go` |
| Bedrock provider | `client/adapters/bedrock.go` |
| Vertex provider | `client/adapters/vertex.go` |
| Azure provider | `client/adapters/azure.go` |
| Provider registry | `client/adapters/provider_registry.go` |
| Provider compatibility | `client/adapters/compat.go` (`OpenAICompat`, `GrokCompat`, etc.) |
| SSE streaming | `client/stream.go` (`parseSSEStream()`, `SSEEvent`) |
| Retry logic | `client/retry.go` (`RetryConfig`, `backoffDelay()`, `shouldRetry()`) |
| Rate limiting | `client/ratelimit.go`, `client/adaptive_ratelimit.go` |
| Caching | `client/cache.go`, `client/semantic_cache.go`, `client/cache_analytics.go` |
| Fallback chains | `client/fallback.go` |
| Auto-continuation | `client/continuation.go` |
| Error types | `client/errors.go` (`EyrieError`, `IsRetriable()`, `IsAuthError()`) |
| Error constants | `errors/errors.go` (API error messages, prompt-too-long parsing) |
| Model catalog | `catalog/` (pricing, context windows, capabilities per provider) |
| Credentials | `credentials/` (key storage, env detection, scrubbing) — `HasSecret` is silent on miss (boolean predicate); `LookupSecret` logs `Debug` on `ErrNotFound` and `Warn` on real backend errors |
| Mock provider | `client/mock.go` |
| Main test file | `client/client_test.go` (httptest servers, provider detection) |
| Linter config | `.golangci.yml` (govet, ineffassign, misspell — minimal) |

<!-- gitnexus:start -->
# GitNexus — Code Intelligence

This project is indexed by GitNexus as **eyrie** (9305 symbols, 33738 relationships, 300 execution flows). Use the GitNexus MCP tools to understand code, assess impact, and navigate safely.

> Index stale? Run `node .gitnexus/run.cjs analyze` from the project root — it auto-selects an available runner. No `.gitnexus/run.cjs` yet? `npx gitnexus analyze` (npm 11 crash → `npm i -g gitnexus`; #1939).

## Always Do

- **MUST run impact analysis before editing any symbol.** Before modifying a function, class, or method, run `impact({target: "symbolName", direction: "upstream"})` and report the blast radius (direct callers, affected processes, risk level) to the user.
- **MUST run `detect_changes()` before committing** to verify your changes only affect expected symbols and execution flows. For regression review, compare against the default branch: `detect_changes({scope: "compare", base_ref: "main"})`.
- **MUST warn the user** if impact analysis returns HIGH or CRITICAL risk before proceeding with edits.
- When exploring unfamiliar code, use `query({search_query: "concept"})` to find execution flows instead of grepping. It returns process-grouped results ranked by relevance.
- When you need full context on a specific symbol — callers, callees, which execution flows it participates in — use `context({name: "symbolName"})`.
- For security review, `explain({target: "fileOrSymbol"})` lists taint findings (source→sink flows; needs `analyze --pdg`).

## Never Do

- NEVER edit a function, class, or method without first running `impact` on it.
- NEVER ignore HIGH or CRITICAL risk warnings from impact analysis.
- NEVER rename symbols with find-and-replace — use `rename` which understands the call graph.
- NEVER commit changes without running `detect_changes()` to check affected scope.

## Resources

| Resource | Use for |
|----------|---------|
| `gitnexus://repo/eyrie/context` | Codebase overview, check index freshness |
| `gitnexus://repo/eyrie/clusters` | All functional areas |
| `gitnexus://repo/eyrie/processes` | All execution flows |
| `gitnexus://repo/eyrie/process/{name}` | Step-by-step execution trace |

## CLI

| Task | Read this skill file |
|------|---------------------|
| Understand architecture / "How does X work?" | `.claude/skills/gitnexus/gitnexus-exploring/SKILL.md` |
| Blast radius / "What breaks if I change X?" | `.claude/skills/gitnexus/gitnexus-impact-analysis/SKILL.md` |
| Trace bugs / "Why is X failing?" | `.claude/skills/gitnexus/gitnexus-debugging/SKILL.md` |
| Rename / extract / split / refactor | `.claude/skills/gitnexus/gitnexus-refactoring/SKILL.md` |
| Tools, resources, schema reference | `.claude/skills/gitnexus/gitnexus-guide/SKILL.md` |
| Index, status, clean, wiki CLI commands | `.claude/skills/gitnexus/gitnexus-cli/SKILL.md` |

<!-- gitnexus:end -->
