/**
 * Eyrie client factory mirrors the herm/langdag pattern.
 *
 * Provider is detected by which API key is set.
 */
import Anthropic from '@anthropic-ai/sdk';
import { type APIProvider } from '../config/providerProfiles.js';
export interface AnthropicClientConfig {
    apiKey?: string;
    defaultHeaders?: Record<string, string>;
    timeout?: number;
    maxRetries?: number;
    fetch?: typeof globalThis.fetch;
    provider?: APIProvider;
    baseURL?: string;
}
export declare function detectProvider(): APIProvider;
export declare function resolveProviderModelEnvOverride(provider?: APIProvider, env?: NodeJS.ProcessEnv): string | undefined;
export declare function parseCustomHeaders(): Record<string, string>;
export declare function createAnthropicClient(config?: AnthropicClientConfig): Promise<Anthropic>;
//# sourceMappingURL=factory.d.ts.map