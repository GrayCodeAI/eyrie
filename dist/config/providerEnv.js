/**
 * Provider environment configuration.
 *
 * Owns the user-facing provider config shape (stored in ~/.hawk/provider.json),
 * the env-var application logic per provider, and all supporting helpers.
 */
import { DEFAULT_CANOPYWAVE_OPENAI_BASE_URL, DEFAULT_GEMINI_OPENAI_BASE_URL, DEFAULT_GROK_OPENAI_BASE_URL, DEFAULT_OPENAI_BASE_URL, DEFAULT_OPENROUTER_OPENAI_BASE_URL, } from './providers.js';
import { OLLAMA_DEFAULT_BASE_URL, OLLAMA_DEFAULT_MODEL } from './providerProfiles.js';
import { getPreferredProviderModel, getProviderDefaultModel } from '../catalog/modelTiers.js';
// ---------------------------------------------------------------------------
// Provider config field mappings
// ---------------------------------------------------------------------------
export const PROVIDER_CONFIG_KEYS = {
    anthropic: {
        apiKey: ['anthropic_api_key'],
        model: ['anthropic_model'],
        baseUrl: 'anthropic_base_url',
    },
    openai: {
        apiKey: ['openai_api_key'],
        model: ['openai_model'],
        baseUrl: 'openai_base_url',
    },
    canopywave: {
        apiKey: ['canopywave_api_key'],
        model: ['canopywave_model'],
        baseUrl: 'canopywave_base_url',
    },
    openrouter: {
        apiKey: ['openrouter_api_key'],
        model: ['openrouter_model'],
        baseUrl: 'openrouter_base_url',
    },
    grok: {
        apiKey: ['grok_api_key', 'xai_api_key'],
        model: ['grok_model', 'xai_model'],
        baseUrl: 'grok_base_url',
    },
    gemini: {
        apiKey: ['gemini_api_key'],
        model: ['gemini_model'],
        baseUrl: 'gemini_base_url',
    },
    ollama: {
        apiKey: [],
        model: ['ollama_model'],
        baseUrl: 'ollama_base_url',
    },
    opencodego: {
        apiKey: ['opencodego_api_key'],
        model: ['opencodego_model'],
        baseUrl: 'opencodego_base_url',
    },
};
// ---------------------------------------------------------------------------
// Low-level helpers
// ---------------------------------------------------------------------------
export function asNonEmptyString(value) {
    return typeof value === 'string' && value.trim() ? value.trim() : undefined;
}
export function normalizeOllamaOpenAIBaseUrl(baseUrl) {
    if (!baseUrl)
        return undefined;
    const trimmed = baseUrl.replace(/\/+$/, '');
    return trimmed.endsWith('/v1') ? trimmed : `${trimmed}/v1`;
}
export function setEnvValue(env, key, value, overwrite) {
    if (!value)
        return;
    if (!overwrite && env[key])
        return;
    env[key] = value;
}
export function applyOpenAICompatibleProvider(env, prefix, apiKey, model, baseUrl, overwrite) {
    setEnvValue(env, `${prefix}_API_KEY`, apiKey, overwrite);
    setEnvValue(env, `${prefix}_MODEL`, model, overwrite);
    setEnvValue(env, `${prefix}_BASE_URL`, baseUrl, overwrite);
    setEnvValue(env, 'OPENAI_API_KEY', apiKey, overwrite);
    setEnvValue(env, 'OPENAI_MODEL', model, overwrite);
    setEnvValue(env, 'OPENAI_BASE_URL', baseUrl, overwrite);
}
export function getProviderModel(config, provider) {
    const modelKeys = PROVIDER_CONFIG_KEYS[provider].model;
    for (const modelKey of modelKeys) {
        const value = asNonEmptyString(config[modelKey]);
        if (value)
            return value;
    }
    return undefined;
}
export function getProviderApiKey(config, provider) {
    const keys = PROVIDER_CONFIG_KEYS[provider].apiKey;
    for (const keyField of keys) {
        const value = asNonEmptyString(config[keyField]);
        if (value)
            return value;
    }
    return undefined;
}
export function getProviderModelKey(provider) {
    return PROVIDER_CONFIG_KEYS[provider].model[0];
}
export function getProviderBaseUrlKey(provider) {
    return PROVIDER_CONFIG_KEYS[provider].baseUrl;
}
export function validateApiKey(apiKey, providerName) {
    if (!apiKey)
        return `${providerName} requires an API key`;
    if (apiKey === 'SUA_CHAVE')
        return `${providerName} API key cannot be placeholder value 'SUA_CHAVE'`;
    if (apiKey.length < 10)
        return `${providerName} API key appears invalid (too short)`;
    return null;
}
export function validateBaseUrl(baseUrl) {
    if (!baseUrl)
        return null;
    try {
        new URL(baseUrl);
        return null;
    }
    catch {
        return `Invalid base URL: ${baseUrl}`;
    }
}
// ---------------------------------------------------------------------------
// Per-provider env application
// ---------------------------------------------------------------------------
function applyAnthropicProviderEnv({ env, config, activeModel, overwrite, catalog }) {
    setEnvValue(env, 'ANTHROPIC_API_KEY', asNonEmptyString(config.anthropic_api_key), overwrite);
    setEnvValue(env, 'ANTHROPIC_MODEL', activeModel ?? getPreferredProviderModel('anthropic', 'sonnet', catalog), overwrite);
    setEnvValue(env, 'ANTHROPIC_BASE_URL', asNonEmptyString(config.anthropic_base_url), overwrite);
    setEnvValue(env, 'ANTHROPIC_VERSION', asNonEmptyString(config.anthropic_version), overwrite);
}
function applyOpenAIProviderEnv({ env, config, activeModel, overwrite, catalog }) {
    setEnvValue(env, 'OPENAI_API_KEY', asNonEmptyString(config.openai_api_key), overwrite);
    setEnvValue(env, 'OPENAI_MODEL', activeModel ?? getProviderDefaultModel('openai', catalog), overwrite);
    setEnvValue(env, 'OPENAI_BASE_URL', asNonEmptyString(config.openai_base_url) ?? DEFAULT_OPENAI_BASE_URL, overwrite);
}
function applyGeminiProviderEnv({ env, config, activeModel, overwrite, catalog }) {
    const apiKey = asNonEmptyString(config.gemini_api_key);
    const baseUrl = asNonEmptyString(config.gemini_base_url) ?? DEFAULT_GEMINI_OPENAI_BASE_URL;
    const model = activeModel ?? getProviderDefaultModel('gemini', catalog);
    applyOpenAICompatibleProvider(env, 'GEMINI', apiKey, model, baseUrl, overwrite);
}
function applyGrokProviderEnv({ env, config, activeModel, overwrite, catalog }) {
    const apiKey = asNonEmptyString(config.grok_api_key) ?? asNonEmptyString(config.xai_api_key);
    const baseUrl = asNonEmptyString(config.grok_base_url) ?? asNonEmptyString(config.xai_base_url) ?? DEFAULT_GROK_OPENAI_BASE_URL;
    const model = activeModel ?? getProviderDefaultModel('grok', catalog);
    setEnvValue(env, 'GROK_API_KEY', asNonEmptyString(config.grok_api_key), overwrite);
    setEnvValue(env, 'XAI_API_KEY', asNonEmptyString(config.xai_api_key), overwrite);
    applyOpenAICompatibleProvider(env, 'GROK', apiKey, model, baseUrl, overwrite);
}
function applyCanopyWaveProviderEnv({ env, config, activeModel, overwrite, catalog }) {
    const apiKey = asNonEmptyString(config.canopywave_api_key);
    const baseUrl = asNonEmptyString(config.canopywave_base_url) ?? DEFAULT_CANOPYWAVE_OPENAI_BASE_URL;
    const model = activeModel ?? getProviderDefaultModel('canopywave', catalog);
    applyOpenAICompatibleProvider(env, 'CANOPYWAVE', apiKey, model, baseUrl, overwrite);
}
function applyOpenRouterProviderEnv({ env, config, activeModel, overwrite, catalog }) {
    const apiKey = asNonEmptyString(config.openrouter_api_key);
    const baseUrl = asNonEmptyString(config.openrouter_base_url) ?? DEFAULT_OPENROUTER_OPENAI_BASE_URL;
    const model = activeModel ?? getProviderDefaultModel('openrouter', catalog);
    applyOpenAICompatibleProvider(env, 'OPENROUTER', apiKey, model, baseUrl, overwrite);
}
function applyOllamaProviderEnv({ env, config, activeModel, overwrite }) {
    setEnvValue(env, 'OPENAI_MODEL', activeModel ?? OLLAMA_DEFAULT_MODEL, overwrite);
    setEnvValue(env, 'OPENAI_BASE_URL', normalizeOllamaOpenAIBaseUrl(asNonEmptyString(config.ollama_base_url)) ?? OLLAMA_DEFAULT_BASE_URL, overwrite);
}
// ---------------------------------------------------------------------------
// Single entry point
// ---------------------------------------------------------------------------
export function applyProviderEnv(provider, context) {
    switch (provider) {
        case 'anthropic': return applyAnthropicProviderEnv(context);
        case 'openai': return applyOpenAIProviderEnv(context);
        case 'gemini': return applyGeminiProviderEnv(context);
        case 'grok': return applyGrokProviderEnv(context);
        case 'canopywave': return applyCanopyWaveProviderEnv(context);
        case 'openrouter': return applyOpenRouterProviderEnv(context);
        case 'ollama': return applyOllamaProviderEnv(context);
    }
}
