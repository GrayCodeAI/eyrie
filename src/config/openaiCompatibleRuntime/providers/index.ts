import {
  OPENAI_COMPATIBLE_RUNTIME_PROFILE_ORDER,
  type OpenAICompatibleRuntimeProfileKey,
} from '../../providerProfiles.js'
import type { OpenAICompatibleRuntimeProvider } from '../types.js'
import { asTrimmedString } from '../utils.js'
import { ANTHROPIC_RUNTIME_PROVIDER } from './anthropic.js'
import { CANOPYWAVE_RUNTIME_PROVIDER } from './canopywave.js'
import { GEMINI_RUNTIME_PROVIDER } from './gemini.js'
import { GROK_RUNTIME_PROVIDER } from './grok.js'
import { createOllamaRuntimeProvider } from './ollama.js'
import { OPENAI_RUNTIME_PROVIDER } from './openai.js'
import { OPENROUTER_RUNTIME_PROVIDER } from './openrouter.js'

export const OPENAI_COMPATIBLE_RUNTIME_PROVIDERS: Record<
  OpenAICompatibleRuntimeProfileKey,
  OpenAICompatibleRuntimeProvider
> = {
  anthropic: ANTHROPIC_RUNTIME_PROVIDER,
  grok: GROK_RUNTIME_PROVIDER,
  gemini: GEMINI_RUNTIME_PROVIDER,
  canopywave: CANOPYWAVE_RUNTIME_PROVIDER,
  openai: OPENAI_RUNTIME_PROVIDER,
  openrouter: OPENROUTER_RUNTIME_PROVIDER,
}

export function resolveRuntimeProvider(
  env: NodeJS.ProcessEnv,
): OpenAICompatibleRuntimeProvider {
  for (const key of OPENAI_COMPATIBLE_RUNTIME_PROFILE_ORDER) {
    const provider = OPENAI_COMPATIBLE_RUNTIME_PROVIDERS[key]
    for (const envKey of provider.detectionEnv) {
      if (asTrimmedString(env[envKey])) {
        return provider
      }
    }
  }

  if (env.OLLAMA_BASE_URL) {
    return createOllamaRuntimeProvider(
      env,
      OPENAI_COMPATIBLE_RUNTIME_PROVIDERS.openai,
    )
  }

  return OPENAI_COMPATIBLE_RUNTIME_PROVIDERS.openai
}
