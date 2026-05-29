# AGENTS.md — Eyrie

Universal LLM provider runtime. One interface for every model. Authentication, routing, streaming, retries, caching — handled.

## Design Principles

- **Model-agnostic** — single interface for 75+ LLM providers
- **Zero opinions** — consumers control routing, caching, and retry strategies
- **Streaming-first** — all responses are streamed; blocking is opt-in

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

- `provider.go` — Provider interface and registry
- `routing.go` — Model routing and fallback chains
- `streaming.go` — SSE streaming with backpressure
- `auth.go` — API key management and rotation
- `cache.go` — Response caching (optional)
- `retry.go` — Retry with exponential backoff + Retry-After
- `catalog.go` — Model catalog and capability discovery

## Conventions

- Go 1.26+, pure Go, no CGO
- Table-driven tests
- Conventional Commits: `feat:`, `fix:`, `docs:`, `refactor:`, `test:`
- No `Co-authored-by:` trailers (auto-stripped by githook)
- `gofumpt` formatting enforced in CI
- Credential setup flow documented in `plans/CREDENTIAL-SETUP-FLOW.md`

## Common Pitfalls

- Provider interface is the boundary — keep it stable
- Streaming tests need careful goroutine management
- `go.work` points at hawk's `external/` for local dev
