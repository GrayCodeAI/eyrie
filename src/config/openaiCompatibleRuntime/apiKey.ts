import type {
  OpenAICompatibleApiKeySource,
  OpenAICompatibleRuntimeProvider,
} from './types.js'
import { asTrimmedString } from './utils.js'

export function resolveProviderApiKey(
  provider: OpenAICompatibleRuntimeProvider,
  env: NodeJS.ProcessEnv,
): {
  apiKey: string
  apiKeySource: OpenAICompatibleApiKeySource
} {
  for (const key of provider.apiKeys) {
    const value = asTrimmedString(env[key.env])
    if (value) {
      return {
        apiKey: value,
        apiKeySource: key.source,
      }
    }
  }

  return {
    apiKey: '',
    apiKeySource: 'none',
  }
}
