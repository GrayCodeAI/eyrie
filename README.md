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
  <a href="#documentation">Docs</a> ·
  <a href="#examples">Examples</a> ·
  <a href="#supported-providers">Providers</a> ·
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

## Documentation

Detailed documentation is available in the [docs/](docs/) directory:

- **[Architecture](docs/ARCHITECTURE.md)** — System design, data flow, and reliability features
- **[Provider Setup Guide](docs/guides/CREDENTIAL-SETUP-FLOW.md)** — Credential configuration and provider setup
- **[Dynamic Model Discovery](docs/guides/DYNAMIC-MODEL-DISCOVERY.md)** — Live model discovery architecture

## Examples

Runnable examples are in the [examples/](examples/) directory:

- **[Basic Chat](examples/basic/)** — Simple synchronous chat
- **[Streaming](examples/streaming/)** — SSE streaming with event handling
- **[Multi-Provider](examples/multi-provider/)** — Fallback chains across providers

Run any example with:

```bash
ANTHROPIC_API_KEY=sk-... go run ./examples/basic/
```

## Supported Providers

12 setup gateways in `catalog/registry/providers.go` (hawk `/config` uses the same list):

| Provider | ID | Env variable |
|---|---|---|
| **Anthropic** | `anthropic` | `ANTHROPIC_API_KEY` |
| **OpenAI** | `openai` | `OPENAI_API_KEY` |
| **Google Gemini** | `gemini` | `GEMINI_API_KEY` |
| **OpenRouter** | `openrouter` | `OPENROUTER_API_KEY` |
| **xAI (Grok)** | `grok` | `XAI_API_KEY` |
| **Z.AI** | `z-ai` | `ZAI_API_KEY` |
| **CanopyWave** | `canopywave` | `CANOPYWAVE_API_KEY` |
| **OpenCode Go** | `opencodego` | `OPENCODEGO_API_KEY` |
| **Kimi (Moonshot)** | `kimi` | `MOONSHOT_API_KEY` |
| **Xiaomi (MiMo) Pay-as-you-go** | `xiaomi_mimo_payg` | `XIAOMI_MIMO_PAYG_API_KEY` |
| **Xiaomi (MiMo) Token Plan** | `xiaomi_mimo_token_plan` | `XIAOMI_MIMO_TOKEN_PLAN_API_KEY` (+ region `cn` / `sgp` / `ams`) |
| **Ollama** | `ollama` | `OLLAMA_BASE_URL` (local; no API key) |

Runtime auto-detection uses a separate priority order for chat when no deployment is pinned; see `config` profiles.

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
├── client/                 # Provider client & streaming interface
├── config/                 # Provider configuration & routing
│   └── credential/         # Credential file management
├── catalog/                # Model catalog & tier system
│   ├── discover/           # Model discovery
│   ├── legacy/             # Legacy model support
│   ├── live/               # Live model data
│   └── registry/           # Model registry
├── codeagent/              # Code agent retry & fallback strategies
├── conversation/           # Conversation engine with branching
├── credentials/            # Credential management
├── docs/                   # Documentation & guides
├── examples/               # Runnable code examples
├── router/                 # Weighted provider router
├── runtime/                # Runtime manifest & routing policies
├── storage/                # SQLite conversation DAG store
├── types/                  # Branded types & API errors
├── errors/                 # Error message constants
├── constants/              # API limits
├── utils/                  # Error utilities
├── internal/
│   ├── api/                # HTTP API handlers
│   ├── cache/              # Response cache warmer
│   ├── health/             # Provider health checker
│   ├── observability/      # OpenTelemetry spans & metrics
│   ├── sdk/                # Go, Python, TypeScript client SDKs
│   └── version/            # Version information
└── assets/                 # Logo and branding
```

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for detailed system design and data flows.

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
