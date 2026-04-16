<div align="center">

# 🦅 eyrie

**Provider Runtime + Model Catalog Layer — The foundation for multi‑LLM applications**

[![license](https://img.shields.io/github/license/GrayCodeAI/eyrie?style=flat-square&logo=open-source-initiative&logoColor=white&color=3BD550)](https://github.com/GrayCodeAI/eyrie/blob/main/LICENSE)
[![stars](https://img.shields.io/github/stars/GrayCodeAI/eyrie?style=flat-square&logo=github&color=FBBF24)](https://github.com/GrayCodeAI/eyrie/stargazers)
[![Discord](https://img.shields.io/badge/Discord-5865F2?style=flat-square&logo=discord&logoColor=white)](https://discord.gg/Fmq46SN8)
[![LinkedIn](https://img.shields.io/badge/LinkedIn-0A66C2?style=flat-square&logo=linkedin&logoColor=white)](https://www.linkedin.com/company/gray-code-ai)
[![X](https://img.shields.io/badge/Follow%20%40GrayCodeAI-000000?style=flat-square&logo=x&logoColor=white)](https://x.com/GrayCodeAI)

*Lightweight TypeScript toolkit used by Hawk for provider‑aware routing, catalog management, and runtime resolution*

[Installation](#installation) · [Features](#features) · [What's Included](#whats-included) · [Usage](#usage) · [Contributing](./CONTRIBUTING.md)

</div>

---

## ✨ What is eyrie?

**eyrie** is a **zero‑dependency TypeScript library** that provides the core runtime and catalog building blocks for LLM‑powered applications. Named after an eagle's nest, it sits at the foundation of your AI stack, offering clean abstractions for:

- 🎯 **Provider Management** – OpenAI, OpenRouter, Anthropic, Grok/xAI, Gemini, Ollama, and any OpenAI‑compatible API
- 📐 **Type Safety** – Fully typed message formats, content blocks, and API responses
- ⚡ **Constants & Utilities** – API limits, error handling, validation helpers
- 🏗️ **Provider Runtime Resolution** – Provider‑scoped key/model/base URL resolution
- 🔄 **Configuration I/O** – Load, save, and apply provider‑specific settings

Provider runtime precedence is **provider‑scoped**: OpenRouter, Grok, and Gemini keys are resolved **before** generic `OPENAI_API_KEY`, matching the integration model used by Hawk and Herm.

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

## 🔥 Features

<table>
<tr>
<td width="50%">

### 🎯 Universal Provider Support
Works with 8+ official provider profiles plus any OpenAI‑compatible endpoint. No vendor lock‑in.

### 📐 Full Type Safety
**64+ exports** with complete TypeScript definitions—messages, blocks, errors, and runtime helpers.

### 🏗️ Runtime Resolution
Intelligent provider detection based on model name, API keys, and base URLs.

</td>
<td width="50%">

### ⚡ Zero Dependencies
No runtime npm packages—just pure, tree‑shakable TypeScript.

### 🔄 Configuration‑First
Read/write user provider configs (`~/.hawk/provider.json`) with typed helpers.

### 🛡️ Error Utilities
Rich error classes, media‑size checks, rate‑limit handling, and localized error messages.

</td>
</tr>
</table>

---

## 📦 What's Included

| Category | Count | Description |
|----------|-------|-------------|
| **Constants** | 11 | API limits, sizes, thresholds |
| **Core Types** | 20 | Messages, blocks, tools |
| **SDK Types** | 30+ | API‑compatible types |
| **Functions** | 13 | Utilities, creators, validators |
| **Total** | **64** | Everything you need |

### 🧾 Constants (11)
API limits and configuration values:
```typescript
import {
  API_IMAGE_MAX_BASE64_SIZE,  // 5MB
  API_MAX_MEDIA_PER_REQUEST,   // 20 items
  API_PDF_MAX_PAGES,           // 100 pages
  // … and more
} from '@hawk/eyrie'
```

### 📝 Types (50+)
Complete type definitions for LLM interactions:
```typescript
import type {
  Message,
  ContentBlock,
  ToolUseBlock,
  APIError,
  // … and more
} from '@hawk/eyrie'
```

### 🛠️ Utilities (13)
Helper functions for common operations:
```typescript
import {
  createUserMessage,
  isTextBlock,
  resolveProviderRequest,
  parsePromptTooLongTokenCounts,
  // … and more
} from '@hawk/eyrie'
```

---

## 💡 Usage Examples

### Working with Messages
```typescript
import { createUserMessage, createAssistantMessage, isTextBlock } from '@hawk/eyrie'

// Create messages
const userMsg = createUserMessage('Hello, how are you?')
const assistantMsg = createAssistantMessage('I’m doing great! How can I help?')

// Type guards
if (isTextBlock(block)) {
  console.log(block.text)
}
```

### Provider Resolution
```typescript
import { resolveProviderRequest } from '@hawk/eyrie'

// Automatically detect provider from model
const provider = resolveProviderRequest({ model: 'gpt‑4o' })
// → { transport: 'chat_completions', resolvedModel: 'gpt‑4o', … }
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

### Provider Configuration I/O

```typescript
import {
  loadProviderConfig,
  saveProviderConfig,
  defaultProviderFromConfig,
  getProviderActiveModel,
  applyProviderConfigToEnv,
} from '@hawk/eyrie'

// Load user's provider configuration
const config = loadProviderConfig()
// → { anthropic_api_key: '…', active_provider: 'anthropic', … }

// Auto‑detect provider from configured keys
const provider = defaultProviderFromConfig(config)
// → 'anthropic' | 'openai' | 'openrouter' | …

// Get active model for a provider
const model = getProviderActiveModel(config, 'anthropic')
// → 'claude‑sonnet‑4‑6'

// Apply config to environment variables
applyProviderConfigToEnv(process.env, config)
// → Sets ANTHROPIC_API_KEY, ANTHROPIC_MODEL, etc.

// Save updated configuration
saveProviderConfig({ …config, active_model: 'new‑model' })
```

---

## 🧪 Testing & CI

```bash
# Run full CI pipeline (typecheck + build + tests)
npm run ci
```

The CI pipeline ensures type safety, build correctness, and runtime behavior.

---

## 🏗️ Architecture

See [ARCHITECTURE.md](./ARCHITECTURE.md) for component diagrams and provider‑flow details.

---

## 🎯 Design Principles

1. **🚀 Zero Dependencies** – No runtime npm packages, just pure TypeScript
2. **🔒 Type Safety** – Full type coverage with strict TypeScript
3. **📦 Self‑Contained** – Everything needed in one package
4. **⚡ Lightweight** – Tree‑shakable, only import what you use
5. **🔄 Provider Agnostic** – Works with any LLM provider
6. **✨ Developer Experience** – Clear APIs, great IntelliSense

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

[⭐ Star us on GitHub](https://github.com/GrayCodeAI/eyrie) · 
[💬 Join Discord](https://discord.gg/Fmq46SN8) · 
[💼 Follow on LinkedIn](https://www.linkedin.com/company/gray-code-ai) · 
[🐦 Follow on X](https://x.com/GrayCodeAI)

</div>