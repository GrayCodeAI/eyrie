export declare const DEFAULT_OPENAI_BASE_URL = "https://api.openai.com/v1";
export declare const DEFAULT_CODEX_BASE_URL = "https://chatgpt.com/backend-api/codex";
type ReasoningEffort = 'low' | 'medium' | 'high';
export type ProviderTransport = 'chat_completions' | 'codex_responses';
export type ResolvedProviderRequest = {
    transport: ProviderTransport;
    requestedModel: string;
    resolvedModel: string;
    baseUrl: string;
    reasoning?: {
        effort: ReasoningEffort;
    };
};
export type ResolvedCodexCredentials = {
    apiKey: string;
    accountId?: string;
    authPath?: string;
    source: 'env' | 'auth.json' | 'none';
};
export declare function isLocalProviderUrl(baseUrl: string | undefined): boolean;
export declare function isCodexBaseUrl(baseUrl: string | undefined): boolean;
export declare function resolveProviderRequest(options?: {
    model?: string;
    baseUrl?: string;
    fallbackModel?: string;
}): ResolvedProviderRequest;
export declare function resolveCodexAuthPath(env?: NodeJS.ProcessEnv): string;
export declare function parseChatgptAccountId(token: string | undefined): string | undefined;
export declare function resolveCodexApiCredentials(env?: NodeJS.ProcessEnv): ResolvedCodexCredentials;
export {};
//# sourceMappingURL=providers.d.ts.map