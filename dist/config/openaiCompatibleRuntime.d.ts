import { OPENAI_COMPATIBLE_RUNTIME_PROVIDERS } from './openaiCompatibleRuntime/providers/index.js';
import { isOpenAICompatibleRuntimeEnabled } from './openaiCompatibleRuntime/enabled.js';
export type { OpenAICompatibleRuntimeMode, OpenAICompatibleApiKeySource, ResolvedOpenAICompatibleRuntime, OpenAICompatibleRuntimeProvider, } from './openaiCompatibleRuntime/types.js';
export { OPENAI_COMPATIBLE_RUNTIME_PROVIDERS, isOpenAICompatibleRuntimeEnabled };
export declare function resolveOpenAICompatibleRuntime(options?: {
    env?: NodeJS.ProcessEnv;
    model?: string;
    baseUrl?: string;
    fallbackModel?: string;
}): import('./openaiCompatibleRuntime/types.js').ResolvedOpenAICompatibleRuntime;
//# sourceMappingURL=openaiCompatibleRuntime.d.ts.map