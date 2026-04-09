/**
 * Eyrie client factory – mirrors the herm/langdag pattern exactly.
 *
 * Provider is detected by which API key is set (first match wins),
 * not by HAWK_CODE_USE_* flags:
 *
 *   ANTHROPIC_API_KEY              → anthropic  (Anthropic SDK direct)
 *   OPENAI_API_KEY                 → openai     (OpenAI shim)
 *   GROK_API_KEY / XAI_API_KEY     → grok       (OpenAI shim, xAI base URL)
 *   GEMINI_API_KEY                → gemini    (OpenAI shim, Google base URL)
 *   OLLAMA_BASE_URL                → ollama     (OpenAI shim, local)
 *   (none set)                     → anthropic  (fallback)
 */
import Anthropic from '@anthropic-ai/sdk';
// ============================================================================
// Provider detection – same logic as herm picking the first non-empty key
// ============================================================================
/**
 * Returns the active provider by checking which API key / URL is set.
 * Priority mirrors herm's config field order exactly:
 *   Anthropic → OpenAI → Grok → Gemini → Ollama
 */
export function detectProvider() {
    if (process.env.ANTHROPIC_API_KEY)
        return 'anthropic';
    if (process.env.OPENAI_API_KEY)
        return 'openai';
    if (process.env.GROK_API_KEY || process.env.XAI_API_KEY)
        return 'grok';
    if (process.env.GEMINI_API_KEY)
        return 'gemini';
    if (process.env.OLLAMA_BASE_URL)
        return 'ollama';
    return 'anthropic';
}
/**
 * Provider-scoped model environment override.
 * Avoids cross-provider leaks (e.g. OPENAI_MODEL overriding Grok sessions).
 */
export function resolveProviderModelEnvOverride(provider = detectProvider(), env = process.env) {
    if (provider === 'gemini')
        return env.GEMINI_MODEL;
    if (provider === 'grok')
        return env.GROK_MODEL || env.XAI_MODEL;
    if (provider === 'openai' || provider === 'ollama')
        return env.OPENAI_MODEL;
    return env.ANTHROPIC_MODEL;
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
