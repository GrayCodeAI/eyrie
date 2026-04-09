# Eyrie Architecture

## Component Overview

```mermaid
flowchart LR
  APP[Consumer App<br/>Hawk / Node service / API server] --> E["eyrie package"]
  E --> T[Typed Models + Message Blocks]
  E --> K[Constants + Error Helpers]
  E --> RR[Provider Runtime Resolver]
  E --> MC[Model Catalog Builders]
```

## Runtime Resolution Flow

```mermaid
flowchart LR
  IN[Env + Profile + Requested Model] --> RR[resolveOpenAICompatibleRuntime]
  RR --> OUT[Resolved Request<br/>transport + base URL + model + key source]
  OUT --> P1[OpenAI-compatible APIs]
  OUT --> P2[OpenRouter]
  OUT --> P3[Anthropic compatible]
  OUT --> P4[xAI / Grok]
  OUT --> P5[Gemini compatible]
  OUT --> P6[Ollama]
```

## Scope

- Eyrie is the provider/runtime and model-catalog layer.
- It is designed for direct use by Hawk and any other TypeScript app.
