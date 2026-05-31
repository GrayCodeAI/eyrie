# eyrie

LLM provider abstraction layer. Routes requests to multiple AI providers with reliability features.

## Build & Test
- `go test ./... -count=1` — run all tests
- `go test ./client/... -count=1 -run "TestGuardrail|TestCoalesce|TestStructured"` — new features

## Architecture
- `client/` — provider implementations (Anthropic, OpenAI), guardrails, coalescing
- `catalog/` — model catalog and routing
- `config/` — deployment configuration
- `credentials/` — API key management
- `router/` — request routing

## Key Patterns
- Provider interface: `Chat()`, `StreamChat()`, `Name()`, `Ping()`
- Guardrails: output validation (PII, secrets, injection, harmful content)
- Request coalescing: deduplicates identical concurrent LLM calls
- Structured output validation with retry-on-failure
- Lifecycle callbacks: 8 events

## Recent Additions
- Output guardrails framework
- Request coalescing
- Lifecycle callback hooks
- Structured output validation with retry
