import { type ResolvedProviderRequest } from './providers.js';
import { type OpenAICompatibleRuntimeProfileKey } from './providerProfiles.js';
export type OpenAICompatibleRuntimeMode = 'openai' | 'openrouter' | 'gemini' | 'grok' | 'anthropic';
export type OpenAICompatibleApiKeySource = 'openai' | 'canopywave' | 'openrouter' | 'gemini' | 'grok' | 'xai' | 'anthropic' | 'none';
export type ResolvedOpenAICompatibleRuntime = {
    mode: OpenAICompatibleRuntimeMode;
    request: ResolvedProviderRequest;
    apiKey: string;
    apiKeySource: OpenAICompatibleApiKeySource;
};
export type OpenAICompatibleRuntimeProvider = {
    mode: OpenAICompatibleRuntimeMode;
    enableEnv?: string;
    defaultBaseUrl: string;
    defaultModel: string;
    detectionEnv: string[];
    modelEnv: string[];
    baseUrlEnv: string[];
    apiKeys: Array<{
        env: string;
        source: OpenAICompatibleApiKeySource;
    }>;
};
export declare const OPENAI_COMPATIBLE_RUNTIME_PROVIDERS: Record<OpenAICompatibleRuntimeProfileKey, OpenAICompatibleRuntimeProvider>;
/**
 * Returns true when an OpenAI-compatible provider is active.
 * Detected by API key presence – same logic as detectProvider() in graycodeClient.
 * Includes provider-scoped key detection, including Anthropic compatibility mode.
 */
export declare function isOpenAICompatibleRuntimeEnabled(env?: NodeJS.ProcessEnv): boolean;
export declare function resolveOpenAICompatibleRuntime(options?: {
    env?: NodeJS.ProcessEnv;
    model?: string;
    baseUrl?: string;
    fallbackModel?: string;
}): ResolvedOpenAICompatibleRuntime;
//# sourceMappingURL=openaiCompatibleRuntime.d.ts.map