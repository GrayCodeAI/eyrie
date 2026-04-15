# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.1.0] - 2025-04-15

### Added
- **Provider Configuration I/O** - Full provider config file management
  - `loadProviderConfig()` / `saveProviderConfig()` - Read/write `~/.hawk/provider.json`
  - `defaultProviderFromConfig()` - Auto-detect provider from configured API keys
  - `getProviderActiveModel()` - Resolve active model with fallback logic
  - `applyProviderConfigToEnv()` - Apply config to environment variables
  - `clearProviderRuntimeEnv()` - Clean up provider env vars
  - `isProviderConfigured()` - Check if provider has valid configuration
- **Provider Detection Order** - Configurable priority for provider auto-detection
- **Model Catalog Integration** - Enhanced `applyProviderConfigToEnv` with catalog support

### Changed
- Centralized all provider config logic from Hawk CLI into eyrie
- Provider config now single source of truth for all consumers

## [1.0.0] - 2025-04-01

### Added
- Provider configuration with support for OpenAI, OpenRouter, Anthropic, Grok/xAI, Gemini, Ollama
- API limits and constants for images, PDFs, and media
- Branded types for SessionId and AgentId
- Connector text types and utilities
- Type-safe ID helpers (asSessionId, asAgentId, toAgentId)
- Model catalog with tier-based resolution (opus/sonnet/haiku)
- Runtime provider resolution for OpenAI-compatible providers

## [0.1.0] - 2025-01-07

### Added
- Initial release of eyrie
- Extracted from Hawk CLI codebase
- Core LLM client functionality
- TypeScript support with full type declarations
- Support for multiple LLM providers

[unreleased]: https://github.com/GrayCodeAI/eyrie/compare/v1.1.0...HEAD
[1.1.0]: https://github.com/GrayCodeAI/eyrie/releases/tag/v1.1.0
[1.0.0]: https://github.com/GrayCodeAI/eyrie/releases/tag/v1.0.0
[0.1.0]: https://github.com/GrayCodeAI/eyrie/releases/tag/v0.1.0
