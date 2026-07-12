# Host–Eyrie Engine Boundary

Status: accepted and being migrated incrementally.

## Decision

Eyrie is a host-neutral LLM runtime. A host such as Hawk owns its product UI,
agent loop, tool execution, permissions, conversation history, checkpoints, and
task semantics. Eyrie owns credentials, model discovery, capability matching,
provider/deployment routing, request and stream normalization, resilience,
usage, and provider telemetry.

```text
Host product ──► eyrie/engine ──► provider runtime ──► model APIs
```

The dependency is one-way. Eyrie must not import a host product.

## Stable host API

New host integrations should import `github.com/GrayCodeAI/eyrie/engine`.
Lower-level packages remain public during the compatibility migration, but a
host should not need to assemble `client`, `catalog`, `config`, `credentials`,
`router`, `runtime`, or `setup` directly.

The facade owns these stable concepts:

- `Engine`
- `GenerateRequest` and `GenerateResponse`
- `Message`, `Tool`, `ToolCall`, and `ToolResult`
- `Requirements` and `Preference`
- `Route`
- `Stream` and typed `Event`
- `CatalogSnapshot` and `Model`
- `CredentialStatus`
- typed `Error` and `ErrorCode`

## Selection contract

Hosts express requirements rather than provider-specific assumptions:

```text
requirements: streaming, tools, vision, structured JSON, reasoning, context
preference:   fast, balanced, reasoning, economical, provider/model override
```

An explicitly requested model is a hard constraint. Eyrie returns a capability
error when it is incompatible unless the host explicitly enables fallback.
Persisted/default selections may be replaced by a compatible catalog route.

## Conversation contract

Eyrie generation is stateless from the host's point of view:

```text
Host owns                         Eyrie owns
----------                        -----------
conversation history              route decision
tool execution                    provider request
permissions                       retries and fallback
checkpoints and replay            stream normalization
product memory                    usage and telemetry
```

Eyrie's generic `conversation` package remains available to other consumers,
but hosts with a product session model should not create two authoritative
conversation stores.

## Streaming contract

`Stream` is pull-based, cancellable, and must be closed. It emits a route event
before provider events and normalizes content, thinking, tool calls, usage,
TTFT, and completion. Unknown future event types are additive and must be
safely ignored by hosts.

Eyrie never executes a host tool. It emits a normalized tool request; the host
performs permission checks, executes it, appends the result to its history, and
starts the next generation.

## Credential contract

The facade accepts an injected `credentials.Store`. Secrets never belong in
provider configuration, model requests, logs, telemetry, or host tool
environments. Credential status values contain identifiers and booleans only.

## Compatibility migration

1. Add the facade without removing existing APIs.
2. Move host credential, catalog, selection, and transport calls to the facade.
3. Enforce new import boundaries while listing temporary compatibility
   exceptions.
4. Remove duplicated host provider routing and mirrored transport types.
5. Remove deprecated lower-level host entry points only at a semantic-version
   boundary.

Standalone Eyrie development is committed and tested first. Hosts then update
their pinned Eyrie submodule revision and run their integration suites.
