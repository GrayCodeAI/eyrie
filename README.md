<p align="center">
  <img src="assets/logo.svg" alt="eyrie" width="480"/>
</p>

<h1 align="center">Universal LLM Provider Runtime</h1>

<p align="center">
  One interface for every model. Authentication, routing, streaming, retries, caching — handled.
</p>

<p align="center">
  <a href="https://golang.org/"><img src="https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue?style=flat-square" alt="License"></a>
  <a href="https://github.com/GrayCodeAI/eyrie/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/GrayCodeAI/eyrie/ci.yml?style=flat-square&label=tests" alt="CI"></a>
  <a href="https://github.com/GrayCodeAI/eyrie/releases"><img src="https://img.shields.io/github/v/release/GrayCodeAI/eyrie?style=flat-square&label=release&color=green" alt="Release"></a>
  <a href="https://pkg.go.dev/github.com/GrayCodeAI/eyrie"><img src="https://img.shields.io/badge/godoc-reference-00ADD8?style=flat-square&logo=go" alt="GoDoc"></a>
</p>

<p align="center">
  <a href="#quick-start">Quick Start</a> ·
  <a href="#features">Features</a> ·
  <a href="#supported-providers">Providers</a> ·
  <a href="#usage">Usage</a> ·
  <a href="#architecture">Architecture</a> ·
  <a href="#contributing">Contributing</a>
</p>

---

## What is eyrie

