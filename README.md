<div align="center">
  <h1>eyrie</h1>
  <p><strong>Universal LLM provider library for Go</strong></p>
  <p>The foundation layer powering <a href="https://github.com/hawk/hawk">hawk</a></p>

  <p>
    <a href="https://golang.org/"><img src="https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go"></a>
    <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue?style=flat-square" alt="License"></a>
    <a href="https://github.com/hawk/eyrie/actions"><img src="https://img.shields.io/github/actions/workflow/status/hawk/eyrie/ci.yml?style=flat-square&label=tests" alt="Tests"></a>
    <a href="https://pkg.go.dev/github.com/hawk/eyrie"><img src="https://img.shields.io/badge/godoc-reference-00ADD8?style=flat-square&logo=go" alt="GoDoc"></a>
  </p>
</div>

---

## Overview

eyrie abstracts 8+ LLM providers behind a single clean `Provider` interface — no external dependencies, pure Go stdlib.

```go
c := client.NewEyrieClient(&client.EyrieConfig{
    Provider: "anthropic",
    APIKey:   os.Getenv("ANTHROPIC_API_KEY"),
})

resp, err := c.Chat(ctx, []client.EyrieMessage{
    {Role: "user", Content: "Hello!"},
}, client.ChatOptions{Model: "claude-sonnet-4-6"})
```

## Features

| | |
|---|---|
| 🔌 **8 providers** | Anthropic, OpenAI, OpenRouter, Grok, Gemini, CanopyWave, Ollama, OpenCodeGo |
| 🌊 **Streaming** | SSE with tool calls, thinking blocks, accumulation by index |
| 🔁 **Retry** | Exponential backoff, jitter, `Retry-After` header |
| 💾 **Prompt caching** | Anthropic `cache_control` breakpoints |
| ♾️ **Continuation** | Auto-retry on `max_tokens` stop reason |
| 🚦 **Rate limiting** | Token bucket per provider via `WithRateLimit` decorator |
| 🧪 **Mock provider** | Test without API keys — echo, fixed, tool_use, error modes |
| 📦 **Model catalog** | Embedded pricing + live fetch from OpenRouter & CanopyWave |
| 🔍 **Auto-detection** | Provider detected from env vars in priority order |
| 🔒 **Zero deps** | Pure Go stdlib |

## Supported Providers

| Provider | Env Key | Type |
|----------|---------|------|
| Anthropic | `ANTHROPIC_API_KEY` | Native |
| OpenAI | `OPENAI_API_KEY` | Native |
| OpenRouter | `OPENROUTER_API_KEY` | OpenAI-compatible |
| Grok (xAI) | `XAI_API_KEY` | OpenAI-compatible |
| Gemini | `GEMINI_API_KEY` | OpenAI-compatible |
| CanopyWave | `CANOPYWAVE_API_KEY` | OpenAI-compatible |
| Ollama | `OLLAMA_BASE_URL` | OpenAI-compatible |
| OpenCodeGo | `OPENCODEGO_API_KEY` | OpenAI-compatible |

## Install

```bash
go get github.com/hawk/eyrie
```

Requires Go 1.26+.

## Usage

### Chat

```go
import "github.com/hawk/eyrie/client"

c := client.NewEyrieClient(&client.EyrieConfig{
    Provider: "anthropic",
    APIKey:   os.Getenv("ANTHROPIC_API_KEY"),
})

resp, err := c.Chat(ctx, []client.EyrieMessage{
    {Role: "system", Content: "You are a helpful assistant."},
    {Role: "user",   Content: "What is Go?"},
}, client.ChatOptions{Model: "claude-sonnet-4-6"})

fmt.Println(resp.Content)
fmt.Println(resp.Usage.TotalTokens)
```

### Streaming

```go
sr, err := c.StreamChat(ctx, messages, client.ChatOptions{Model: "claude-sonnet-4-6"})
if err != nil {
    panic(err)
}
defer sr.Close()

for evt := range sr.Events {
    switch evt.Type {
    case "content":
        fmt.Print(evt.Content)
    case "tool_call":
        fmt.Printf("\n[tool: %s %v]\n", evt.ToolCall.Name, evt.ToolCall.Arguments)
    case "thinking":
        fmt.Printf("\n[thinking: %s]\n", evt.Thinking)
    case "done":
        fmt.Println()
    case "error":
        log.Printf("stream error: %s", evt.Error)
    }
}
```

### Provider auto-detection

```go
// Detects from env vars: anthropic → openrouter → grok → gemini → canopywave → openai → opencodego → ollama
provider := client.DetectProvider()
```

### Prompt caching (Anthropic)

```go
// Sets cache_control on the conversation prefix — up to 5 min cache, significant cost savings
cachedMsgs := client.AddCacheBreakpoints(messages)
```

### Output continuation

```go
// Automatically continues when stop_reason == "max_tokens" (up to 3 continuations by default)
resp, err := client.ChatWithContinuation(ctx, provider, messages,
    client.ChatOptions{Model: "claude-sonnet-4-6"},
    client.DefaultContinuationConfig(),
)
```

### Rate limiting

```go
limiter := client.NewRateLimiter(client.RateLimitConfig{RequestsPerMinute: 60})

// Wrap any Provider — composes cleanly
rateLimited := client.WithRateLimit(myProvider, limiter)
```

### Testing with mock provider

```go
mock := client.NewMockProvider(client.MockModeFixed)
mock.Response = "mocked response"

resp, _ := mock.Chat(ctx, messages, opts)
fmt.Println(resp.Content)   // "mocked response"
fmt.Println(mock.CallCount()) // 1
fmt.Println(mock.LastCall().Options.Model)
```

### Model catalog

```go
import "github.com/hawk/eyrie/catalog"

cat := catalog.DefaultModelCatalog()                          // embedded
cat = catalog.LoadModelCatalogSync("/tmp/eyrie-catalog.json") // from cache

// Live fetch from OpenRouter + CanopyWave
cat, err := catalog.FetchModelCatalog("/tmp/eyrie-catalog.json", map[string]string{
    "OPENROUTER_API_KEY": os.Getenv("OPENROUTER_API_KEY"),
})

models := catalog.ModelsForProvider(&cat, "anthropic")
model  := catalog.GetProviderDefaultModel("anthropic", &cat)
warn   := catalog.GetModelDeprecationWarning("claude-3-7-sonnet", "anthropic")
```

### Provider config file

```go
import "github.com/hawk/eyrie/config"

cfg      := config.LoadProviderConfig("")           // reads ~/.hawk/provider.json
provider := config.DefaultProviderFromConfig(cfg)
config.ApplyProviderConfigToEnv(cfg, false, nil)    // applies to os.Environ
```

## Architecture

```
eyrie/
├── client/      Provider interface, Anthropic + OpenAI clients, mock,
│                streaming, retry, cache, continuation, rate limit
├── catalog/     Model catalog, tiers, names, deprecation, provider data
├── config/      Provider profiles, env config, OpenAI-compatible runtime
├── types/       Message types, content blocks, SDK types, usage, IDs
├── errors/      Error message constants and parsing utilities
├── constants/   API limits (image, PDF, media)
└── utils/       SSL error detection, API error sanitization
```

## Contributing

PRs welcome. Please run `go test ./... -race` before submitting.

## License

[MIT](LICENSE) © 2026 Hawk Contributors
