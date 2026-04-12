import {
  DEFAULT_ANTHROPIC_OPENAI_BASE_URL,
  DEFAULT_GEMINI_OPENAI_BASE_URL,
  DEFAULT_GROK_OPENAI_BASE_URL,
  DEFAULT_OPENAI_BASE_URL,
  DEFAULT_OPENROUTER_OPENAI_BASE_URL,
  DEFAULT_CANOPYWAVE_OPENAI_BASE_URL,
} from './providers.js'

export type APIProvider =
  | 'anthropic'
  | 'openai'
  | 'canopywave'
  | 'openrouter'
  | 'grok'
  | 'gemini'
  | 'ollama'

export const API_PROVIDER_DETECTION_ORDER: APIProvider[] = [
  'anthropic',
  'openrouter',
  'grok',
  'gemini',
  'canopywave',
  'openai',
  'ollama',
]

export const PROVIDER_MODEL_ENV_KEYS: Record<APIProvider, string[]> = {
  anthropic: ['ANTHROPIC_MODEL', 'OPENAI_MODEL'],
  openai: ['OPENAI_MODEL'],
  canopywave: ['CANOPYWAVE_MODEL', 'OPENAI_MODEL'],
  openrouter: ['OPENROUTER_MODEL', 'OPENAI_MODEL'],
  grok: ['GROK_MODEL', 'XAI_MODEL', 'OPENAI_MODEL'],
  gemini: ['GEMINI_MODEL', 'OPENAI_MODEL'],
  ollama: ['OLLAMA_MODEL', 'OPENAI_MODEL'],
}

export const OLLAMA_DEFAULT_BASE_URL = 'http://localhost:11434/v1'
export const OLLAMA_DEFAULT_MODEL = 'llama3.1:8b'

export type RuntimeProviderProfile = {
  mode: 'anthropic' | 'openai' | 'openrouter' | 'grok' | 'gemini'
  defaultBaseUrl: string
  defaultModel: string
  detectionEnv: string[]
  modelEnv: string[]
  baseUrlEnv: string[]
  apiKeys: Array<{
    env: string
    source: 'openai' | 'openrouter' | 'gemini' | 'grok' | 'xai' | 'anthropic' | 'canopywave'
  }>
}

export const OPENAI_COMPATIBLE_RUNTIME_PROFILE_ORDER = [
  'openrouter',
  'grok',
  'gemini',
  'anthropic',
  'canopywave',
  'openai',
] as const

export type OpenAICompatibleRuntimeProfileKey =
  (typeof OPENAI_COMPATIBLE_RUNTIME_PROFILE_ORDER)[number]

export const OPENAI_COMPATIBLE_RUNTIME_PROFILES: Record<
  OpenAICompatibleRuntimeProfileKey,
  RuntimeProviderProfile
> = {
  anthropic: {
    mode: 'anthropic',
    defaultBaseUrl: DEFAULT_ANTHROPIC_OPENAI_BASE_URL,
    defaultModel: 'claude-3-5-sonnet-latest',
    detectionEnv: ['ANTHROPIC_API_KEY'],
    modelEnv: PROVIDER_MODEL_ENV_KEYS.anthropic,
    baseUrlEnv: ['ANTHROPIC_BASE_URL', 'OPENAI_BASE_URL', 'OPENAI_API_BASE'],
    apiKeys: [
      { env: 'ANTHROPIC_API_KEY', source: 'anthropic' },
      { env: 'OPENAI_API_KEY', source: 'openai' },
    ],
  },
  grok: {
    mode: 'grok',
    defaultBaseUrl: DEFAULT_GROK_OPENAI_BASE_URL,
    defaultModel: 'grok-2',
    detectionEnv: ['GROK_API_KEY', 'XAI_API_KEY'],
    modelEnv: PROVIDER_MODEL_ENV_KEYS.grok,
    baseUrlEnv: ['GROK_BASE_URL', 'XAI_BASE_URL', 'OPENAI_BASE_URL', 'OPENAI_API_BASE'],
    apiKeys: [
      { env: 'GROK_API_KEY', source: 'grok' },
      { env: 'XAI_API_KEY', source: 'xai' },
      { env: 'OPENAI_API_KEY', source: 'openai' },
    ],
  },
  gemini: {
    mode: 'gemini',
    defaultBaseUrl: DEFAULT_GEMINI_OPENAI_BASE_URL,
    defaultModel: 'gemini-2.0-flash',
    detectionEnv: ['GEMINI_API_KEY'],
    modelEnv: PROVIDER_MODEL_ENV_KEYS.gemini,
    baseUrlEnv: ['GEMINI_BASE_URL', 'OPENAI_BASE_URL', 'OPENAI_API_BASE'],
    apiKeys: [
      { env: 'GEMINI_API_KEY', source: 'gemini' },
      { env: 'OPENAI_API_KEY', source: 'openai' },
    ],
  },
  canopywave: {
    mode: 'openai',
    defaultBaseUrl: DEFAULT_CANOPYWAVE_OPENAI_BASE_URL,
    defaultModel: 'zai/glm-4.6',
    detectionEnv: ['CANOPYWAVE_API_KEY'],
    modelEnv: PROVIDER_MODEL_ENV_KEYS.canopywave,
    baseUrlEnv: ['CANOPYWAVE_BASE_URL', 'OPENAI_BASE_URL', 'OPENAI_API_BASE'],
    apiKeys: [
      { env: 'CANOPYWAVE_API_KEY', source: 'canopywave' },
      { env: 'OPENAI_API_KEY', source: 'openai' },
    ],
  },
  openai: {
    mode: 'openai',
    defaultBaseUrl: DEFAULT_OPENAI_BASE_URL,
    defaultModel: 'gpt-4o',
    detectionEnv: ['OPENAI_API_KEY'],
    modelEnv: PROVIDER_MODEL_ENV_KEYS.openai,
    baseUrlEnv: ['OPENAI_BASE_URL', 'OPENAI_API_BASE'],
    apiKeys: [{ env: 'OPENAI_API_KEY', source: 'openai' }],
  },
  openrouter: {
    mode: 'openrouter',
    defaultBaseUrl: DEFAULT_OPENROUTER_OPENAI_BASE_URL,
    defaultModel: 'openai/gpt-4o-mini',
    detectionEnv: ['OPENROUTER_API_KEY'],
    modelEnv: PROVIDER_MODEL_ENV_KEYS.openrouter,
    baseUrlEnv: ['OPENROUTER_BASE_URL', 'OPENAI_BASE_URL', 'OPENAI_API_BASE'],
    apiKeys: [
      { env: 'OPENROUTER_API_KEY', source: 'openrouter' },
      { env: 'OPENAI_API_KEY', source: 'openai' },
    ],
  },
}
