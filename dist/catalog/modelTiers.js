/**
 * Model tier tables and resolution functions.
 *
 * Maps Hawk model tiers (opus/sonnet/haiku) to concrete model IDs per provider.
 * All resolution checks the runtime catalog first; static tables are compile-time
 * fallbacks only.
 */
import { loadModelCatalogSync, modelsForProvider } from './modelCatalog.js';
/**
 * Tier names as a const array — use this when you need to iterate or
 * check membership (e.g. for alias resolution in the CLI).
 */
export const MODEL_TIER_ALIASES = ['sonnet', 'haiku', 'opus'];
// ---------------------------------------------------------------------------
// OpenAI-compatible model mappings (tier → model)
// ---------------------------------------------------------------------------
export const OPENAI_MODEL_DEFAULTS = {
    opus: 'gpt-4o',
    sonnet: 'gpt-4o-mini',
    haiku: 'gpt-4o-mini',
};
export const GEMINI_MODEL_DEFAULTS = {
    opus: 'gemini-2.5-pro-preview-03-25',
    sonnet: 'gemini-2.0-flash',
    haiku: 'gemini-2.0-flash-lite',
};
// ---------------------------------------------------------------------------
// Version-specific model configs (tier × version → model ID per provider)
// ---------------------------------------------------------------------------
export const HAWK_3_7_SONNET_CONFIG = {
    anthropic: 'claude-3-7-sonnet-20250219',
    openai: 'gpt-4o-mini',
    canopywave: 'zai/glm-4.6',
    openrouter: 'openai/gpt-4o-mini',
    grok: 'grok-2',
    gemini: 'gemini-2.0-flash',
    ollama: 'llama3.1:8b',
    opencodego: 'kimi-k2.5',
};
export const HAWK_3_5_V2_SONNET_CONFIG = {
    anthropic: 'claude-3-5-sonnet-20241022',
    openai: 'gpt-4o-mini',
    canopywave: 'zai/glm-4.6',
    openrouter: 'openai/gpt-4o-mini',
    grok: 'grok-2',
    gemini: 'gemini-2.0-flash',
    ollama: 'llama3.1:8b',
    opencodego: 'kimi-k2.5',
};
export const HAWK_3_5_HAIKU_CONFIG = {
    anthropic: 'claude-3-5-haiku-20241022',
    openai: 'gpt-4o-mini',
    canopywave: 'zai/glm-4.6',
    openrouter: 'openai/gpt-4o-mini',
    grok: 'grok-2',
    gemini: 'gemini-2.0-flash-lite',
    ollama: 'llama3.2:3b',
    opencodego: 'kimi-k2.5',
};
export const HAWK_HAIKU_4_5_CONFIG = {
    anthropic: 'claude-haiku-4-5-20251001',
    openai: 'gpt-4o-mini',
    canopywave: 'zai/glm-4.6',
    openrouter: 'openai/gpt-4o-mini',
    grok: 'grok-2',
    gemini: 'gemini-2.0-flash-lite',
    ollama: 'llama3.2:3b',
    opencodego: 'kimi-k2.5',
};
export const HAWK_SONNET_4_CONFIG = {
    anthropic: 'claude-sonnet-4-20250514',
    openai: 'gpt-4o-mini',
    canopywave: 'zai/glm-4.6',
    openrouter: 'openai/gpt-4o-mini',
    grok: 'grok-2',
    gemini: 'gemini-2.0-flash',
    ollama: 'llama3.1:8b',
    opencodego: 'kimi-k2.5',
};
export const HAWK_SONNET_4_5_CONFIG = {
    anthropic: 'claude-sonnet-4-5-20250929',
    openai: 'gpt-4o',
    canopywave: 'zai/glm-4.6',
    openrouter: 'openai/gpt-4o',
    grok: 'grok-2',
    gemini: 'gemini-2.0-flash',
    ollama: 'llama3.1:70b',
    opencodego: 'kimi-k2.5',
};
export const HAWK_SONNET_4_6_CONFIG = {
    anthropic: 'claude-sonnet-4-6',
    openai: 'gpt-4o',
    canopywave: 'zai/glm-4.6',
    openrouter: 'openai/gpt-4o',
    grok: 'grok-2',
    gemini: 'gemini-2.0-flash',
    ollama: 'llama3.1:70b',
    opencodego: 'kimi-k2.5',
};
export const HAWK_OPUS_4_CONFIG = {
    anthropic: 'claude-opus-4-20250514',
    openai: 'gpt-4o',
    canopywave: 'zai/glm-4.6',
    openrouter: 'openai/gpt-4o',
    grok: 'grok-2',
    gemini: 'gemini-2.5-pro-preview-03-25',
    ollama: 'llama3.1:70b',
    opencodego: 'kimi-k2.5',
};
export const HAWK_OPUS_4_1_CONFIG = {
    anthropic: 'claude-opus-4-1-20250805',
    openai: 'gpt-4o',
    canopywave: 'zai/glm-4.6',
    openrouter: 'openai/gpt-4o',
    grok: 'grok-2',
    gemini: 'gemini-2.5-pro-preview-03-25',
    ollama: 'llama3.1:70b',
    opencodego: 'kimi-k2.5',
};
export const HAWK_OPUS_4_5_CONFIG = {
    anthropic: 'claude-opus-4-5-20251101',
    openai: 'gpt-4o',
    canopywave: 'zai/glm-4.6',
    openrouter: 'openai/gpt-4o',
    grok: 'grok-2',
    gemini: 'gemini-2.5-pro-preview-03-25',
    ollama: 'llama3.1:70b',
    opencodego: 'kimi-k2.5',
};
export const HAWK_OPUS_4_6_CONFIG = {
    anthropic: 'claude-opus-4-6',
    openai: 'gpt-4o',
    canopywave: 'zai/glm-4.6',
    openrouter: 'openai/gpt-4o',
    grok: 'grok-2',
    gemini: 'gemini-2.5-pro-preview-03-25',
    ollama: 'llama3.1:70b',
    opencodego: 'kimi-k2.5',
};
// @[MODEL LAUNCH]: Register the new config here.
export const ALL_MODEL_CONFIGS = {
    haiku35: HAWK_3_5_HAIKU_CONFIG,
    haiku45: HAWK_HAIKU_4_5_CONFIG,
    sonnet35: HAWK_3_5_V2_SONNET_CONFIG,
    sonnet37: HAWK_3_7_SONNET_CONFIG,
    sonnet40: HAWK_SONNET_4_CONFIG,
    sonnet45: HAWK_SONNET_4_5_CONFIG,
    sonnet46: HAWK_SONNET_4_6_CONFIG,
    opus40: HAWK_OPUS_4_CONFIG,
    opus41: HAWK_OPUS_4_1_CONFIG,
    opus45: HAWK_OPUS_4_5_CONFIG,
    opus46: HAWK_OPUS_4_6_CONFIG,
};
const MODEL_KEYS = Object.keys(ALL_MODEL_CONFIGS);
export const CANONICAL_MODEL_IDS = Object.values(ALL_MODEL_CONFIGS).map(c => c.anthropic);
export const CANONICAL_ID_TO_KEY = Object.fromEntries(Object.entries(ALL_MODEL_CONFIGS).map(([key, cfg]) => [cfg.anthropic, key]));
// ---------------------------------------------------------------------------
// Tier → model key resolution tables (private)
// ---------------------------------------------------------------------------
const PREFERRED_MODEL_KEYS_BY_PROVIDER = {
    anthropic: { opus: 'opus46', sonnet: 'sonnet46', haiku: 'haiku45' },
    openai: { opus: 'opus46', sonnet: 'sonnet46', haiku: 'haiku45' },
    canopywave: { opus: 'opus46', sonnet: 'sonnet46', haiku: 'haiku45' },
    openrouter: { opus: 'opus46', sonnet: 'sonnet46', haiku: 'haiku45' },
    grok: { opus: 'opus46', sonnet: 'sonnet46', haiku: 'haiku45' },
    gemini: { opus: 'opus46', sonnet: 'sonnet46', haiku: 'haiku45' },
    ollama: { opus: 'opus46', sonnet: 'sonnet46', haiku: 'haiku45' },
    opencodego: { opus: 'opus46', sonnet: 'sonnet46', haiku: 'haiku45' },
};
const MODEL_FALLBACK_KEYS_BY_TIER = {
    opus: ['opus46', 'opus45', 'opus41', 'opus40'],
    sonnet: ['sonnet46', 'sonnet45', 'sonnet40', 'sonnet37', 'sonnet35'],
    haiku: ['haiku45', 'haiku35'],
};
// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------
function getProviderModelPool(provider) {
    const seen = new Set();
    const ordered = [];
    for (const key of MODEL_KEYS) {
        const model = ALL_MODEL_CONFIGS[key][provider];
        if (!seen.has(model)) {
            seen.add(model);
            ordered.push(model);
        }
    }
    return ordered;
}
function catalogModelIds(catalog, provider) {
    return modelsForProvider(catalog, provider).map(m => m.id);
}
// ---------------------------------------------------------------------------
// Public resolution API
// ---------------------------------------------------------------------------
/**
 * Ordered candidate model IDs for a provider/tier pair.
 * Preferred launch target first, then older fallbacks.
 */
