/**
 * Provider environment configuration.
 *
 * Owns the user-facing provider config shape (stored in ~/.hawk/provider.json),
 * the env-var application logic per provider, and all supporting helpers.
 */
import { existsSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { homedir } from 'node:os';
import { dirname, join } from 'node:path';
import { DEFAULT_CANOPYWAVE_OPENAI_BASE_URL, DEFAULT_GEMINI_OPENAI_BASE_URL, DEFAULT_GROK_OPENAI_BASE_URL, DEFAULT_OPENAI_BASE_URL, DEFAULT_OPENROUTER_OPENAI_BASE_URL, OPENCODEGO_DEFAULT_BASE_URL, } from './providers.js';
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
function applyOpenCodeGoProviderEnv({ env, config, activeModel, overwrite, catalog }) {
    const apiKey = asNonEmptyString(config.opencodego_api_key);
    const baseUrl = asNonEmptyString(config.opencodego_base_url) ?? OPENCODEGO_DEFAULT_BASE_URL;
    const model = activeModel ?? getProviderDefaultModel('opencodego', catalog);
    applyOpenAICompatibleProvider(env, 'OPENCODEGO', apiKey, model, baseUrl, overwrite);
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
        case 'opencodego': return applyOpenCodeGoProviderEnv(context);
    }
}
// ---------------------------------------------------------------------------
// Provider detection and config file I/O
// ---------------------------------------------------------------------------
/** Provider detection priority order */
export const PROVIDER_DETECTION_ORDER = [
    'anthropic',
    'openrouter',
    'grok',
    'gemini',
    'canopywave',
    'openai',
    'opencodego',
    'ollama',
];
/**
 * Gets the default config directory path.
 * Uses HAWK_CONFIG_DIR env var if set, otherwise ~/.hawk
 */
export function getProviderConfigDir() {
    return (process.env.HAWK_CONFIG_DIR ?? join(homedir(), '.hawk')).normalize('NFC');
}
/**
 * Gets the full path to the provider config file.
 */
export function getProviderConfigPath() {
    return join(getProviderConfigDir(), 'provider.json');
}
/**
 * Loads the provider config from disk.
 * Returns null if the file doesn't exist or is invalid.
 */
export function loadProviderConfig(path = getProviderConfigPath()) {
    if (!existsSync(path))
        return null;
    try {
        const content = readFileSync(path, 'utf8');
        const parsed = JSON.parse(content);
        if (!parsed || typeof parsed !== 'object') {
            console.warn(`[eyrie] Invalid config format at ${path}: expected object, got ${typeof parsed}`);
            return null;
        }
        return parsed;
    }
    catch (error) {
        const message = error instanceof Error ? error.message : String(error);
        console.warn(`[eyrie] Failed to load config from ${path}: ${message}`);
        return null;
    }
}
/**
 * Saves the provider config to disk.
 */
export function saveProviderConfig(config, path = getProviderConfigPath()) {
    mkdirSync(dirname(path), { recursive: true });
    writeFileSync(path, `${JSON.stringify(config, null, 2)}\n`, 'utf8');
}
/**
 * Checks if a provider has valid configuration (API key or base URL for Ollama).
 */
export function isProviderConfigured(config, provider) {
    const keys = PROVIDER_CONFIG_KEYS[provider];
    // Ollama only needs base URL
    if (provider === 'ollama') {
        return !!asNonEmptyString(config[keys.baseUrl]);
    }
    // Check if any API key is configured
    return keys.apiKey.some(keyField => !!asNonEmptyString(config[keyField]));
}
/**
 * Determines the default provider from config.
 * First checks explicit active_provider, then falls back to detection order.
 */
export function defaultProviderFromConfig(config) {
    if (!config)
        return null;
    const explicitProvider = asNonEmptyString(config.active_provider);
    if (explicitProvider && isProviderConfigured(config, explicitProvider)) {
        return explicitProvider;
    }
    for (const provider of PROVIDER_DETECTION_ORDER) {
        if (isProviderConfigured(config, provider))
            return provider;
    }
    return null;
}
/**
 * Checks if any provider-specific model is configured.
 */
function hasProviderScopedModel(config) {
    return PROVIDER_DETECTION_ORDER.some(provider => !!getProviderModel(config, provider));
}
/**
 * Gets the active model for a specific provider from config.
 * Handles both provider-specific model keys and legacy active_model.
 */
export function getProviderActiveModel(config, provider) {
    const providerSpecificModel = getProviderModel(config, provider);
    if (providerSpecificModel)
        return providerSpecificModel;
    if (hasProviderScopedModel(config))
        return undefined;
    const legacyModel = asNonEmptyString(config.active_model);
    if (!legacyModel)
        return undefined;
    // Legacy compatibility: active_model historically represented the default
    // configured provider only, not all providers.
    return defaultProviderFromConfig(config) === provider ? legacyModel : undefined;
}
/**
 * Clears all provider-related environment variables.
 */
export function clearProviderRuntimeEnv(env) {
    const keys = [
        'ANTHROPIC_API_KEY',
        'ANTHROPIC_MODEL',
        'ANTHROPIC_BASE_URL',
        'ANTHROPIC_VERSION',
        'OPENAI_API_KEY',
        'OPENAI_MODEL',
        'OPENAI_BASE_URL',
        'OPENROUTER_API_KEY',
        'OPENROUTER_MODEL',
        'OPENROUTER_BASE_URL',
        'CANOPYWAVE_API_KEY',
        'CANOPYWAVE_MODEL',
        'CANOPYWAVE_BASE_URL',
        'GROK_API_KEY',
        'GROK_MODEL',
        'GROK_BASE_URL',
        'XAI_API_KEY',
        'XAI_MODEL',
        'XAI_BASE_URL',
        'GEMINI_API_KEY',
        'GEMINI_MODEL',
        'GEMINI_BASE_URL',
        'OLLAMA_BASE_URL',
        'OPENCODEGO_API_KEY',
        'OPENCODEGO_MODEL',
        'OPENCODEGO_BASE_URL',
    ];
    for (const key of keys) {
        delete env[key];
    }
}
/**
 * Applies the full provider configuration to environment variables.
 * This is the main entry point for configuring the runtime environment.
 *
 * @returns The detected provider, or null if no provider is configured
 */
export function applyProviderConfigToEnv(env = process.env, config = loadProviderConfig(), options) {
    if (!config) {
        return null;
    }
    const provider = defaultProviderFromConfig(config);
    if (!provider)
        return null;
    const overwrite = options?.overwrite === true;
    const catalog = options?.catalog;
    if (overwrite) {
        clearProviderRuntimeEnv(env);
    }
    const activeModel = getProviderActiveModel(config, provider);
    const explorationModel = asNonEmptyString(config.exploration_model);
    setEnvValue(env, 'GRAYCODE_SMALL_FAST_MODEL', explorationModel, overwrite);
    applyProviderEnv(provider, {
        env,
        config,
        activeModel,
        overwrite,
        catalog,
    });
    return provider;
}
