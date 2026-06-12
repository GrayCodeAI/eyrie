<div align="center">

# 🦅 eyrie Architecture

**Universal LLM Provider Runtime**

[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://go.dev/)
[![Port](https://img.shields.io/badge/Port-8080-orange)](https://www.iana.org/assignments/service-names-port-numbers/service-names-port-numbers.xhtml)
[![Protocol](https://img.shields.io/badge/Protocol-REST-blue)](https://swagger.io/specification/)

</div>

---

## 🎯 Overview

eyrie is the LLM provider runtime for the hawk ecosystem. It sits between the application and LLM APIs, handling **authentication**, **model resolution**, **streaming**, **retries**, **rate limiting**, and **caching**.

> 💡 No hawk ecosystem component talks to an LLM API directly — all communication goes through eyrie.

---

## 🧱 Components

```
eyrie/
├── api/openapi.yaml         📜 REST API contract (OpenAPI 3.1) — embedded HTTP server surface
├── client/
│   ├── client.go            🔌 Provider interface + EyrieClient factory
│   ├── anthropic.go         🟠 Anthropic Claude provider
│   ├── openai.go            🟢 OpenAI / OpenAI-compat provider
│   ├── gemini.go            🔵 Google Gemini provider
│   ├── bedrock.go           🟡 AWS Bedrock provider
│   ├── vertex.go            🔵 Google Vertex AI provider
│   ├── azure.go             🔷 Azure OpenAI provider
│   ├── provider_registry.go 🔍 Auto-detection + registration
│   ├── compat.go            🔧 Compatibility configs (Grok, OpenRouter, etc.)
│   ├── stream.go            📡 SSE stream parsing
│   ├── retry.go             🔄 Exponential backoff + Retry-After
│   ├── ratelimit.go         🪣 Token-bucket rate limiting per provider
│   ├── cache.go             💾 Response caching
│   ├── semantic_cache.go    🧠 Similarity-based cache lookup
│   ├── fallback.go          🔀 Provider fallback chains
│   └── errors.go            ❌ EyrieError type
├── catalog/                 📋 Model catalog — pricing, context windows, tiers
├── config/                  ⚙️ Configuration and credential resolution
├── conversation/            🌳 Conversation graph engine (branching DAG)
├── credentials/             🔑 API key management and env detection
├── router/                  🚦 Weighted provider routing
├── storage/                 🗄️ Conversation store (SQLite DAG)
└── internal/
    ├── api/                 🌐 HTTP server, route handlers, auth middleware
    ├── cache/               💾 Cache infrastructure
    ├── health/              💚 Provider health checker
    ├── observability/       📊 OpenTelemetry spans and metrics
    ├── shrink/              📦 Response compression
    └── version/             🏷️ Version constants
```

---

## 🌐 API

| | |
|---|---|
| **Contract** | [`api/openapi.yaml`](../api/openapi.yaml) |
| **Port** | `:8080` (default). Override: `eyrie serve <port>` |
| **Auth** | Bearer token or `X-API-Key` header. Set via `EYRIE_API_KEY` |

<details>
<summary><b>📡 Endpoint Summary</b></summary>

| Method | Path | Tag | Description |
|--------|------|-----|-------------|
| `GET` | `/health` | health | Health check |
| `POST` | `/prompt` | prompt | Execute a prompt at root |
| `POST` | `/nodes/{id}/prompt` | prompt | Continue from a node |
| `GET` | `/nodes` | nodes | List root nodes |
| `GET` | `/nodes/{id}` | nodes | Get a specific node |
| `DELETE` | `/nodes/{id}` | nodes | Delete node + descendants |
| `GET` | `/nodes/{id}/tree` | nodes | Get subtree |
| `PUT` | `/nodes/{id}/aliases/{alias}` | aliases | Create alias |
| `DELETE` | `/aliases/{alias}` | aliases | Delete alias |
| `GET` | `/api/usage` | analytics | Token usage analytics |
| `GET` | `/api/costs` | analytics | Cost breakdown |
| `GET` | `/api/health/providers` | providers | Provider health |

</details>

---

## 🔍 Provider Detection

Auto-detects active provider from env vars in priority order:

| Priority | Env Var | Provider |
|:--------:|---------|----------|
| 1 | `ANTHROPIC_API_KEY` | 🟠 Anthropic Claude |
| 2 | `OPENAI_API_KEY` | 🟢 OpenAI |
| 3 | `GEMINI_API_KEY` | 🔵 Google Gemini |
| 4 | `OPENROUTER_API_KEY` | 🔀 OpenRouter |
| 5 | `CANOPYWAVE_API_KEY` | 📡 CanopyWave |
| 6 | `XAI_API_KEY` | ⚡ Grok (xAI) |
| 7 | `ZAI_API_KEY` | 🤖 ZAI |
| 8 | — | 🦙 Ollama (localhost socket) |

---

## 📡 Streaming

All responses are streamed via **SSE**. Blocking responses wrap the stream internally.

```go
sr, err := client.StreamChat(ctx, messages, opts)
defer sr.Close()
for event := range sr.Events() { ... }
```

---

## 🔄 Retry & Rate Limiting

| Feature | Behavior |
|---------|----------|
| **Retries** | HTTP 429, 500, 502, 503, 529 |
| **Backoff** | Exponential + jitter |
| **Retry-After** | Respected on 429 responses |
| **Rate Limiting** | Per-provider token-bucket |

---

## 💾 Caching

| Layer | Strategy | Key |
|-------|----------|-----|
| **Exact** | Hash match | provider + model + message hash |
| **Semantic** | Cosine similarity | Prompt embeddings (optional, configurable TTL) |

---

## 🌳 Conversation Graph

Conversations are stored as a **DAG** in SQLite. Each prompt creates a `Node`; branching is first-class. Nodes are addressable by ID or named alias.