export function getProviderModelCandidates(provider, tier) {
    const seen = new Set();
    const ordered = [];
    const preferredKey = PREFERRED_MODEL_KEYS_BY_PROVIDER[provider][tier];
    const preferred = ALL_MODEL_CONFIGS[preferredKey][provider];
    if (preferred && !seen.has(preferred)) {
        seen.add(preferred);
        ordered.push(preferred);
    }
    for (const key of MODEL_FALLBACK_KEYS_BY_TIER[tier]) {
        const model = ALL_MODEL_CONFIGS[key][provider];
        if (model && !seen.has(model)) {
            seen.add(model);
            ordered.push(model);
        }
    }
    return ordered;
}
/**
 * Preferred default model for a provider/tier pair.
 * Checks runtime catalog first, falls back to static config table.
 * Pass an already-loaded catalog to avoid re-reading from disk.
 */
export function getPreferredProviderModel(provider, tier, catalog) {
    const resolvedCatalog = catalog ?? loadModelCatalogSync();
    if (provider !== 'anthropic') {
        return getProviderDefaultModel(provider, resolvedCatalog);
    }
    const ids = catalogModelIds(resolvedCatalog, provider);
    if (ids.length > 0) {
        // Prefer the first catalog model if it matches a known candidate
        const candidates = getProviderModelCandidates(provider, tier);
        for (const candidate of candidates) {
            if (ids.includes(candidate))
                return candidate;
        }
        return ids[0];
    }
    const candidates = getProviderModelCandidates(provider, tier);
    if (candidates.length > 0)
        return candidates[0];
    const pool = getProviderModelPool(provider);
    if (pool.length > 0)
        return pool[0];
    return ALL_MODEL_CONFIGS.sonnet46[provider];
}
/**
 * Default model for a provider — the first catalog entry, falling back to
 * the static config table.
 * Pass an already-loaded catalog to avoid re-reading from disk.
 */
export function getProviderDefaultModel(provider, catalog) {
    const resolvedCatalog = catalog ?? loadModelCatalogSync();
    const ids = catalogModelIds(resolvedCatalog, provider);
    if (ids.length > 0)
        return ids[0];
    const pool = getProviderModelPool(provider);
    if (pool.length > 0)
        return pool[0];
    return ALL_MODEL_CONFIGS.sonnet46[provider];
}
