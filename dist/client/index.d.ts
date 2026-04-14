/**
 * eyrie - Universal LLM Client
 *
 * Wraps standard SDKs (OpenAI, Anthropic, etc.)
 * Supports 54+ providers via baseUrl config
 * Simple API for hawk to use
 */
import { type ProviderConfig } from '../providers/registry';
export interface EyrieConfig {
    provider?: string;
    apiKey?: string;
    baseUrl?: string;
    model?: string;
    maxRetries?: number;
}
export interface EyrieMessage {
    role: 'user' | 'assistant' | 'system';
    content: string;
    images?: string[];
    tools?: EyrieTool[];
}
export interface EyrieTool {
    name: string;
    description: string;
    parameters: Record<string, unknown>;
}
export interface EyrieUsage {
    promptTokens: number;
    completionTokens: number;
    totalTokens: number;
}
export interface EyrieResponse {
    content: string;
    usage?: EyrieUsage;
    toolCalls?: Array<{
        name: string;
        arguments: Record<string, unknown>;
    }>;
    finishReason: string;
}
export interface EyrieStreamEvent {
    type: 'content' | 'tool_call' | 'done' | 'error';
    content?: string;
    toolCall?: {
        name: string;
        arguments: string;
    };
    error?: string;
}
export declare class EyrieClient {
    private clients;
    private defaultProvider;
    private apiKeys;
    constructor(config?: EyrieConfig);
    /**
     * Set API key for a provider
     */
    setApiKey(provider: string, apiKey: string): void;
    /**
     * Get client for a provider
     */
    private getClient;
    /**
     * Get provider config
     */
    private getProviderConfig;
    private readonly clientFactories;
    /**
     * Resolve the default model for a provider from the eyrie runtime catalog.
     * Returns undefined for providers not tracked in the catalog (e.g. groq, ollama)
     * — callers must supply a model explicitly in that case.
     */
    private resolveDefaultModel;
    /**
     * Create provider client
     */
    private createClient;
    /**
     * Anthropic (Claude) client
     */
    private createAnthropicClient;
    /**
     * OpenAI client
     */
    private createOpenAIClient;
    /**
     * OpenAI-compatible client (for all other providers, including groq)
     */
    private createOpenAICompatibleClient;
    /**
     * Generate content with the specified provider
     */
    chat(messages: EyrieMessage[], options?: {
        provider?: string;
        model?: string;
        temperature?: number;
        maxTokens?: number;
        tools?: EyrieTool[];
        stream?: boolean;
    }): Promise<EyrieResponse>;
    /**
     * Stream content with the specified provider
     */
    streamChat(messages: EyrieMessage[], options?: {
        provider?: string;
        model?: string;
        temperature?: number;
        maxTokens?: number;
        tools?: EyrieTool[];
    }): AsyncGenerator<EyrieStreamEvent>;
    /**
     * List available providers
     */
    getProviders(): string[];
    /**
     * Get provider info
     */
    getProviderInfo(provider: string): ProviderConfig | undefined;
}
export declare function createEyrie(config?: EyrieConfig): EyrieClient;
export { CORE_PROVIDERS, OPENAI_COMPATIBLE_PROVIDERS, ProviderConfig, ProviderType } from '../providers/registry.js';
//# sourceMappingURL=index.d.ts.map