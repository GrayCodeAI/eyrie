export type { APIProvider, RuntimeProviderProfile } from './providerProfiles/types.js'
import type { APIProvider, RuntimeProviderProfile } from './providerProfiles/types.js'
import { ANTHROPIC_RUNTIME_PROFILE } from './providerProfiles/providers/anthropic.js'
import { CANOPYWAVE_RUNTIME_PROFILE } from './providerProfiles/providers/canopywave.js'
import { GEMINI_RUNTIME_PROFILE } from './providerProfiles/providers/gemini.js'
import { GROK_RUNTIME_PROFILE } from './providerProfiles/providers/grok.js'
import { OPENAI_RUNTIME_PROFILE } from './providerProfiles/providers/openai.js'
import { OPENROUTER_RUNTIME_PROFILE } from './providerProfiles/providers/openrouter.js'
import { OPENCODEGO_RUNTIME_PROFILE } from './providerProfiles/providers/opencodego.js'

export const API_PROVIDER_DETECTION_ORDER: APIProvider[] = [
  'anthropic',
  'openrouter',
  'grok',
  'gemini',
  'canopywave',
  'openai',
  'opencodego',
  'ollama',
]

export const PROVIDER_MODEL_ENV_KEYS: Record<APIProvider, string[]> = {
  anthropic: [...ANTHROPIC_RUNTIME_PROFILE.modelEnv],
  openai: [...OPENAI_RUNTIME_PROFILE.modelEnv],
  canopywave: [...CANOPYWAVE_RUNTIME_PROFILE.modelEnv],
  openrouter: [...OPENROUTER_RUNTIME_PROFILE.modelEnv],
  grok: [...GROK_RUNTIME_PROFILE.modelEnv],
  gemini: [...GEMINI_RUNTIME_PROFILE.modelEnv],
  ollama: ['OLLAMA_MODEL', 'OPENAI_MODEL'],
  opencodego: [...OPENCODEGO_RUNTIME_PROFILE.modelEnv],
}

export const OLLAMA_DEFAULT_BASE_URL = 'http://localhost:11434/v1'
export const OLLAMA_DEFAULT_MODEL = 'llama3.1:8b'

export const OPENCODEGO_DEFAULT_BASE_URL = 'https://opencode.ai/zen/go/v1'
export const OPENCODEGO_DEFAULT_MODEL = 'kimi-k2.5'

export const OPENAI_COMPATIBLE_RUNTIME_PROFILE_ORDER = [
  'openrouter',
  'grok',
  'gemini',
  'anthropic',
  'canopywave',
  'openai',
  'opencodego',
] as const

export type OpenAICompatibleRuntimeProfileKey =
  (typeof OPENAI_COMPATIBLE_RUNTIME_PROFILE_ORDER)[number]

export const OPENAI_COMPATIBLE_RUNTIME_PROFILES: Record<
  OpenAICompatibleRuntimeProfileKey,
  RuntimeProviderProfile
> = {
  anthropic: ANTHROPIC_RUNTIME_PROFILE,
  grok: GROK_RUNTIME_PROFILE,
  gemini: GEMINI_RUNTIME_PROFILE,
  canopywave: CANOPYWAVE_RUNTIME_PROFILE,
  openai: OPENAI_RUNTIME_PROFILE,
  openrouter: OPENROUTER_RUNTIME_PROFILE,
  opencodego: OPENCODEGO_RUNTIME_PROFILE,
}
