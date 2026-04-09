# Eyrie Architecture

Eyrie is the provider/runtime and model-catalog layer consumed by Hawk and other TypeScript apps.

## 1) Context (C4-L1)

```mermaid
flowchart LR
  APP[Consumer App<br/>Hawk / Node service] --> E[Eyrie Library]
  E --> APIs[LLM Provider APIs]
  E --> CATALOGSRC[Catalog Source + Provider Model Endpoints]
```

## 2) Containers / Modules (C4-L2)

```mermaid
flowchart TB
  subgraph Eyrie["eyrie package"]
    IDX["Public API Surface<br/>src/index.ts"]
    CFG["Runtime Resolution<br/>src/config/openaiCompatibleRuntime.ts"]
    PROV["Provider Request Resolver<br/>src/config/providers.ts"]
    CAT["Model Catalog Engine<br/>src/catalog/modelCatalog.ts"]
    FACTORY["Provider Factory<br/>src/client/factory.ts"]
    CLIENT["Unified Client Facade<br/>src/client/index.ts"]
    TYPES["Types + Error Utilities<br/>src/types/* + src/errors/*"]
  end

  IDX --> CFG
  IDX --> PROV
  IDX --> CAT
  IDX --> FACTORY
  IDX --> CLIENT
  IDX --> TYPES
```

## 3) Runtime Resolution Flow (provider-scoped)

```mermaid
sequenceDiagram
  participant H as Hawk / Consumer
  participant R as resolveOpenAICompatibleRuntime
  participant P as resolveProviderRequest
  participant OUT as Resolved Runtime

  H->>R: env + optional model/baseUrl overrides
  R->>R: detect provider by scoped keys
  Note over R: Priority:<br/>OPENROUTER -> GROK/XAI -> GEMINI -> ANTHROPIC -> OPENAI -> OLLAMA
  R->>P: model + baseUrl + fallbackModel
  P-->>R: normalized request<br/>(transport, resolvedModel, baseUrl, reasoning)
  R-->>OUT: mode + request + apiKey + apiKeySource
  OUT-->>H: provider-safe runtime config
```

## 4) Model Catalog Flow

```mermaid
sequenceDiagram
  participant H as Hawk / Consumer
  participant C as fetchModelCatalog
  participant SRC as Remote Catalog JSON
  participant OR as OpenRouter /models
  participant FS as Local Cache File

  H->>C: fetchModelCatalog(cachePath, sourceUrl, env)
  C->>SRC: fetch base catalog
  SRC-->>C: providers + metadata
  C->>OR: optional live fetch (if OPENROUTER_API_KEY)
  OR-->>C: dynamic openrouter models
  C->>C: merge defaults + remote + dynamic entries
  C->>FS: write model_catalog.json
  C-->>H: normalized catalog
```

## 5) Design Notes

- Provider-scoped resolution prevents key/model leakage across providers.
- Catalog behavior is resilient: embedded defaults + cache fallback + best-effort live enrichment.
- Public API is centralized through `src/index.ts` to keep integration stable as internals evolve.
