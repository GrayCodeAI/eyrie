/**
 * eyrie - Universal LLM Client
 *
 * Wraps standard SDKs (OpenAI, Anthropic, etc.)
 * Supports 54+ providers via baseUrl config
 * Simple API for hawk to use
 */
import { CORE_PROVIDERS, OPENAI_COMPATIBLE_PROVIDERS, type ProviderConfig, type ProviderType } from '../providers/registry.js';
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
     * OpenAI-compatible client (for all other providers)
     */
    private createOpenAICompatibleClient;
    /**
     * Google Gemini client
     */
    private createGoogleClient;
    /**
     * Mistral client
     */
    private createMistralClient;
    /**
     * Groq client
     */
    private createGroqClient;
    /**
     * Send a chat message
     */
    chat(messages: EyrieMessage[], options?: {
        provider?: string;
        model?: string;
        temperature?: number;
        maxTokens?: number;
    }): Promise<EyrieResponse>;
    /**
     * Stream chat response
     */
    stream(messages: EyrieMessage[], options?: {
        provider?: string;
        model?: string;
        temperature?: number;
        maxTokens?: number;
    }): AsyncGenerator<EyrieStreamEvent>;
    /**
     * List available providers
     */
    listProviders(): string[];
    /**
     * Get provider info
     */
    getProviderInfo(provider: string): ProviderConfig;
}
export declare function createEyrie(config?: EyrieConfig): EyrieClient;
export { CORE_PROVIDERS, OPENAI_COMPATIBLE_PROVIDERS, type ProviderConfig, type ProviderType };
//# sourceMappingURL=index.d.ts.map