export type APIProvider = 'anthropic' | 'openai' | 'canopywave' | 'openrouter' | 'grok' | 'gemini' | 'ollama';
export declare const API_PROVIDER_DETECTION_ORDER: APIProvider[];
export declare const PROVIDER_MODEL_ENV_KEYS: Record<APIProvider, string[]>;
export type RuntimeProviderProfile = {
    mode: 'anthropic' | 'openai' | 'openrouter' | 'grok' | 'gemini';
    defaultBaseUrl: string;
    defaultModel: string;
    detectionEnv: string[];
    modelEnv: string[];
    baseUrlEnv: string[];
    apiKeys: Array<{
        env: string;
        source: 'openai' | 'openrouter' | 'gemini' | 'grok' | 'xai' | 'anthropic' | 'canopywave';
    }>;
};
export declare const OPENAI_COMPATIBLE_RUNTIME_PROFILE_ORDER: readonly ["openrouter", "grok", "gemini", "anthropic", "canopywave", "openai"];
export type OpenAICompatibleRuntimeProfileKey = (typeof OPENAI_COMPATIBLE_RUNTIME_PROFILE_ORDER)[number];
export declare const OPENAI_COMPATIBLE_RUNTIME_PROFILES: Record<OpenAICompatibleRuntimeProfileKey, RuntimeProviderProfile>;
//# sourceMappingURL=providerProfiles.d.ts.map