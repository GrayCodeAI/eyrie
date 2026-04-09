/**
 * Eyrie client factory – mirrors the herm/langdag pattern exactly.
 *
 * Provider is detected by which API key is set (first match wins),
 * not by HAWK_CODE_USE_* flags:
 *
 *   ANTHROPIC_API_KEY              → anthropic  (Anthropic SDK direct)
 *   OPENAI_API_KEY                 → openai     (OpenAI shim)
 *   OPENROUTER_API_KEY             → openrouter (OpenAI shim, OpenRouter base URL)
 *   GROK_API_KEY / XAI_API_KEY     → grok       (OpenAI shim, xAI base URL)
 *   GEMINI_API_KEY                → gemini    (OpenAI shim, Google base URL)
 *   OLLAMA_BASE_URL                → ollama     (OpenAI shim, local)
 *   (none set)                     → anthropic  (fallback)
 */

import Anthropic from '@anthropic-ai/sdk'

// ============================================================================
// Types
// ============================================================================

export type APIProvider =
  | 'anthropic'
  | 'openai'
  | 'openrouter'
  | 'grok'
  | 'gemini'
  | 'ollama'

export interface AnthropicClientConfig {
  /** API key. Falls back to ANTHROPIC_API_KEY env var. */
  apiKey?: string
  /** Pre-built headers (session ID, user-agent, custom headers, …). */
  defaultHeaders?: Record<string, string>
  /** Timeout in ms. Default: 600 000. */
  timeout?: number
  maxRetries?: number
  /** Custom fetch (e.g. test overrides). */
  fetch?: typeof globalThis.fetch
  /** Explicit provider override. When omitted, detectProvider() checks API keys. */
  provider?: APIProvider
  /** Base URL override. */
  baseURL?: string
}

// ============================================================================
// Provider detection – same logic as herm picking the first non-empty key
// ============================================================================

/**
 * Returns the active provider by checking which API key / URL is set.
 * Priority mirrors herm's config field order exactly:
 *   Anthropic → OpenAI → OpenRouter → Grok → Gemini → Ollama
 */
export function detectProvider(): APIProvider {
  if (process.env.ANTHROPIC_API_KEY) return 'anthropic'
  if (process.env.OPENROUTER_API_KEY) return 'openrouter'
  if (process.env.GROK_API_KEY || process.env.XAI_API_KEY) return 'grok'
  if (process.env.GEMINI_API_KEY) return 'gemini'
  if (process.env.OPENAI_API_KEY) return 'openai'
  if (process.env.OLLAMA_BASE_URL) return 'ollama'
  return 'anthropic'
}

/**
 * Provider-scoped model environment override.
 * Avoids cross-provider leaks (e.g. OPENAI_MODEL overriding Grok sessions).
 */
export function resolveProviderModelEnvOverride(
  provider: APIProvider = detectProvider(),
  env: NodeJS.ProcessEnv = process.env,
): string | undefined {
  if (provider === 'gemini') return env.GEMINI_MODEL
  if (provider === 'grok') return env.GROK_MODEL || env.XAI_MODEL
  if (provider === 'openrouter') return env.OPENROUTER_MODEL || env.OPENAI_MODEL
  if (provider === 'openai' || provider === 'ollama') return env.OPENAI_MODEL
  return env.ANTHROPIC_MODEL
}

// ============================================================================
// Custom header parsing
// ============================================================================

/**
 * Parses GRAYCODE_CUSTOM_HEADERS into a key/value map.
 * Accepts newline-separated curl-style "Name: Value" entries.
 */
export function parseCustomHeaders(): Record<string, string> {
  const result: Record<string, string> = {}
  const raw = process.env.GRAYCODE_CUSTOM_HEADERS
  if (!raw) return result
  for (const line of raw.split(/\r?\n/)) {
    const trimmed = line.trim()
    if (!trimmed) continue
    const colon = trimmed.indexOf(':')
    if (colon === -1) continue
    const name = trimmed.slice(0, colon).trim()
    const value = trimmed.slice(colon + 1).trim()
    if (name) result[name] = value
  }
  return result
}

// ============================================================================
// Factory – creates an Anthropic instance from standard env vars
// ============================================================================

/**
 * Creates a configured Anthropic client.
 * Call detectProvider() first – if the result is not 'anthropic', route to
 * the OpenAI shim in hawk instead of calling this function.
 */
export async function createAnthropicClient(
  config: AnthropicClientConfig = {},
): Promise<Anthropic> {
  return new Anthropic({
    apiKey: config.apiKey ?? process.env.ANTHROPIC_API_KEY,
    defaultHeaders: config.defaultHeaders,
    maxRetries: config.maxRetries,
    timeout: config.timeout ?? 600_000,
    dangerouslyAllowBrowser: true,
    ...(config.fetch && { fetch: config.fetch }),
    ...(config.baseURL && { baseURL: config.baseURL }),
  })
}
