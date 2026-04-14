export declare const DEFAULT_OPENAI_BASE_URL = "https://api.openai.com/v1";
export declare const DEFAULT_OPENROUTER_OPENAI_BASE_URL = "https://openrouter.ai/api/v1";
export declare const DEFAULT_CANOPYWAVE_OPENAI_BASE_URL = "https://inference.canopywave.io/v1";
export declare const DEFAULT_GEMINI_OPENAI_BASE_URL = "https://generativelanguage.googleapis.com/v1beta/openai";
export declare const DEFAULT_ANTHROPIC_OPENAI_BASE_URL = "https://api.anthropic.com/v1";
export declare const DEFAULT_GROK_OPENAI_BASE_URL = "https://api.x.ai/v1";
export declare const OPENCODEGO_DEFAULT_BASE_URL = "https://api.opencode.ai/v1";
type ReasoningEffort = 'low' | 'medium' | 'high';
export type ProviderTransport = 'chat_completions';
export type ResolvedProviderRequest = {
    transport: ProviderTransport;
    requestedModel: string;
    resolvedModel: string;
    baseUrl: string;
    reasoning?: {
        effort: ReasoningEffort;
    };
};
export declare function isLocalProviderUrl(baseUrl: string | undefined): boolean;
export declare function resolveProviderRequest(options?: {
    model?: string;
    baseUrl?: string;
    fallbackModel?: string;
}): ResolvedProviderRequest;
export {};
//# sourceMappingURL=providers.d.ts.map