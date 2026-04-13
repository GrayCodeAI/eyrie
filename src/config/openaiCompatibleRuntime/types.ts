import type { ResolvedProviderRequest } from '../providers.js'

export type OpenAICompatibleRuntimeMode =
  | 'openai'
  | 'openrouter'
  | 'gemini'
  | 'grok'
  | 'anthropic'

export type OpenAICompatibleApiKeySource =
  | 'openai'
  | 'canopywave'
  | 'openrouter'
  | 'gemini'
  | 'grok'
  | 'xai'
  | 'anthropic'
  | 'none'

export type ResolvedOpenAICompatibleRuntime = {
  mode: OpenAICompatibleRuntimeMode
  request: ResolvedProviderRequest
  apiKey: string
  apiKeySource: OpenAICompatibleApiKeySource
}

export type OpenAICompatibleRuntimeProvider = {
  mode: OpenAICompatibleRuntimeMode
  enableEnv?: string
  defaultBaseUrl: string
  defaultModel: string
  detectionEnv: string[]
  modelEnv: string[]
  baseUrlEnv: string[]
  apiKeys: Array<{
    env: string
    source: OpenAICompatibleApiKeySource
  }>
}
