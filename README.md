# 🦅 @hawk/eyrie

<div align="center">

[![npm version](https://img.shields.io/npm/v/@hawk/eyrie.svg)](https://www.npmjs.com/package/@hawk/eyrie)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.7-blue.svg)](https://www.typescriptlang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![GitHub](https://img.shields.io/badge/GitHub-GrayCodeAI-black.svg)](https://github.com/GrayCodeAI)

**Core LLM client library for [Hawk](https://github.com/GrayCodeAI/hawk)**

*The engine that powers multi-provider AI interactions*

[Installation](#installation) • [Usage](#usage) • [API](#api) • [Architecture](#architecture)

</div>

---

## ✨ Features

- 🎯 **Zero UI Dependencies** - Pure logic, no React/Ink
- 🔒 **Type Safe** - Full TypeScript with branded types
- 🚀 **Multi-Provider** - OpenAI, Codex, Ollama support
- 📦 **Dependency-Free** - Only Node.js built-ins
- ⚡ **Production Ready** - Extracted from Hawk CLI

---

## 📦 Installation

```bash
# From npm (when published)
npm install @hawk/eyrie

# From GitHub (current)
npm install github:GrayCodeAI/eyrie#main
```

---

## 🚀 Quick Start

```typescript
import {
  // 🔧 Provider Resolution
  resolveProviderRequest,
  resolveCodexApiCredentials,
  
  // 🆔 Type-Safe IDs
  type AgentId,
  type SessionId,
  asSessionId,
  asAgentId,
  
  // 📏 API Limits
  API_IMAGE_MAX_BASE64_SIZE,
  API_MAX_MEDIA_PER_REQUEST,
} from '@hawk/eyrie'

// Resolve provider for any model
const provider = resolveProviderRequest({ model: 'gpt-4o' })
// → { transport: 'chat_completions', resolvedModel: 'gpt-4o', ... }

// Create branded IDs
const sessionId = asSessionId('sess-123') // ✓ Type-safe
const agentId = asAgentId('agent-456')    // ✓ Type-safe
```

---

## 📚 API Reference

### 🔧 Provider Configuration

Resolve which provider to use for any model:

```typescript
import { resolveProviderRequest, type ResolvedProviderRequest } from '@hawk/eyrie'

const config: ResolvedProviderRequest = resolveProviderRequest({
  model: 'gpt-4o',
  baseUrl: 'https://api.openai.com/v1' // optional
})

// Result:
// {
//   transport: 'chat_completions',
//   requestedModel: 'gpt-4o',
//   resolvedModel: 'gpt-4o',
//   baseUrl: 'https://api.openai.com/v1'
// }
```

### 🆔 Branded Types

Type-safe IDs prevent mixing different ID types:

```typescript
import { type AgentId, type SessionId, asAgentId, asSessionId } from '@hawk/eyrie'

function createSession(sessionId: SessionId, agentId: AgentId) {
  // ✓ Type-safe: Can't accidentally swap IDs
}

createSession(
  asSessionId('sess-abc'),  // ✓ Works
  asAgentId('agent-xyz')    // ✓ Works
)
```

### 📏 API Limits

Constants for validation and limits:

```typescript
import {
  API_IMAGE_MAX_BASE64_SIZE,    // 5MB
  IMAGE_MAX_WIDTH,              // 7680px
  IMAGE_MAX_HEIGHT,             // 7680px
  API_PDF_MAX_PAGES,            // 100 pages
  PDF_MAX_PAGES_PER_READ,       // 10 pages
  API_MAX_MEDIA_PER_REQUEST,    // 20 items
} from '@hawk/eyrie'

// Validate before sending to API
if (imageBase64.length > API_IMAGE_MAX_BASE64_SIZE) {
  throw new Error('Image too large')
}
```

### 🔌 Connector Types

Handle GrayCode connector messages:

```typescript
import { 
  type ConnectorTextBlock, 
  isConnectorTextBlock 
} from '@hawk/eyrie'

function processBlock(block: unknown) {
  if (isConnectorTextBlock(block)) {
    // block is now typed as ConnectorTextBlock
    console.log(block.text)
  }
}
```

---

## 🏗️ Architecture

```
┌─────────────────────────────────────────────┐
│              hawk (CLI App)                  │
│  ┌─────────┐ ┌──────────┐ ┌──────────┐     │
│  │  React  │ │ Commands │ │   Tools  │     │
│  │   Ink   │ │          │ │          │     │
│  └────┬────┘ └────┬─────┘ └────┬─────┘     │
└───────┼──────────┼──────────┼─────────────┘
        │          │          │
        └──────────┴──────────┘
                   │
                   ▼ imports @hawk/eyrie
┌─────────────────────────────────────────────┐
│              @hawk/eyrie                     │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐    │
│  │ Constants│ │  Types   │ │Providers │    │
│  │  (API    │ │ (Branded │ │ (Model   │    │
│  │  Limits) │ │   IDs)   │ │Resolution│    │
│  └──────────┘ └──────────┘ └──────────┘    │
└────────────────────┬────────────────────────┘
                     │
                     ▼ HTTP/API calls
┌─────────────────────────────────────────────┐
│         LLM Provider APIs                    │
│  OpenAI  •  Codex  •  Ollama  •  etc.       │
└─────────────────────────────────────────────┘
```

### Design Principles

1. **🎯 Zero UI Dependencies** - No React, Ink, or terminal logic
2. **🔒 Type Safety** - Branded types prevent ID confusion
3. **📦 Self-Contained** - Only Node.js built-ins (fs, os, path)
4. **⚡ Production Tested** - Extracted from working Hawk codebase
5. **🔄 Version Independent** - Can version separately from Hawk

---

## 🛠️ Development

```bash
# Clone the repository
git clone https://github.com/GrayCodeAI/eyrie.git
cd eyrie

# Install dependencies
npm install

# Build the project
npm run build

# Watch mode for development
npm run dev

# Type checking
npm run typecheck
```

---

## 📖 Related Projects

| Project | Language | Description |
|---------|----------|-------------|
| **[hawk](https://github.com/GrayCodeAI/hawk)** | TypeScript | Terminal UI for LLMs |
| **[langdag](https://github.com/anthropics/langdag)** | Go | LLM abstraction library |
| **[herm](https://github.com/anthropics/herm)** | Go | Terminal UI using langdag |

**Pattern:** `langdag` → `herm` (Go) | `eyrie` → `hawk` (TypeScript)

---

## 🤝 Contributing

This package is maintained as part of the Hawk project.

- 🐛 **Bug Reports:** [GrayCodeAI/hawk/issues](https://github.com/GrayCodeAI/hawk/issues)
- 💡 **Feature Requests:** [GrayCodeAI/hawk/discussions](https://github.com/GrayCodeAI/hawk/discussions)
- 📖 **Documentation:** See [Hawk repository](https://github.com/GrayCodeAI/hawk)

---

## 📄 License

MIT © [GrayCodeAI](https://github.com/GrayCodeAI)

---

<div align="center">

**Made with ❤️ by the Hawk team**

[⭐ Star us on GitHub](https://github.com/GrayCodeAI/eyrie) • [🐦 Follow on X](https://x.com/GrayCodeAI)

</div>
