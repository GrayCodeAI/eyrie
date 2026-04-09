import { DEFAULT_ANTHROPIC_OPENAI_BASE_URL, DEFAULT_GEMINI_OPENAI_BASE_URL, DEFAULT_GROK_OPENAI_BASE_URL, DEFAULT_OPENAI_BASE_URL, DEFAULT_OPENROUTER_OPENAI_BASE_URL, resolveProviderRequest, } from './providers.js';
export const OPENAI_COMPATIBLE_RUNTIME_PROVIDERS = {
    anthropic: {
        mode: 'anthropic',
        enableEnv: undefined,
        defaultBaseUrl: DEFAULT_ANTHROPIC_OPENAI_BASE_URL,
        defaultModel: 'claude-3-5-sonnet-latest',
        modelEnv: ['ANTHROPIC_MODEL', 'OPENAI_MODEL'],
        baseUrlEnv: ['ANTHROPIC_BASE_URL', 'OPENAI_BASE_URL', 'OPENAI_API_BASE'],
        apiKeys: [
            { env: 'ANTHROPIC_API_KEY', source: 'anthropic' },
            { env: 'OPENAI_API_KEY', source: 'openai' },
        ],
    },
    grok: {
        mode: 'grok',
        enableEnv: undefined,
        defaultBaseUrl: DEFAULT_GROK_OPENAI_BASE_URL,
        defaultModel: 'grok-2',
        modelEnv: ['GROK_MODEL', 'XAI_MODEL', 'OPENAI_MODEL'],
        baseUrlEnv: [
            'GROK_BASE_URL',
            'XAI_BASE_URL',
            'OPENAI_BASE_URL',
            'OPENAI_API_BASE',
        ],
        apiKeys: [
            { env: 'GROK_API_KEY', source: 'grok' },
            { env: 'XAI_API_KEY', source: 'xai' },
            { env: 'OPENAI_API_KEY', source: 'openai' },
        ],
    },
    gemini: {
        mode: 'gemini',
        enableEnv: undefined,
        defaultBaseUrl: DEFAULT_GEMINI_OPENAI_BASE_URL,
        defaultModel: 'gemini-2.0-flash',
        modelEnv: ['GEMINI_MODEL', 'OPENAI_MODEL'],
        baseUrlEnv: ['GEMINI_BASE_URL', 'OPENAI_BASE_URL', 'OPENAI_API_BASE'],
        apiKeys: [
            { env: 'GEMINI_API_KEY', source: 'gemini' },
            { env: 'OPENAI_API_KEY', source: 'openai' },
        ],
    },
    openai: {
        mode: 'openai',
        enableEnv: undefined,
        defaultBaseUrl: DEFAULT_OPENAI_BASE_URL,
        defaultModel: 'gpt-4o',
        modelEnv: ['OPENAI_MODEL'],
        baseUrlEnv: ['OPENAI_BASE_URL', 'OPENAI_API_BASE'],
        apiKeys: [{ env: 'OPENAI_API_KEY', source: 'openai' }],
    },
    openrouter: {
        mode: 'openrouter',
        enableEnv: undefined,
        defaultBaseUrl: DEFAULT_OPENROUTER_OPENAI_BASE_URL,
        defaultModel: 'openai/gpt-4o-mini',
        modelEnv: ['OPENROUTER_MODEL', 'OPENAI_MODEL'],
        baseUrlEnv: ['OPENROUTER_BASE_URL', 'OPENAI_BASE_URL', 'OPENAI_API_BASE'],
        apiKeys: [
            { env: 'OPENROUTER_API_KEY', source: 'openrouter' },
            { env: 'OPENAI_API_KEY', source: 'openai' },
        ],
    },
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
function isEnvTruthy(value) {
    if (!value)
        return false;
    const normalized = value.trim().toLowerCase();
    return (normalized === '1' ||
        normalized === 'true' ||
        normalized === 'yes' ||
        normalized === 'on');
}
function resolveRuntimeProvider(env) {
    // Detect by provider-specific API key presence first.
    // OPENAI_API_KEY can be present as a compatibility mirror for other providers.
    if (env.OPENROUTER_API_KEY)
        return OPENAI_COMPATIBLE_RUNTIME_PROVIDERS.openrouter;
    if (env.GROK_API_KEY || env.XAI_API_KEY)
        return OPENAI_COMPATIBLE_RUNTIME_PROVIDERS.grok;
    if (env.GEMINI_API_KEY)
        return OPENAI_COMPATIBLE_RUNTIME_PROVIDERS.gemini;
    if (env.ANTHROPIC_API_KEY)
        return OPENAI_COMPATIBLE_RUNTIME_PROVIDERS.anthropic;
    if (env.OPENAI_API_KEY)
        return OPENAI_COMPATIBLE_RUNTIME_PROVIDERS.openai;
    // Ollama: no API key, just a base URL
    if (env.OLLAMA_BASE_URL) {
        return {
            ...OPENAI_COMPATIBLE_RUNTIME_PROVIDERS.openai,
            defaultBaseUrl: env.OLLAMA_BASE_URL || OLLAMA_BASE_URL,
            defaultModel: 'llama3.1',
            modelEnv: ['OLLAMA_MODEL'],
            baseUrlEnv: ['OLLAMA_BASE_URL'],
            apiKeys: [],
            isOllama: true,
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
