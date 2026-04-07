/**
 * Multi-Provider LLM SDKs
 *
 * Re-exports from all major LLM provider SDKs.
 * Support: Anthropic, OpenAI, Google Gemini, Groq
 */
export { default as Anthropic } from '@anthropic-ai/sdk';
export { APIError as AnthropicAPIError, APIConnectionError as AnthropicAPIConnectionError, APIConnectionTimeoutError as AnthropicAPIConnectionTimeoutError, APIUserAbortError as AnthropicAPIUserAbortError, } from '@anthropic-ai/sdk';
export interface Message {
    role: 'user' | 'assistant' | 'system';
    content: string;
}
export interface CompletionRequest {
    model: string;
    messages: Message[];
    max_tokens?: number;
    temperature?: number;
    top_p?: number;
    stream?: boolean;
    tools?: Tool[];
}
export interface CompletionResponse {
    content: string;
    usage?: {
        input_tokens: number;
        output_tokens: number;
    };
}
export interface Tool {
    type: 'function';
    function: {
        name: string;
        description: string;
        parameters: {
            type: 'object';
            properties?: Record<string, unknown>;
            required?: string[];
        };
    };
}
export interface StreamEvent {
    type: 'content' | 'tool_call' | 'tool_result' | 'done' | 'error';
    content?: string;
    tool_call?: {
        name: string;
        arguments: Record<string, unknown>;
    };
    error?: string;
}
export declare const PROVIDERS: {
    readonly ANTHROPIC: "anthropic";
    readonly OPENAI: "openai";
    readonly GEMINI: "gemini";
    readonly GROQ: "groq";
    readonly VERCEL: "vercel";
    readonly VERTEX: "vertex";
    readonly BEDROCK: "bedrock";
    readonly MISTRAL: "mistral";
    readonly OPENCODE: "opencode";
};
export type Provider = typeof PROVIDERS[keyof typeof PROVIDERS];
export interface ProviderConfig {
    name: Provider;
    client: unknown;
    defaultModel: string;
    supportsStreaming: boolean;
    supportsTools: boolean;
}
//# sourceMappingURL=index.d.ts.map