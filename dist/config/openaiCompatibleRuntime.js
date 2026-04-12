import { resolveProviderRequest, } from './providers.js';
import { OPENAI_COMPATIBLE_RUNTIME_PROFILE_ORDER, OPENAI_COMPATIBLE_RUNTIME_PROFILES, } from './providerProfiles.js';
export const OPENAI_COMPATIBLE_RUNTIME_PROVIDERS = {
    anthropic: { ...OPENAI_COMPATIBLE_RUNTIME_PROFILES.anthropic },
    grok: { ...OPENAI_COMPATIBLE_RUNTIME_PROFILES.grok },
    gemini: { ...OPENAI_COMPATIBLE_RUNTIME_PROFILES.gemini },
    canopywave: { ...OPENAI_COMPATIBLE_RUNTIME_PROFILES.canopywave },
    openai: { ...OPENAI_COMPATIBLE_RUNTIME_PROFILES.openai },
    openrouter: { ...OPENAI_COMPATIBLE_RUNTIME_PROFILES.openrouter },
};
// Ollama is OpenAI-compatible but key-less (just needs a base URL)
const OLLAMA_BASE_URL = 'http://localhost:11434/v1';
function asTrimmedString(value) {
    return typeof value === 'string' && value.trim() ? value.trim() : undefined;
}
function firstEnvValue(env, keys) {
    for (const key of keys) {
        const value = asTrimmedString(env[key]);
        if (value)
            return value;
    }
    return undefined;
}
function resolveRuntimeProvider(env) {
    // Detect by provider-specific API key presence first.
    // OPENAI_API_KEY can be present as a compatibility mirror for other providers.
    for (const key of OPENAI_COMPATIBLE_RUNTIME_PROFILE_ORDER) {
        const provider = OPENAI_COMPATIBLE_RUNTIME_PROVIDERS[key];
        for (const envKey of provider.detectionEnv) {
            if (asTrimmedString(env[envKey])) {
                return provider;
            }
        }
    }
    // Ollama: no API key, just a base URL
    if (env.OLLAMA_BASE_URL) {
        return {
            ...OPENAI_COMPATIBLE_RUNTIME_PROVIDERS.openai,
            defaultBaseUrl: env.OLLAMA_BASE_URL || OLLAMA_BASE_URL,
            defaultModel: 'llama3.1',
            modelEnv: ['OLLAMA_MODEL'],
            baseUrlEnv: ['OLLAMA_BASE_URL'],
            apiKeys: [],
        };
    }
    return OPENAI_COMPATIBLE_RUNTIME_PROVIDERS.openai;
}
function resolveProviderApiKey(provider, env) {
    for (const key of provider.apiKeys) {
        const value = asTrimmedString(env[key.env]);
        if (value) {
            return {
                apiKey: value,
                apiKeySource: key.source,
            };
        }
    }
    return {
        apiKey: '',
        apiKeySource: 'none',
    };
}
/**
 * Returns true when an OpenAI-compatible provider is active.
 * Detected by API key presence – same logic as detectProvider() in graycodeClient.
 * Includes provider-scoped key detection, including Anthropic compatibility mode.
 */
export function isOpenAICompatibleRuntimeEnabled(env = process.env) {
    return !!(env.OPENROUTER_API_KEY ||
        env.GROK_API_KEY ||
        env.XAI_API_KEY ||
        env.GEMINI_API_KEY ||
        env.ANTHROPIC_API_KEY ||
        env.CANOPYWAVE_API_KEY ||
        env.OPENAI_API_KEY ||
        env.OLLAMA_BASE_URL);
}
export function resolveOpenAICompatibleRuntime(options) {
    const env = options?.env ?? process.env;
    const provider = resolveRuntimeProvider(env);
    const runtimeModel = options?.model ?? firstEnvValue(env, provider.modelEnv);
    const runtimeBaseUrl = options?.baseUrl ?? firstEnvValue(env, provider.baseUrlEnv);
    const request = resolveProviderRequest({
        model: runtimeModel,
        baseUrl: runtimeBaseUrl ?? provider.defaultBaseUrl,
        fallbackModel: options?.fallbackModel ??
            firstEnvValue(env, provider.modelEnv) ??
            provider.defaultModel,
    });
    return {
        mode: provider.mode,
        request,
        ...resolveProviderApiKey(provider, env),
    };
}
