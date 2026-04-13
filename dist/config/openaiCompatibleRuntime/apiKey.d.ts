import type { OpenAICompatibleApiKeySource, OpenAICompatibleRuntimeProvider } from './types.js';
export declare function resolveProviderApiKey(provider: OpenAICompatibleRuntimeProvider, env: NodeJS.ProcessEnv): {
    apiKey: string;
    apiKeySource: OpenAICompatibleApiKeySource;
};
//# sourceMappingURL=apiKey.d.ts.map