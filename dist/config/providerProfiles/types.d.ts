export type APIProvider = 'anthropic' | 'openai' | 'canopywave' | 'openrouter' | 'grok' | 'gemini' | 'ollama' | 'opencodego';
type RuntimeMode = 'anthropic' | 'openai' | 'openrouter' | 'grok' | 'gemini' | 'opencodego';
type ApiKeySource = 'openai' | 'openrouter' | 'gemini' | 'grok' | 'xai' | 'anthropic' | 'canopywave' | 'opencodego';
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