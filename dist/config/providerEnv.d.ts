/**
 * Provider environment configuration.
 *
 * Owns the user-facing provider config shape (stored in ~/.hawk/provider.json),
 * the env-var application logic per provider, and all supporting helpers.
 */
import type { APIProvider } from './providerProfiles/types.js';
import type { ModelCatalog } from '../catalog/types.js';
export type ProviderConfig = {
    _version?: string;
    active_provider?: APIProvider;
    anthropic_api_key?: string;
    grok_api_key?: string;
    xai_api_key?: string;
    openai_api_key?: string;
    canopywave_api_key?: string;
    openrouter_api_key?: string;
    gemini_api_key?: string;
    ollama_base_url?: string;
    opencodego_api_key?: string;
    anthropic_base_url?: string;
    canopywave_base_url?: string;
    grok_base_url?: string;
    xai_base_url?: string;
    openai_base_url?: string;
    openrouter_base_url?: string;
    gemini_base_url?: string;
    opencodego_base_url?: string;
    anthropic_model?: string;
    openai_model?: string;
    canopywave_model?: string;
    grok_model?: string;
    xai_model?: string;
    openrouter_model?: string;
    gemini_model?: string;
    ollama_model?: string;
    opencodego_model?: string;
    active_model?: string;
    exploration_model?: string;
    anthropic_version?: string;
};
export type ProviderEnvApplyContext = {
    env: NodeJS.ProcessEnv;
    config: ProviderConfig;
    activeModel: string | undefined;
    overwrite: boolean;
    catalog?: ModelCatalog | null;
};
export declare const PROVIDER_CONFIG_KEYS: Record<APIProvider, {
    apiKey: Array<keyof ProviderConfig>;
    model: Array<keyof ProviderConfig>;
    baseUrl: keyof ProviderConfig;
}>;
export declare function asNonEmptyString(value: unknown): string | undefined;
export declare function normalizeOllamaOpenAIBaseUrl(baseUrl: string | undefined): string | undefined;
export declare function setEnvValue(env: NodeJS.ProcessEnv, key: string, value: string | undefined, overwrite: boolean): void;
export declare function applyOpenAICompatibleProvider(env: NodeJS.ProcessEnv, prefix: string, apiKey: string | undefined, model: string, baseUrl: string, overwrite: boolean): void;
export declare function getProviderModel(config: ProviderConfig, provider: APIProvider): string | undefined;
export declare function getProviderApiKey(config: ProviderConfig, provider: APIProvider): string | undefined;
export declare function getProviderModelKey(provider: APIProvider): keyof ProviderConfig;
export declare function getProviderBaseUrlKey(provider: APIProvider): keyof ProviderConfig;
export declare function validateApiKey(apiKey: string | undefined, providerName: string): string | null;
export declare function validateBaseUrl(baseUrl: string | undefined): string | null;
export declare function applyProviderEnv(provider: APIProvider, context: ProviderEnvApplyContext): void;
/** Provider detection priority order */
export declare const PROVIDER_DETECTION_ORDER: APIProvider[];
/**
 * Gets the default config directory path.
 * Uses HAWK_CONFIG_DIR env var if set, otherwise ~/.hawk
 */
export declare function getProviderConfigDir(): string;
/**
 * Gets the full path to the provider config file.
 */
export declare function getProviderConfigPath(): string;
/**
 * Loads the provider config from disk.
 * Returns null if the file doesn't exist or is invalid.
 */
export declare function loadProviderConfig(path?: string): ProviderConfig | null;
/**
 * Saves the provider config to disk.
 */
export declare function saveProviderConfig(config: ProviderConfig, path?: string): void;
/**
 * Checks if a provider has valid configuration (API key or base URL for Ollama).
 */
export declare function isProviderConfigured(config: ProviderConfig, provider: APIProvider): boolean;
/**
 * Determines the default provider from config.
 * First checks explicit active_provider, then falls back to detection order.
 */
export declare function defaultProviderFromConfig(config: ProviderConfig | null): APIProvider | null;
/**
 * Gets the active model for a specific provider from config.
 * Handles both provider-specific model keys and legacy active_model.
 */
export declare function getProviderActiveModel(config: ProviderConfig, provider: APIProvider): string | undefined;
/**
 * Clears all provider-related environment variables.
 */
export declare function clearProviderRuntimeEnv(env: NodeJS.ProcessEnv): void;
/**
 * Applies the full provider configuration to environment variables.
 * This is the main entry point for configuring the runtime environment.
 *
 * @returns The detected provider, or null if no provider is configured
 */
export declare function applyProviderConfigToEnv(env?: NodeJS.ProcessEnv, config?: ProviderConfig | null, options?: {
    overwrite?: boolean;
    skipValidation?: boolean;
    catalog?: ModelCatalog | null;
}): APIProvider | null;
//# sourceMappingURL=providerEnv.d.ts.map