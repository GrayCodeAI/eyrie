import {
  OLLAMA_DEFAULT_BASE_URL,
  OLLAMA_DEFAULT_MODEL,
} from '../../providerProfiles.js'
import type { OpenAICompatibleRuntimeProvider } from '../types.js'

export function createOllamaRuntimeProvider(
  env: NodeJS.ProcessEnv,
  openaiRuntimeProvider: OpenAICompatibleRuntimeProvider,
): OpenAICompatibleRuntimeProvider {
  return {
    ...openaiRuntimeProvider,
    defaultBaseUrl: env.OLLAMA_BASE_URL || OLLAMA_DEFAULT_BASE_URL,
    defaultModel: OLLAMA_DEFAULT_MODEL,
    modelEnv: ['OLLAMA_MODEL'],
    baseUrlEnv: ['OLLAMA_BASE_URL'],
    apiKeys: [],
  }
}
