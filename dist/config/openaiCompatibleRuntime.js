import { resolveProviderRequest, } from './providers.js';
import { OPENAI_COMPATIBLE_RUNTIME_PROVIDERS, resolveRuntimeProvider, } from './openaiCompatibleRuntime/providers/index.js';
import { resolveProviderApiKey } from './openaiCompatibleRuntime/apiKey.js';
import { firstEnvValue, } from './openaiCompatibleRuntime/utils.js';
import { isOpenAICompatibleRuntimeEnabled } from './openaiCompatibleRuntime/enabled.js';
export { OPENAI_COMPATIBLE_RUNTIME_PROVIDERS, isOpenAICompatibleRuntimeEnabled };
export function resolveOpenAICompatibleRuntime(options) {
    const env = options?.env ?? process.env;
    const provider = resolveRuntimeProvider(env);
    const runtimeModel = options?.model ?? firstEnvValue(env, provider.modelEnv);
    const runtimeBaseUrl = options?.baseUrl ?? firstEnvValue(env, provider.baseUrlEnv);
    const request = resolveProviderRequest({
        model: runtimeModel,
        baseUrl: runtimeBaseUrl ?? provider.defaultBaseUrl,
        fallbackModel: options?.fallbackModel ??
            firstEnvValue(env, provider.modelEnv) ??
            provider.defaultModel,
    });
    return {
        mode: provider.mode,
        request,
        ...resolveProviderApiKey(provider, env),
    };
}
