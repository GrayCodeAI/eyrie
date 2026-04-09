# ADR 0001: Provider-Scoped Runtime Precedence

- Status: Accepted
- Date: 2026-04-09

## Context

Eyrie resolves OpenAI-compatible runtime settings for multiple providers. Generic compatibility variables (`OPENAI_*`) may coexist with provider-scoped keys, creating ambiguity and accidental cross-provider routing.

## Decision

Runtime provider detection is based on scoped-key precedence:

1. `OPENROUTER_API_KEY`
2. `GROK_API_KEY` / `XAI_API_KEY`
3. `GEMINI_API_KEY`
4. `ANTHROPIC_API_KEY`
5. `OPENAI_API_KEY`
6. `OLLAMA_BASE_URL` (keyless local mode)

`resolveOpenAICompatibleRuntime` remains the source of truth for mode, resolved request, and API key source.

## Consequences

- Predictable behavior when multiple keys are present.
- Reduced risk of stale `OPENAI_*` vars hijacking provider sessions.
- Consumers (including Hawk) get consistent resolution semantics.