eyrie is the LLM provider runtime that powers the [hawk](https://github.com/GrayCodeAI/hawk) coding agent. It handles everything between your application and LLM APIs — authentication, model resolution, streaming, retries, rate limiting, and caching.

When your app calls a model, eyrie figures out which provider to use, how to talk to it, and how to stream the response back. Switch from Anthropic to Ollama? eyrie handles the translation. API returns 529? eyrie retries with backoff. Response hits `max_tokens`? eyrie continues automatically.

**Your app never talks to an LLM API directly. eyrie does.**

## Quick Start

```bash
go get github.com/GrayCodeAI/eyrie
```

Requires Go 1.26+. Zero external dependencies.

```go
import "github.com/GrayCodeAI/eyrie/client"

// Create a client — provider auto-detected from environment
c := client.NewEyrieClient(&client.EyrieConfig{
    Provider: client.DetectProvider(),
})

// Stream a response
sr, err := c.StreamChat(ctx, messages, client.ChatOptions{
    Model: "claude-sonnet-4-6",
})
defer sr.Close()

for evt := range sr.Events {
    switch evt.Type {
    case "content":   // stream text
    case "tool_call": // execute tool
    case "done":      // response complete
    }
}
```

## Features

### Provider Routing

Automatically detects and routes to the right provider based on environment variables, config files, or explicit selection.

### Model Resolution

Maps abstract tiers (opus/sonnet/haiku) to concrete model IDs per provider. Ships with an embedded catalog of pricing, context windows, and capabilities.

### Streaming

Parses SSE for Anthropic and OpenAI formats — text, tool calls, and thinking blocks.

### Reliability

- Retries on 429/500/529 with exponential backoff and `Retry-After` support
- Auto-continuation when `stop_reason == max_tokens`
- Provider fallback chains for high availability

### Rate Limiting

Token bucket rate limiter per provider — prevents hitting API limits before they happen.

### Caching

- Response caching with configurable TTL
- Semantic similarity caching for repeated prompts
- Anthropic prompt caching breakpoints on system prompt and conversation prefix

### Cost Tracking

Built-in cost estimation per call, with per-provider pricing from the embedded model catalog.

## Supported Providers

| Provider | Env Variable | Notes |
|---|---|---|
| **Anthropic** | `ANTHROPIC_API_KEY` | Default · thinking, caching |
| **OpenAI** | `OPENAI_API_KEY` | Full tool use + reasoning |
| **OpenRouter** | `OPENROUTER_API_KEY` | 200+ models via one key |
| **Grok (xAI)** | `XAI_API_KEY` | |
| **Gemini** | `GEMINI_API_KEY` | |
| **CanopyWave** | `CANOPYWAVE_API_KEY` | |
| **Ollama** | `OLLAMA_BASE_URL` | Local models, no key needed |
| **OpenCodeGo** | `OPENCODEGO_API_KEY` | |

Providers are detected automatically in the order listed above.

## Usage

### Basic Chat

```go
resp, err := c.Chat(ctx, messages, client.ChatOptions{
    Model: "gpt-4o",
})
```

### Streaming with Continuation

```go
// Auto-continues when max_tokens is hit
resp, err := client.ChatWithContinuation(ctx, provider, messages,
    client.ChatOptions{Model: model},
    client.DefaultContinuationConfig(),
)
```

### Mock Provider for Testing

```go
mock := client.NewMockProvider(client.MockModeFixed)
mock.Response = "Here is the code you asked for..."

resp, _ := mock.Chat(ctx, messages, opts)
// No real API calls — perfect for tests
```

### Model Catalog

```go
cat := catalog.DefaultModelCatalog()

// Get the best model for a tier
model := catalog.GetPreferredProviderModel("anthropic", catalog.TierSonnet, &cat)
// → "claude-sonnet-4-6"

// Check deprecation warnings
warn := catalog.GetModelDeprecationWarning("claude-3-7-sonnet", "anthropic")
```

### Provider Configuration

```go
cfg := config.LoadProviderConfig("")             // load from disk
config.ApplyProviderConfigToEnv(cfg, false, nil) // apply to environment
config.SaveProviderConfig(cfg, "")               // save changes
```

## Architecture

```
eyrie/
├── cmd/eyrie/              # CLI binary
├── internal/
│   ├── client/             # Provider implementations (51 files)
│   │   ├── providers/      # Anthropic, OpenAI, Azure, Vertex, etc.
│   │   ├── middleware/     # Retry, rate limit, cache, fallback
│   │   └── metrics/        # Cost, call, analytics
│   ├── server/             # HTTP API and gateway
│   ├── config/             # Provider configuration & routing
│   ├── catalog/            # Model catalog & tier system
│   ├── registry/           # Runtime manifest & routing policies
│   ├── routing/            # Weighted provider router
│   ├── storage/            # SQLite conversation DAG store
│   ├── conversation/       # Conversation engine with branching
│   ├── observability/      # OpenTelemetry spans & metrics
│   ├── health/             # Provider health checker
│   ├── cache/              # Response cache warmer
│   ├── types/              # Branded types & API errors
│   ├── errors/             # Error message constants
│   ├── constants/          # API limits
│   ├── utils/              # Error utilities
│   └── sdk/                # Go, Python, TypeScript client SDKs
└── assets/                 # Logo and branding
```

## Ecosystem

eyrie is part of the hawk-eco:

| Component | Repository | Purpose |
|---|---|---|
| **hawk** | [GrayCodeAI/hawk](https://github.com/GrayCodeAI/hawk) | AI coding agent |
| **eyrie** | This repo | LLM provider runtime |
| **tok** | [GrayCodeAI/tok](https://github.com/GrayCodeAI/tok) | Tokenizer & compression |
| **yaad** | [GrayCodeAI/yaad](https://github.com/GrayCodeAI/yaad) | Graph-based memory |
| **trace** | [GrayCodeAI/trace](https://github.com/GrayCodeAI/trace) | Session capture |

## Development

### Prerequisites

- Go 1.26+

### Build & Test

```bash
go build ./cmd/eyrie          # Build binary
go test -race ./...           # Run all tests with race detector
make ci                       # Run full CI suite (lint, test, security)
make cover                    # Generate coverage report
```

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, commit conventions, and the PR process.

Quick start:

1. Fork and create a branch: `git checkout -b feat/short-description`
2. Make changes in small, focused commits
3. Run `make ci` locally
4. Open a pull request

Use [Conventional Commits](https://www.conventionalcommits.org/) for commit messages — release-please uses them for versioning.

## License

MIT — see [LICENSE](LICENSE) for details.

© 2026 GrayCode AI
