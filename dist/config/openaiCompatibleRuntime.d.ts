import { type ResolvedProviderRequest } from './providers.js';
export type OpenAICompatibleRuntimeMode = 'openai' | 'openrouter' | 'gemini' | 'grok' | 'anthropic';
export type OpenAICompatibleApiKeySource = 'openai' | 'openrouter' | 'gemini' | 'grok' | 'xai' | 'anthropic' | 'none';
export type ResolvedOpenAICompatibleRuntime = {
    mode: OpenAICompatibleRuntimeMode;
    request: ResolvedProviderRequest;
    apiKey: string;
    apiKeySource: OpenAICompatibleApiKeySource;
};
type RuntimeProviderMode = OpenAICompatibleRuntimeMode;
export type OpenAICompatibleRuntimeProvider = {
    mode: RuntimeProviderMode;
    enableEnv?: string;
    defaultBaseUrl: string;
    defaultModel: string;
    modelEnv: string[];
    baseUrlEnv: string[];
    apiKeys: Array<{
        env: string;
        source: OpenAICompatibleApiKeySource;
    }>;
};
export declare const OPENAI_COMPATIBLE_RUNTIME_PROVIDERS: Record<RuntimeProviderMode, OpenAICompatibleRuntimeProvider>;
/**
 * Returns true when an OpenAI-compatible provider is active.
 * Detected by API key presence – same logic as detectProvider() in graycodeClient.
 * Note: ANTHROPIC_API_KEY is NOT included here; that goes through the Anthropic SDK directly.
 */
export declare function isOpenAICompatibleRuntimeEnabled(env?: NodeJS.ProcessEnv): boolean;
export declare function resolveOpenAICompatibleRuntime(options?: {
    env?: NodeJS.ProcessEnv;
    model?: string;
    baseUrl?: string;
    fallbackModel?: string;
}): ResolvedOpenAICompatibleRuntime;
export {};
//# sourceMappingURL=openaiCompatibleRuntime.d.ts.map