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
//# sourceMappingURL=providerEnv.d.ts.map