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
import { API_PROVIDER_DETECTION_ORDER, PROVIDER_MODEL_ENV_KEYS, } from '../config/providerProfiles.js';
// ============================================================================
// Provider detection – same logic as herm picking the first non-empty key
// ============================================================================
/**
 * Returns the active provider by checking which API key / URL is set.
 * Priority favors provider-scoped keys ahead of generic OPENAI_API_KEY:
 *   Anthropic → OpenRouter → Grok → Gemini → CanopyWave → OpenAI → Ollama
 */
export function detectProvider() {
    for (const provider of API_PROVIDER_DETECTION_ORDER) {
        if ((provider === 'anthropic' && process.env.ANTHROPIC_API_KEY) ||
            (provider === 'openrouter' && process.env.OPENROUTER_API_KEY) ||
            (provider === 'grok' && (process.env.GROK_API_KEY || process.env.XAI_API_KEY)) ||
            (provider === 'gemini' && process.env.GEMINI_API_KEY) ||
            (provider === 'canopywave' && process.env.CANOPYWAVE_API_KEY) ||
            (provider === 'openai' && process.env.OPENAI_API_KEY) ||
            (provider === 'ollama' && process.env.OLLAMA_BASE_URL)) {
            return provider;
        }
    }
    return 'anthropic';
}
/**
 * Provider-scoped model environment override.
 * Avoids cross-provider leaks (e.g. OPENAI_MODEL overriding Grok sessions).
 */
export function resolveProviderModelEnvOverride(provider = detectProvider(), env = process.env) {
    for (const key of PROVIDER_MODEL_ENV_KEYS[provider]) {
        const value = env[key];
        if (typeof value === 'string' && value.trim()) {
            return value.trim();
        }
    }
    return undefined;
}
// ============================================================================
// Custom header parsing
// ============================================================================
/**
 * Parses GRAYCODE_CUSTOM_HEADERS into a key/value map.
 * Accepts newline-separated curl-style "Name: Value" entries.
 */
export function parseCustomHeaders() {
    const result = {};
    const raw = process.env.GRAYCODE_CUSTOM_HEADERS;
    if (!raw)
        return result;
    for (const line of raw.split(/\r?\n/)) {
        const trimmed = line.trim();
        if (!trimmed)
            continue;
        const colon = trimmed.indexOf(':');
        if (colon === -1)
            continue;
        const name = trimmed.slice(0, colon).trim();
        const value = trimmed.slice(colon + 1).trim();
        if (name)
            result[name] = value;
    }
    return result;
}
// ============================================================================
// Factory – creates an Anthropic instance from standard env vars
// ============================================================================
/**
 * Creates a configured Anthropic client.
 * Call detectProvider() first – if the result is not 'anthropic', route to
 * the OpenAI shim in hawk instead of calling this function.
 */
export async function createAnthropicClient(config = {}) {
    return new Anthropic({
        apiKey: config.apiKey ?? process.env.ANTHROPIC_API_KEY,
        defaultHeaders: config.defaultHeaders,
        maxRetries: config.maxRetries,
        timeout: config.timeout ?? 600_000,
        dangerouslyAllowBrowser: true,
        ...(config.fetch && { fetch: config.fetch }),
        ...(config.baseURL && { baseURL: config.baseURL }),
    });
}
