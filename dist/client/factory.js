/**
 * Eyrie client factory mirrors the herm/langdag pattern.
 *
 * Provider is detected by which API key is set.
 */
import Anthropic from '@anthropic-ai/sdk';
import { PROVIDER_MODEL_ENV_KEYS, } from '../config/providerProfiles.js';
import { detectProviderFromEnv } from './factory/detection.js';
export function detectProvider() {
    return detectProviderFromEnv(process.env);
}
export function resolveProviderModelEnvOverride(provider = detectProvider(), env = process.env) {
    for (const key of PROVIDER_MODEL_ENV_KEYS[provider]) {
        const value = env[key];
        if (typeof value === 'string' && value.trim().length > 0) {
            return value.trim();
        }
    }
    return undefined;
}
export function parseCustomHeaders() {
    const result = {};
    const raw = process.env.GRAYCODE_CUSTOM_HEADERS;
    if (typeof raw === 'string' && raw.length > 0) {
        for (const line of raw.split(/\r?\n/)) {
            const trimmed = line.trim();
            if (trimmed.length === 0)
                continue;
            const colon = trimmed.indexOf(':');
            if (colon === -1)
                continue;
            const name = trimmed.slice(0, colon).trim();
            const value = trimmed.slice(colon + 1).trim();
            if (name.length > 0) {
                result[name] = value;
            }
        }
    }
    return result;
}
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
