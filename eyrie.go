// Package eyrie is the core LLM client library for hawk.
//
// It provides API provider configurations, model resolution, API limits,
// base types (messages, IDs, connectors), and error types.
//
// Sub-packages:
//   - types: Message types, content blocks, usage, IDs, connectors
//   - errors: Error message constants and utilities
//   - constants: API limits (image, PDF, media)
//   - catalog: Model catalog, tiers, names, deprecation, provider data
//   - config: Provider configuration, profiles, env, OpenAI-compatible runtime
//   - client: EyrieClient, factory, provider detection
//   - utils: Error utilities (SSL detection, API error sanitization)
package eyrie

// Version of the eyrie library.
const Version = "0.1.0"
