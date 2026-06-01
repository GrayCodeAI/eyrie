# Eyrie Architecture

Eyrie is a Universal LLM Provider Runtime — the middleware layer between applications and LLM APIs. It handles provider routing, authentication, streaming, retries, caching, cost tracking, and conversation management.

---

## System Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        Application Layer                         │
│                    (hawk, custom apps, CLI)                       │
└────────────────────────────┬────────────────────────────────────┘
                             │
┌────────────────────────────▼────────────────────────────────────┐
│                         Eyrie Runtime                            │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────────┐   │
│  │  Router   │ │  Cache   │ │ Retry    │ │ Cost Tracker     │   │
│  │ (weighted │ │ (LRU +   │ │ (exp.   │ │ (per-call        │   │
│  │  + fallback│ │  semantic│ │  backoff │ │  estimation)     │   │
│  │  + circuit│ │  sim.)   │ │  + jitter│ │                  │   │
│  │  breaker) │ │          │ │          │ │                  │   │
│  └─────┬─────┘ └────┬─────┘ └────┬─────┘ └────────┬─────────┘   │
│        │            │            │                │             │
│  ┌─────▼────────────▼────────────▼────────────────▼─────────┐   │
│  │              Provider Interface                            │   │
│  │  Chat() | StreamChat() | Ping() | Name()                  │   │
│  └─────┬────────────┬────────────┬──────────────────────────┘   │
│        │            │            │                               │
│  ┌─────▼─────┐ ┌────▼─────┐ ┌───▼──────┐ ┌──────────────┐     │
│  │ Anthropic │ │  OpenAI  │ │  Gemini  │ │   Ollama     │     │
│  │  Client   │ │  Client  │ │  Client  │ │   Client     │     │
│  └───────────┘ └──────────┘ └──────────┘ └──────────────┘     │
└────────────────────────────┬────────────────────────────────────┘
                             │
┌────────────────────────────▼────────────────────────────────────┐
│                      LLM Provider APIs                          │
│         Anthropic | OpenAI | Google | xAI | Ollama | ...        │
└─────────────────────────────────────────────────────────────────┘
```

---

## Core Abstractions

### Provider Interface

The central abstraction — all LLM interactions go through this interface:

```go
type Provider interface {
    Chat(ctx context.Context, params MessageCreateParams) (*Message, error)
    StreamChat(ctx context.Context, params MessageCreateParams) (<-chan StreamEvent, error)
    Ping(ctx context.Context) error
    Name() string
}
```

### Decorator Pattern

Capabilities are composed via decoration:

```
Provider (base)
├── TracingProvider (adds OpenTelemetry spans)
├── CachedProvider (adds LRU response cache)
├── FallbackProvider (chains providers, tries next on transient errors)
├── Router (weighted load balancing + circuit breaker)
└── MockProvider (for testing)
```

---

## Package Structure

| Package | Purpose | Key Types |
|---------|---------|-----------|
| `client/` | Provider implementations, streaming, retry, caching | `AnthropicClient`, `OpenAIClient`, `CachedProvider` |
| `catalog/` | Model catalog, tiers, pricing, deprecation | `ModelCatalog`, `Tier`, `ModelInfo` |
| `config/` | Provider config, credentials, routing, profiles | `Config`, `ProviderConfig`, `Profile` |
| `conversation/` | Conversation engine with DAG branching | `Engine`, `Branch` |
| `credentials/` | Credential storage (keyring, env, map) | `Store`, `KeyringStore` |
| `router/` | Weighted provider routing, circuit breaker | `Router`, `CircuitBreaker` |
| `runtime/` | Runtime manifest, preflight checks | `Manifest`, `Preflight` |
| `storage/` | SQLite conversation DAG store | `Store`, `DAGNode` |
| `types/` | Branded types, API errors, retry config | `Message`, `APIError`, `TransientError` |
| `internal/` | API server, cache warmer, health checker, SDKs | `Server`, `CacheWarmer` |
| `setup/` | Setup UI, credential flow, deployment | `Setup`, `CredentialFlow` |

---

## Data Flow

### Chat Request

```
1. Application calls eyrie.Chat(params)
2. Router selects provider (weighted random + circuit breaker state)
3. CachedProvider checks LRU cache → cache hit? return cached response
4. Provider.Chat() makes HTTP request to LLM API
5. Retry middleware handles transient errors (429, 500, 529)
6. Response parsed, cost estimated, cache updated
7. TracingProvider records OpenTelemetry span
8. Response returned to application
```

### Streaming Request

```
1. Application calls eyrie.StreamChat(params)
2. Router selects provider
3. Provider.StreamChat() opens SSE connection
4. Events parsed from SSE stream (Anthropic or OpenAI format)
5. Events forwarded to application via channel
6. On max_tokens: auto-continuation (new request with conversation so far)
7. TracingProvider records span on completion
```

---

## Reliability

### Retry Strategy

- **Transient errors**: 429, 500, 502, 503, 504, 529
- **Backoff**: Exponential with jitter (base 1s, max 60s)
- **Max retries**: Configurable (default 3)
- **Retry-After**: Respects server-provided delay headers

### Circuit Breaker

- **States**: Closed → Open → Half-Open
- **Trip threshold**: 5 consecutive failures
- **Reset timeout**: 30 seconds
- **Half-open**: Single probe request

### Fallback Chain

```
Primary (Anthropic) → Secondary (OpenAI) → Tertiary (Gemini)
     ↓ (transient)         ↓ (transient)         ↓ (transient)
   Try next              Try next              Return error
```

---

## Caching

### Response Cache

- **Type**: LRU with TTL
- **Key**: Hash of (model, messages, tools, temperature)
- **TTL**: Configurable (default 5 minutes)
- **Max entries**: Configurable (default 1000)

### Semantic Similarity Cache

- **Type**: Vector similarity search
- **Threshold**: Configurable cosine similarity (default 0.95)
- **Backend**: In-memory with periodic persistence

### Anthropic Prompt Caching

- **Breakpoints**: Configurable cache control markers
- **Automatic**: Inserts breakpoints at system message and tool definitions

---

## Cost Tracking

### Per-Call Estimation

```go
type CostEstimate struct {
    InputCost    float64  // Based on input tokens × model price
    OutputCost   float64  // Based on output tokens × model price
    CacheSavings float64  // Savings from cache hits
    TotalCost    float64  // Net cost
}
```

### Model Catalog

Embedded catalog with pricing for all supported models:

```
Model                    Input $/1M   Output $/1M
claude-sonnet-4-20250514    3.00        15.00
gpt-4o                      2.50        10.00
gemini-2.0-flash            0.10         0.40
...
```

---

## Conversation DAG

Branching conversation storage in SQLite:

```
Root Message
├── Branch A (user edit)
│   ├── Assistant response
│   └── Branch A.1
└── Branch B (retry)
    └── Assistant response
```

- **Storage**: SQLite with WAL mode
- **Branching**: Prefix-based message IDs
- **Orphan detection**: Automatic cleanup of abandoned branches

---

## Security

- **Credential storage**: OS keyring (macOS Keychain, Linux Secret Service)
- **API key redaction**: Automatic in logs and error messages
- **TLS**: Configurable for API server mode
- **Rate limiting**: Token bucket per provider

---

## Observability

### OpenTelemetry Integration

- **Traces**: Per-request spans with provider, model, tokens, latency
- **Metrics**: Request count, latency histogram, error rate, cache hit rate
- **Exporters**: OTLP, Jaeger, Prometheus (configurable)

---

## Build & Release

- **Language**: Go 1.26+, zero CGO
- **Binary**: Single static binary (~15MB)
- **Platforms**: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64
- **Release**: GoReleaser with SHA-256 checksums
- **CI**: GitHub Actions (9 jobs: format, module, vet, lint, test, security, secrets, markdown, build matrix)
