# @hawk/eyrie

Core LLM client library for [Hawk](https://github.com/lakshmanpatel/hawk) - extracted to enable clean separation between the API layer and the CLI application.

## Overview

eyrie provides the foundational components needed to interact with LLM providers (OpenAI, Codex, etc.) in a type-safe, dependency-free manner. It's designed to be the "engine" that powers Hawk's terminal UI.

## Installation

```bash
npm install @hawk/eyrie
```

## Usage

```typescript
import {
  // Provider resolution
  resolveProviderRequest,
  resolveCodexApiCredentials,
  
  // Types
  type AgentId,
  type SessionId,
  
  // ID helpers
  asSessionId,
  asAgentId,
  toAgentId,
  
  // API limits
  API_IMAGE_MAX_BASE64_SIZE,
  API_MAX_MEDIA_PER_REQUEST,
} from '@hawk/eyrie'

// Resolve which provider to use for a model
const provider = resolveProviderRequest({ model: 'gpt-4o' })
// => { transport: 'chat_completions', resolvedModel: 'gpt-4o', ... }

// Create typed IDs
const sessionId = asSessionId('sess-123')
const agentId = asAgentId('agent-456')
```

## What's Included

### Constants (11)
API limits and configuration:
- `API_IMAGE_MAX_BASE64_SIZE` - Max size for base64-encoded images
- `IMAGE_MAX_WIDTH`, `IMAGE_MAX_HEIGHT` - Image dimension limits
- `PDF_TARGET_RAW_SIZE`, `API_PDF_MAX_PAGES` - PDF processing limits
- `API_MAX_MEDIA_PER_REQUEST` - Media attachments per request
- `DEFAULT_OPENAI_BASE_URL`, `DEFAULT_CODEX_BASE_URL` - Default endpoints

### Types (7)
Core type definitions:
- `SessionId`, `AgentId` - Branded types for IDs
- `ConnectorTextBlock`, `ConnectorTextDelta` - Connector message types
- `ProviderTransport`, `ResolvedProviderRequest`, `ResolvedCodexCredentials` - Provider types

### Functions (10)
Utility and resolution functions:
- `asSessionId()`, `asAgentId()`, `toAgentId()` - ID type helpers
- `isConnectorTextBlock()` - Type guard for connector blocks
- `resolveProviderRequest()` - Determine provider from model
- `resolveCodexApiCredentials()` - Get Codex auth credentials
- `isLocalProviderUrl()`, `isCodexBaseUrl()` - URL detection
- `resolveCodexAuthPath()`, `parseChatgptAccountId()` - Auth helpers

## Architecture

```
┌─────────────────────────────────────┐
│            hawk (CLI)               │
│  UI, commands, tools, state         │
└──────────────┬──────────────────────┘
               │ imports
               ↓
┌─────────────────────────────────────┐
│         @hawk/eyrie                 │
│  Constants, types, provider config  │
│  Zero UI dependencies               │
└──────────────┬──────────────────────┘
               │ uses
               ↓
┌─────────────────────────────────────┐
│      LLM Provider APIs              │
│  OpenAI, Codex, Ollama, etc.        │
└─────────────────────────────────────┘
```

## Development

### Building

```bash
npm install
npm run build
```

### Type Checking

```bash
npm run typecheck
```

### Watch Mode

```bash
npm run dev
```

## Design Principles

1. **Zero UI Dependencies** - eyrie knows nothing about terminals, React, or Ink
2. **Node.js Only** - Uses only built-in Node.js modules (fs, os, path)
3. **Type Safe** - Full TypeScript support with declaration files
4. **Self-Contained** - No external runtime dependencies except SDK
5. **Tested** - Extracted from production Hawk codebase

## Relationship to Hawk

eyrie was extracted from the [Hawk](https://github.com/lakshmanpatel/hawk) codebase following the same pattern as [langdag](https://github.com/anthropics/langdag) → [herm](https://github.com/anthropics/herm):

- **langdag** (Go) → **herm** (Go CLI) 
- **eyrie** (TypeScript) → **hawk** (TypeScript CLI)

This separation allows:
- Independent versioning of the core library
- Potential reuse in other projects
- Clearer architecture boundaries
- Easier testing of core logic

## License

MIT

## Contributing

This package is primarily maintained as part of the Hawk project. For issues and contributions, please refer to the main [Hawk repository](https://github.com/lakshmanpatel/hawk).
