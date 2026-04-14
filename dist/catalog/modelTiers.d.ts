/**
 * Model tier tables and resolution functions.
 *
 * Maps Hawk model tiers (opus/sonnet/haiku) to concrete model IDs per provider.
 * All resolution checks the runtime catalog first; static tables are compile-time
 * fallbacks only.
 */
import type { APIProvider } from '../config/providerProfiles/types.js';
import type { ModelCatalog } from './types.js';
export type ModelName = string;
export type ModelTier = 'opus' | 'sonnet' | 'haiku';
/**
 * Tier names as a const array — use this when you need to iterate or
 * check membership (e.g. for alias resolution in the CLI).
 */
export declare const MODEL_TIER_ALIASES: readonly ["sonnet", "haiku", "opus"];
export type ModelTierAlias = (typeof MODEL_TIER_ALIASES)[number];
export type ModelConfig = Record<APIProvider, ModelName>;
export declare const OPENAI_MODEL_DEFAULTS: {
    readonly opus: "gpt-4o";
    readonly sonnet: "gpt-4o-mini";
    readonly haiku: "gpt-4o-mini";
};
export declare const GEMINI_MODEL_DEFAULTS: {
    readonly opus: "gemini-2.5-pro-preview-03-25";
    readonly sonnet: "gemini-2.0-flash";
    readonly haiku: "gemini-2.0-flash-lite";
};
export declare const HAWK_3_7_SONNET_CONFIG: {
    readonly anthropic: "claude-3-7-sonnet-20250219";
    readonly openai: "gpt-4o-mini";
    readonly canopywave: "zai/glm-4.6";
    readonly openrouter: "openai/gpt-4o-mini";
    readonly grok: "grok-2";
    readonly gemini: "gemini-2.0-flash";
    readonly ollama: "llama3.1:8b";
    readonly opencodego: "opencode-go";
};
export declare const HAWK_3_5_V2_SONNET_CONFIG: {
    readonly anthropic: "claude-3-5-sonnet-20241022";
    readonly openai: "gpt-4o-mini";
    readonly canopywave: "zai/glm-4.6";
    readonly openrouter: "openai/gpt-4o-mini";
    readonly grok: "grok-2";
    readonly gemini: "gemini-2.0-flash";
    readonly ollama: "llama3.1:8b";
    readonly opencodego: "opencode-go";
};
export declare const HAWK_3_5_HAIKU_CONFIG: {
    readonly anthropic: "claude-3-5-haiku-20241022";
    readonly openai: "gpt-4o-mini";
    readonly canopywave: "zai/glm-4.6";
    readonly openrouter: "openai/gpt-4o-mini";
    readonly grok: "grok-2";
    readonly gemini: "gemini-2.0-flash-lite";
    readonly ollama: "llama3.2:3b";
    readonly opencodego: "opencode-go";
};
export declare const HAWK_HAIKU_4_5_CONFIG: {
    readonly anthropic: "claude-haiku-4-5-20251001";
    readonly openai: "gpt-4o-mini";
    readonly canopywave: "zai/glm-4.6";
    readonly openrouter: "openai/gpt-4o-mini";
    readonly grok: "grok-2";
    readonly gemini: "gemini-2.0-flash-lite";
    readonly ollama: "llama3.2:3b";
    readonly opencodego: "opencode-go";
};
export declare const HAWK_SONNET_4_CONFIG: {
    readonly anthropic: "claude-sonnet-4-20250514";
    readonly openai: "gpt-4o-mini";
    readonly canopywave: "zai/glm-4.6";
    readonly openrouter: "openai/gpt-4o-mini";
    readonly grok: "grok-2";
    readonly gemini: "gemini-2.0-flash";
    readonly ollama: "llama3.1:8b";
    readonly opencodego: "opencode-go";
};
export declare const HAWK_SONNET_4_5_CONFIG: {
    readonly anthropic: "claude-sonnet-4-5-20250929";
    readonly openai: "gpt-4o";
    readonly canopywave: "zai/glm-4.6";
    readonly openrouter: "openai/gpt-4o";
    readonly grok: "grok-2";
    readonly gemini: "gemini-2.0-flash";
    readonly ollama: "llama3.1:70b";
    readonly opencodego: "opencode-go";
};
export declare const HAWK_SONNET_4_6_CONFIG: {
    readonly anthropic: "claude-sonnet-4-6";
    readonly openai: "gpt-4o";
    readonly canopywave: "zai/glm-4.6";
    readonly openrouter: "openai/gpt-4o";
    readonly grok: "grok-2";
    readonly gemini: "gemini-2.0-flash";
    readonly ollama: "llama3.1:70b";
    readonly opencodego: "opencode-go";
};
export declare const HAWK_OPUS_4_CONFIG: {
    readonly anthropic: "claude-opus-4-20250514";
    readonly openai: "gpt-4o";
    readonly canopywave: "zai/glm-4.6";
    readonly openrouter: "openai/gpt-4o";
    readonly grok: "grok-2";
    readonly gemini: "gemini-2.5-pro-preview-03-25";
    readonly ollama: "llama3.1:70b";
    readonly opencodego: "opencode-go";
};
export declare const HAWK_OPUS_4_1_CONFIG: {
    readonly anthropic: "claude-opus-4-1-20250805";
    readonly openai: "gpt-4o";
    readonly canopywave: "zai/glm-4.6";
    readonly openrouter: "openai/gpt-4o";
    readonly grok: "grok-2";
    readonly gemini: "gemini-2.5-pro-preview-03-25";
    readonly ollama: "llama3.1:70b";
    readonly opencodego: "opencode-go";
};
export declare const HAWK_OPUS_4_5_CONFIG: {
    readonly anthropic: "claude-opus-4-5-20251101";
    readonly openai: "gpt-4o";
    readonly canopywave: "zai/glm-4.6";
    readonly openrouter: "openai/gpt-4o";
    readonly grok: "grok-2";
    readonly gemini: "gemini-2.5-pro-preview-03-25";
    readonly ollama: "llama3.1:70b";
    readonly opencodego: "opencode-go";
};
export declare const HAWK_OPUS_4_6_CONFIG: {
    readonly anthropic: "claude-opus-4-6";
    readonly openai: "gpt-4o";
    readonly canopywave: "zai/glm-4.6";
    readonly openrouter: "openai/gpt-4o";
    readonly grok: "grok-2";
    readonly gemini: "gemini-2.5-pro-preview-03-25";
    readonly ollama: "llama3.1:70b";
    readonly opencodego: "opencode-go";
};
export declare const ALL_MODEL_CONFIGS: {
    readonly haiku35: {
        readonly anthropic: "claude-3-5-haiku-20241022";
        readonly openai: "gpt-4o-mini";
        readonly canopywave: "zai/glm-4.6";
        readonly openrouter: "openai/gpt-4o-mini";
        readonly grok: "grok-2";
        readonly gemini: "gemini-2.0-flash-lite";
        readonly ollama: "llama3.2:3b";
        readonly opencodego: "opencode-go";
    };
    readonly haiku45: {
        readonly anthropic: "claude-haiku-4-5-20251001";
        readonly openai: "gpt-4o-mini";
        readonly canopywave: "zai/glm-4.6";
        readonly openrouter: "openai/gpt-4o-mini";
        readonly grok: "grok-2";
        readonly gemini: "gemini-2.0-flash-lite";
        readonly ollama: "llama3.2:3b";
        readonly opencodego: "opencode-go";
    };
    readonly sonnet35: {
        readonly anthropic: "claude-3-5-sonnet-20241022";
        readonly openai: "gpt-4o-mini";
        readonly canopywave: "zai/glm-4.6";
        readonly openrouter: "openai/gpt-4o-mini";
        readonly grok: "grok-2";
        readonly gemini: "gemini-2.0-flash";
        readonly ollama: "llama3.1:8b";
        readonly opencodego: "opencode-go";
    };
    readonly sonnet37: {
        readonly anthropic: "claude-3-7-sonnet-20250219";
        readonly openai: "gpt-4o-mini";
        readonly canopywave: "zai/glm-4.6";
        readonly openrouter: "openai/gpt-4o-mini";
        readonly grok: "grok-2";
        readonly gemini: "gemini-2.0-flash";
        readonly ollama: "llama3.1:8b";
        readonly opencodego: "opencode-go";
    };
    readonly sonnet40: {
        readonly anthropic: "claude-sonnet-4-20250514";
        readonly openai: "gpt-4o-mini";
        readonly canopywave: "zai/glm-4.6";
        readonly openrouter: "openai/gpt-4o-mini";
        readonly grok: "grok-2";
        readonly gemini: "gemini-2.0-flash";
        readonly ollama: "llama3.1:8b";
        readonly opencodego: "opencode-go";
    };
    readonly sonnet45: {
        readonly anthropic: "claude-sonnet-4-5-20250929";
        readonly openai: "gpt-4o";
        readonly canopywave: "zai/glm-4.6";
        readonly openrouter: "openai/gpt-4o";
        readonly grok: "grok-2";
        readonly gemini: "gemini-2.0-flash";
        readonly ollama: "llama3.1:70b";
        readonly opencodego: "opencode-go";
    };
    readonly sonnet46: {
        readonly anthropic: "claude-sonnet-4-6";
        readonly openai: "gpt-4o";
        readonly canopywave: "zai/glm-4.6";
        readonly openrouter: "openai/gpt-4o";
        readonly grok: "grok-2";
        readonly gemini: "gemini-2.0-flash";
        readonly ollama: "llama3.1:70b";
        readonly opencodego: "opencode-go";
    };
    readonly opus40: {
        readonly anthropic: "claude-opus-4-20250514";
        readonly openai: "gpt-4o";
        readonly canopywave: "zai/glm-4.6";
        readonly openrouter: "openai/gpt-4o";
        readonly grok: "grok-2";
        readonly gemini: "gemini-2.5-pro-preview-03-25";
        readonly ollama: "llama3.1:70b";
        readonly opencodego: "opencode-go";
    };
    readonly opus41: {
        readonly anthropic: "claude-opus-4-1-20250805";
        readonly openai: "gpt-4o";
        readonly canopywave: "zai/glm-4.6";
        readonly openrouter: "openai/gpt-4o";
        readonly grok: "grok-2";
        readonly gemini: "gemini-2.5-pro-preview-03-25";
        readonly ollama: "llama3.1:70b";
        readonly opencodego: "opencode-go";
    };
    readonly opus45: {
        readonly anthropic: "claude-opus-4-5-20251101";
        readonly openai: "gpt-4o";
        readonly canopywave: "zai/glm-4.6";
        readonly openrouter: "openai/gpt-4o";
        readonly grok: "grok-2";
        readonly gemini: "gemini-2.5-pro-preview-03-25";
        readonly ollama: "llama3.1:70b";
        readonly opencodego: "opencode-go";
    };
    readonly opus46: {
        readonly anthropic: "claude-opus-4-6";
        readonly openai: "gpt-4o";
        readonly canopywave: "zai/glm-4.6";
        readonly openrouter: "openai/gpt-4o";
        readonly grok: "grok-2";
        readonly gemini: "gemini-2.5-pro-preview-03-25";
        readonly ollama: "llama3.1:70b";
        readonly opencodego: "opencode-go";
    };
};
export type ModelKey = keyof typeof ALL_MODEL_CONFIGS;
/** Union of all canonical Anthropic model IDs */
export type CanonicalModelId = (typeof ALL_MODEL_CONFIGS)[ModelKey]['anthropic'];
export declare const CANONICAL_MODEL_IDS: [CanonicalModelId, ...CanonicalModelId[]];
export declare const CANONICAL_ID_TO_KEY: Record<CanonicalModelId, ModelKey>;
/**
 * Ordered candidate model IDs for a provider/tier pair.
 * Preferred launch target first, then older fallbacks.
 */
export declare function getProviderModelCandidates(provider: APIProvider, tier: ModelTier): ModelName[];
/**
 * Preferred default model for a provider/tier pair.
 * Checks runtime catalog first, falls back to static config table.
 * Pass an already-loaded catalog to avoid re-reading from disk.
 */
export declare function getPreferredProviderModel(provider: APIProvider, tier: ModelTier, catalog?: ModelCatalog | null): ModelName;
/**
 * Default model for a provider — the first catalog entry, falling back to
 * the static config table.
 * Pass an already-loaded catalog to avoid re-reading from disk.
 */
export declare function getProviderDefaultModel(provider: APIProvider, catalog?: ModelCatalog | null): ModelName;
//# sourceMappingURL=modelTiers.d.ts.map