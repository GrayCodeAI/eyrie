/**
 * Unified Provider Registry
 *
 * Following the pi-mono pattern - one SDK, many providers via baseUrl
 */
import { OpenAI } from 'openai';
export type ProviderType = 'anthropic' | 'openai' | 'openai-compatible' | 'google' | 'mistral' | 'bedrock';
export interface ProviderConfig {
    name: string;
    type: ProviderType;
    baseUrl?: string;
    envKey: string;
    defaultModel: string;
    supportsStreaming: boolean;
    supportsTools: boolean;
    supportsReasoning: boolean;
    compat?: OpenAICompatConfig;
}
export interface OpenAICompatConfig {
    supportsStore?: boolean;
    supportsDeveloperRole?: boolean;
    supportsReasoningEffort?: boolean;
    supportsUsageInStreaming?: boolean;
    supportsStrictMode?: boolean;
    maxTokensField?: 'max_completion_tokens' | 'max_tokens';
    requiresToolResultName?: boolean;
    requiresAssistantAfterToolResult?: boolean;
    requiresThinkingAsText?: boolean;
    thinkingFormat?: 'openai' | 'zai' | 'qwen' | 'openrouter';
    reasoningEffortMap?: Partial<Record<string, string>>;
}
export declare const CORE_PROVIDERS: Record<string, ProviderConfig>;
export declare const OPENAI_COMPATIBLE_PROVIDERS: Record<string, ProviderConfig>;
export declare const ALL_PROVIDERS: Record<string, ProviderConfig>;
export declare const PROVIDER_LIST: string[];
export declare function getProvider(name: string): ProviderConfig | undefined;
export declare function getAllProviders(): ProviderConfig[];
export declare function getOpenAICompatibleProviders(): ProviderConfig[];
export declare function isOpenAICompatible(provider: string): boolean;
export declare function createOpenAIClient(provider: string, apiKey?: string, baseUrl?: string): OpenAI;
export declare function getEnvApiKey(provider: string): string | undefined;
export declare function checkApiKey(provider: string): boolean;
//# sourceMappingURL=registry.d.ts.map