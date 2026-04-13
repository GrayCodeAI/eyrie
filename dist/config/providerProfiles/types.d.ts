export type APIProvider = 'anthropic' | 'openai' | 'canopywave' | 'openrouter' | 'grok' | 'gemini' | 'ollama';
type RuntimeMode = 'anthropic' | 'openai' | 'openrouter' | 'grok' | 'gemini';
type ApiKeySource = 'openai' | 'openrouter' | 'gemini' | 'grok' | 'xai' | 'anthropic' | 'canopywave';
export type RuntimeProviderProfile = {
    mode: RuntimeMode;
    defaultBaseUrl: string;
    defaultModel: string;
    detectionEnv: string[];
    modelEnv: string[];
    baseUrlEnv: string[];
    apiKeys: Array<{
        env: string;
        source: ApiKeySource;
    }>;
};
export {};
//# sourceMappingURL=types.d.ts.map