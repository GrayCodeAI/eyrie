<div align="center">

# 🦅 eyrie

**Provider Runtime + Model Catalog Layer**

[![Version](https://img.shields.io/badge/version-1.0.1-blue.svg)](https://github.com/GrayCodeAI/eyrie)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.7-blue.svg)](https://www.typescriptlang.org/)
[![Zero Dependencies](https://img.shields.io/badge/dependencies-0-brightgreen.svg)](./package.json)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)
[![GitHub](https://img.shields.io/badge/GitHub-GrayCodeAI-black.svg)](https://github.com/GrayCodeAI)

*Provider-aware TypeScript toolkit used by Hawk for runtime routing and model catalogs*

[Installation](#installation) · [Usage](#usage) · [API](#api) · [Contributing](./CONTRIBUTING.md)

</div>

---

## ✨ What is eyrie?

**eyrie** is a lightweight TypeScript library that provides the runtime and catalog building blocks for LLM-powered applications. Named after an eagle's nest, it sits at the foundation of your AI stack, offering clean abstractions for:

- 🎯 **Provider Management** - Work with OpenAI, OpenRouter, Anthropic, Grok/xAI, Gemini, and Ollama
- 📐 **Type Safety** - Fully typed message formats, content blocks, and API responses
- ⚡ **Constants & Utilities** - API limits, error handling, validation helpers
- 🏗️ **Provider Runtime Resolution** - Provider-scoped key/model/base URL resolution

Provider runtime precedence is provider-scoped (OpenRouter/Grok/Gemini keys are resolved before generic `OPENAI_API_KEY`), matching the Hawk/Herm integration model.

> **Why eyrie?** Because building LLM apps shouldn't require juggling dozens of SDK versions. One clean interface, any provider.

---

## 🚀 Installation

```bash
# From GitHub (recommended)
npm install github:GrayCodeAI/eyrie#main

# or with yarn
yarn add github:GrayCodeAI/eyrie#main

# or with pnpm
pnpm add github:GrayCodeAI/eyrie#main
```

---

## 📦 What's Included

### Constants (11)
API limits and configuration values:
```typescript
import {
  API_IMAGE_MAX_BASE64_SIZE,  // 5MB
  API_MAX_MEDIA_PER_REQUEST,   // 20 items
  API_PDF_MAX_PAGES,           // 100 pages
  // ... and more
} from '@hawk/eyrie'
```

### Types (50+)
Complete type definitions for LLM interactions:
```typescript
import type {
  Message,
  ContentBlock,
  ToolUseBlock,
  APIError,
  // ... and more
} from '@hawk/eyrie'
```

### Utilities (13)
Helper functions for common operations:
```typescript
import {
  createUserMessage,
  isTextBlock,
  resolveProviderRequest,
  parsePromptTooLongTokenCounts,
  // ... and more
} from '@hawk/eyrie'
```

## ✅ Testing & CI

```bash
npm run ci
```

This runs typecheck + build + runtime tests used by CI.

---

## 💡 Usage Examples

### Working with Messages
```typescript
import { createUserMessage, createAssistantMessage, isTextBlock } from '@hawk/eyrie'

// Create messages
const userMsg = createUserMessage('Hello, how are you?')
const assistantMsg = createAssistantMessage('I'm doing great! How can I help?')

// Type guards
if (isTextBlock(block)) {
  console.log(block.text)
}
```

### Provider Resolution
```typescript
import { resolveProviderRequest } from '@hawk/eyrie'

// Automatically detect provider from model
const provider = resolveProviderRequest({ model: 'gpt-4o' })
// → { transport: 'chat_completions', resolvedModel: 'gpt-4o', ... }
```

### Error Handling
```typescript
import { APIError, isMediaSizeError, getImageTooLargeErrorMessage } from '@hawk/eyrie'

// Check error types
if (isMediaSizeError(error.message)) {
  console.error(getImageTooLargeErrorMessage())
}

// API error classes
throw new APIError(429, { message: 'Rate limited' }, 'Too many requests', {})
```

### Content Blocks
```typescript
import type { ContentBlock, TextBlock, ToolUseBlock } from '@hawk/eyrie'

const blocks: ContentBlock[] = [
  { type: 'text', text: 'Hello' },
  { type: 'tool_use', id: '1', name: 'search', input: { query: 'weather' } }
]
```

---

## 🏗️ Architecture

```
┌─────────────────────────────────────────┐
│           Your Application              │
│  (Hawk CLI, Web App, Server, etc.)      │
└──────────────────┬──────────────────────┘
                   │ imports @hawk/eyrie
                   ▼
┌─────────────────────────────────────────┐
│              @hawk/eyrie                │
│  ┌─────────────┐  ┌─────────────────┐  │
│  │   Types     │  │    Constants    │  │
│  │  • Message  │  │  • API Limits   │  │
│  │  • Content  │  │  • Error Msg    │  │
│  │  • Tools    │  │  • Provider     │  │
│  └─────────────┘  └─────────────────┘  │
│  ┌─────────────┐  ┌─────────────────┐  │
│  │  Utilities  │  │  Provider Config│  │
│  │  • Creators │  │  • Resolution   │  │
│  │  • Validators│  │  • Credentials │  │
│  │  • Parsers  │  │  • Auth         │  │
│  └─────────────┘  └─────────────────┘  │
└─────────────────────────────────────────┘
                   │
                   ▼ Zero npm dependencies
┌─────────────────────────────────────────┐
│           LLM Providers                 │
│  OpenAI • GrayCode • Codex • Ollama    │
└─────────────────────────────────────────┘
```

---

## 📊 Exports Overview

| Category | Count | Description |
|----------|-------|-------------|
| **Constants** | 11 | API limits, sizes, thresholds |
| **Core Types** | 20 | Messages, blocks, tools |
| **SDK Types** | 30+ | API-compatible types |
| **Functions** | 13 | Utilities, creators, validators |
| **Total** | **64** | Everything you need |

---

## 🎯 Design Principles

1. **🚀 Zero Dependencies** - No runtime npm packages, just pure TypeScript
2. **🔒 Type Safety** - Full type coverage with strict TypeScript
3. **📦 Self-Contained** - Everything needed in one package
4. **⚡ Lightweight** - Tree-shakeable, only import what you use
5. **🔄 Provider Agnostic** - Works with any LLM provider
6. **✨ Developer Experience** - Clear APIs, great IntelliSense

---

## 🛠️ Development

```bash
# Clone the repository
git clone https://github.com/GrayCodeAI/eyrie.git
cd eyrie

# Install dev dependencies
npm install

# Build the project
npm run build

# Watch mode for development
npm run dev

# Type check
npm run typecheck
```

---

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](./CONTRIBUTING.md) for details.

---

## 📄 License

MIT © [GrayCodeAI](https://github.com/GrayCodeAI)

---

<div align="center">

**Made with ❤️ for the AI development community**

[⭐ Star us on GitHub](https://github.com/GrayCodeAI/eyrie) · [🐦 Follow on X](https://x.com/Lakshman2302) · [💬 Join Discord](https://discord.gg/AyGB7TjA)

</div>
