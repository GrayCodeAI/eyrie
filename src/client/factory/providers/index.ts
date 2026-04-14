import type { APIProvider } from '../../../config/providerProfiles.js'
import { isAnthropicConfigured } from './anthropic.js'
import { isCanopyWaveConfigured } from './canopywave.js'
import { isGeminiConfigured } from './gemini.js'
import { isGrokConfigured } from './grok.js'
import { isOllamaConfigured } from './ollama.js'
import { isOpenAIConfigured } from './openai.js'
import { isOpenRouterConfigured } from './openrouter.js'
import { isOpenCodeGoConfigured } from './opencodego.js'

export const PROVIDER_PRESENCE_CHECKS: Record<
  APIProvider,
  (env: NodeJS.ProcessEnv) => boolean
> = {
  anthropic: isAnthropicConfigured,
  openrouter: isOpenRouterConfigured,
  grok: isGrokConfigured,
  gemini: isGeminiConfigured,
  canopywave: isCanopyWaveConfigured,
  openai: isOpenAIConfigured,
  ollama: isOllamaConfigured,
  opencodego: isOpenCodeGoConfigured,
}
