# Eyrie Components (C4-L3)

This document expands eyrie internals at component level for runtime resolution and model-catalog behavior.

## 1) Public API Surface and Module Wiring

```mermaid
flowchart LR
  IDX["src/index.ts"] --> CFG["config/openaiCompatibleRuntime.ts"]
  IDX --> PROV["config/providers.ts"]
  IDX --> CAT["catalog/modelCatalog.ts"]
  IDX --> FACT["client/factory.ts"]
  IDX --> CLIENT["client/index.ts"]
  IDX --> TYPES["types/* + errors/* + utils/*"]
```

## 2) Runtime Resolver Components

```mermaid
flowchart TB
  ENV["process.env / caller overrides"] --> DETECT["resolveRuntimeProvider()"]
  DETECT --> KEY["resolveProviderApiKey()"]
  DETECT --> MODEL["provider model/baseUrl env chains"]
  MODEL --> REQ["resolveProviderRequest()"]
  KEY --> OUT["ResolvedOpenAICompatibleRuntime"]
  REQ --> OUT
```

## 3) Catalog Engine Components

```mermaid
flowchart TB
  LOAD["loadModelCatalogSync(cachePath)"] --> FALLBACK["embedded default catalog"]
  FETCH["fetchModelCatalog(...)"] --> REMOTE["remote base catalog JSON"]
  FETCH --> ORLIVE["optional OpenRouter /models fetch"]
  REMOTE --> MERGE["merge defaults + remote + dynamic"]
  ORLIVE --> MERGE
  MERGE --> WRITE["write cache file"]
  MERGE --> OUT["ModelCatalog"]
```

## 4) Provider Factory and Client Facade

```mermaid
flowchart LR
  CALLER["Hawk / Consumer"] --> FACT["client/factory.ts<br/>detectProvider + createAnthropicClient"]
  CALLER --> FACADE["client/index.ts<br/>EyrieClient"]
  CONTRACT["errors + typed contracts"]:::ghost
  FACADE --> SDKS["SDK adapters<br/>OpenAI / Anthropic / Google / Mistral / Groq"]
  FACT --> SDKS
  FACADE --> CONTRACT

  classDef ghost fill:#fff,stroke:#bbb,color:#666;
```

## 5) Provider Configuration I/O Components

```mermaid
flowchart TB
    FILE["~/.hawk/provider.json"] --> LOAD["loadProviderConfig()"]
    LOAD --> PARSE["Parse ProviderConfig"]
    PARSE --> DETECT["defaultProviderFromConfig()"]
    DETECT --> ORDER["PROVIDER_DETECTION_ORDER"]
    ORDER --> CHECK["isProviderConfigured()"]
    CHECK --> MODEL["getProviderActiveModel()"]
    MODEL --> ENV["applyProviderConfigToEnv()"]
    ENV --> SET["setEnvValue() per provider"]
    SET --> OUT["process.env populated"]
    
    SAVE["saveProviderConfig()"] --> WRITE["Write to ~/.hawk/provider.json"]
```

### Provider Config Flow
1. **Load** - `loadProviderConfig()` reads `~/.hawk/provider.json`
2. **Detect** - `defaultProviderFromConfig()` finds first configured provider by priority
3. **Resolve** - `getProviderActiveModel()` gets provider-specific model with fallback
4. **Apply** - `applyProviderConfigToEnv()` sets environment variables for the runtime

## 6) Contract Boundary

- Runtime and provider behavior is exported from `src/index.ts`; consumers should not depend on internal file layout.
- Provider precedence and catalog strategy are intentionally centralized to keep all consumers consistent.
- **Provider configuration I/O is now fully centralized in eyrie** - consumers (Hawk, etc.) delegate all config operations to eyrie.
