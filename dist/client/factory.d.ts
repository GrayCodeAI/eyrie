/**
 * Eyrie client factory – mirrors the herm/langdag pattern exactly.
 *
 * Provider is detected by which API key is set (first match wins),
 * not by HAWK_CODE_USE_* flags:
 *
 *   ANTHROPIC_API_KEY              → anthropic  (Anthropic SDK direct)
 *   OPENROUTER_API_KEY             → openrouter (OpenAI shim, OpenRouter base URL)
 *   GROK_API_KEY / XAI_API_KEY     → grok       (OpenAI shim, xAI base URL)
 *   GEMINI_API_KEY                 → gemini     (OpenAI shim, Google base URL)
 *   CANOPYWAVE_API_KEY             → canopywave (OpenAI shim, CanopyWave base URL)
 *   OPENAI_API_KEY                 → openai     (OpenAI shim)
 *   OLLAMA_BASE_URL                → ollama     (OpenAI shim, local)
 *   (none set)                     → anthropic  (fallback)
 */
import Anthropic from '@anthropic-ai/sdk';
import { type APIProvider } from '../config/providerProfiles.js';
export interface AnthropicClientConfig {
    /** API key. Falls back to ANTHROPIC_API_KEY env var. */
    apiKey?: string;
    /** Pre-built headers (session ID, user-agent, custom headers, …). */
    defaultHeaders?: Record<string, string>;
    /** Timeout in ms. Default: 600 000. */
    timeout?: number;
    maxRetries?: number;
    /** Custom fetch (e.g. test overrides). */
    fetch?: typeof globalThis.fetch;
    /** Explicit provider override. When omitted, detectProvider() checks API keys. */
    provider?: APIProvider;
    /** Base URL override. */
    baseURL?: string;
}
/**
 * Returns the active provider by checking which API key / URL is set.
 * Priority favors provider-scoped keys ahead of generic OPENAI_API_KEY:
 *   Anthropic → OpenRouter → Grok → Gemini → CanopyWave → OpenAI → Ollama
 */
export declare function detectProvider(): APIProvider;
/**
 * Provider-scoped model environment override.
 * Avoids cross-provider leaks (e.g. OPENAI_MODEL overriding Grok sessions).
 */
export declare function resolveProviderModelEnvOverride(provider?: APIProvider, env?: NodeJS.ProcessEnv): string | undefined;
/**
 * Parses GRAYCODE_CUSTOM_HEADERS into a key/value map.
 * Accepts newline-separated curl-style "Name: Value" entries.
 */
export declare function parseCustomHeaders(): Record<string, string>;
/**
 * Creates a configured Anthropic client.
 * Call detectProvider() first – if the result is not 'anthropic', route to
 * the OpenAI shim in hawk instead of calling this function.
 */
export declare function createAnthropicClient(config?: AnthropicClientConfig): Promise<Anthropic>;
//# sourceMappingURL=factory.d.ts.map