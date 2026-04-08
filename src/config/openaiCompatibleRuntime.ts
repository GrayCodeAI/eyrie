import {
  DEFAULT_ANTHROPIC_OPENAI_BASE_URL,
  DEFAULT_CODEX_BASE_URL,
  DEFAULT_GEMINI_OPENAI_BASE_URL,
  DEFAULT_GROK_OPENAI_BASE_URL,
  DEFAULT_OPENAI_BASE_URL,
  resolveCodexApiCredentials,
  resolveProviderRequest,
  type ResolvedCodexCredentials,
  type ResolvedProviderRequest,
} from './providers.js'

export type OpenAICompatibleRuntimeMode =
  | 'openai'
  | 'gemini'
  | 'grok'
  | 'anthropic'
  | 'codex'

export type OpenAICompatibleApiKeySource =
  | 'openai'
  | 'gemini'
  | 'google'
  | 'grok'
  | 'xai'
  | 'anthropic'
  | 'codex_env'
  | 'codex_auth_json'
  | 'none'

export type ResolvedOpenAICompatibleRuntime = {
  mode: OpenAICompatibleRuntimeMode
  request: ResolvedProviderRequest
  apiKey: string
  apiKeySource: OpenAICompatibleApiKeySource
  codexCredentials?: ResolvedCodexCredentials
}

type RuntimeProviderMode = Exclude<OpenAICompatibleRuntimeMode, 'codex'>

export type OpenAICompatibleRuntimeProvider = {
  mode: RuntimeProviderMode
  enableEnv?: string
  defaultBaseUrl: string
  defaultModel: string
  modelEnv: string[]
  baseUrlEnv: string[]
  apiKeys: Array<{
    env: string
    source: OpenAICompatibleApiKeySource
  }>
}

export const OPENAI_COMPATIBLE_RUNTIME_PROVIDERS: Record<
  RuntimeProviderMode,
  OpenAICompatibleRuntimeProvider
> = {
  anthropic: {
    mode: 'anthropic',
    enableEnv: undefined,
    defaultBaseUrl: DEFAULT_ANTHROPIC_OPENAI_BASE_URL,
    defaultModel: 'claude-3-5-sonnet-latest',
    modelEnv: ['ANTHROPIC_MODEL', 'OPENAI_MODEL'],
    baseUrlEnv: ['ANTHROPIC_BASE_URL', 'OPENAI_BASE_URL', 'OPENAI_API_BASE'],
    apiKeys: [
      { env: 'ANTHROPIC_API_KEY', source: 'anthropic' },
      { env: 'OPENAI_API_KEY', source: 'openai' },
    ],
  },
  grok: {
    mode: 'grok',
    enableEnv: undefined,
    defaultBaseUrl: DEFAULT_GROK_OPENAI_BASE_URL,
    defaultModel: 'grok-2',
    modelEnv: ['GROK_MODEL', 'XAI_MODEL', 'OPENAI_MODEL'],
    baseUrlEnv: [
      'GROK_BASE_URL',
      'XAI_BASE_URL',
      'OPENAI_BASE_URL',
      'OPENAI_API_BASE',
    ],
    apiKeys: [
      { env: 'GROK_API_KEY', source: 'grok' },
      { env: 'XAI_API_KEY', source: 'xai' },
      { env: 'OPENAI_API_KEY', source: 'openai' },
    ],
  },
  gemini: {
    mode: 'gemini',
    enableEnv: undefined,
    defaultBaseUrl: DEFAULT_GEMINI_OPENAI_BASE_URL,
    defaultModel: 'gemini-2.0-flash',
    modelEnv: ['GEMINI_MODEL', 'OPENAI_MODEL'],
    baseUrlEnv: ['GEMINI_BASE_URL', 'OPENAI_BASE_URL', 'OPENAI_API_BASE'],
    apiKeys: [
      { env: 'GEMINI_API_KEY', source: 'gemini' },
      { env: 'GOOGLE_API_KEY', source: 'google' },
      { env: 'OPENAI_API_KEY', source: 'openai' },
    ],
  },
  openai: {
    mode: 'openai',
    enableEnv: undefined,
    defaultBaseUrl: DEFAULT_OPENAI_BASE_URL,
    defaultModel: 'gpt-4o',
    modelEnv: ['OPENAI_MODEL'],
    baseUrlEnv: ['OPENAI_BASE_URL', 'OPENAI_API_BASE'],
    apiKeys: [{ env: 'OPENAI_API_KEY', source: 'openai' }],
  },
}

// Ollama is OpenAI-compatible but key-less (just needs a base URL)
const OLLAMA_BASE_URL = 'http://localhost:11434/v1'

/**
 * Provider priority mirrors herm's config field order:
 * Grok → Gemini → OpenAI → Ollama
 * (Anthropic is handled separately via createAnthropicClient, not the shim)
 */
const EXPLICIT_PROVIDER_PRIORITY: RuntimeProviderMode[] = [
  'grok',
  'gemini',
  'anthropic',
]

function asTrimmedString(value: unknown): string | undefined {
  return typeof value === 'string' && value.trim() ? value.trim() : undefined
}

function firstEnvValue(
  env: NodeJS.ProcessEnv,
  keys: string[],
): string | undefined {
  for (const key of keys) {
    const value = asTrimmedString(env[key])
    if (value) return value
  }
  return undefined
}

function isEnvTruthy(value: string | undefined): boolean {
  if (!value) return false
  const normalized = value.trim().toLowerCase()
  return (
    normalized === '1' ||
    normalized === 'true' ||
    normalized === 'yes' ||
    normalized === 'on'
  )
}

function resolveRuntimeProvider(
  env: NodeJS.ProcessEnv,
): OpenAICompatibleRuntimeProvider & { isOllama?: boolean } {
  // Detect by API key presence, same as herm picks the first non-empty key
  if (env.OPENAI_API_KEY) return OPENAI_COMPATIBLE_RUNTIME_PROVIDERS.openai
  if (env.GROK_API_KEY || env.XAI_API_KEY) return OPENAI_COMPATIBLE_RUNTIME_PROVIDERS.grok
  if (env.GEMINI_API_KEY || env.GOOGLE_API_KEY) return OPENAI_COMPATIBLE_RUNTIME_PROVIDERS.gemini
  // Ollama: no API key, just a base URL
  if (env.OLLAMA_BASE_URL) {
    return {
      ...OPENAI_COMPATIBLE_RUNTIME_PROVIDERS.openai,
      defaultBaseUrl: env.OLLAMA_BASE_URL || OLLAMA_BASE_URL,
      defaultModel: 'llama3.1',
      modelEnv: ['OLLAMA_MODEL'],
      baseUrlEnv: ['OLLAMA_BASE_URL'],
      apiKeys: [],
      isOllama: true,
    }
  }
  return OPENAI_COMPATIBLE_RUNTIME_PROVIDERS.openai
}

function resolveProviderApiKey(
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

/**
 * Returns true when an OpenAI-compatible provider is active.
 * Detected by API key presence – same logic as detectProvider() in graycodeClient.
 * Note: ANTHROPIC_API_KEY is NOT included here; that goes through the Anthropic SDK directly.
 */
export function isOpenAICompatibleRuntimeEnabled(
  env: NodeJS.ProcessEnv = process.env,
): boolean {
  return !!(
    env.GROK_API_KEY ||
    env.XAI_API_KEY ||
    env.GEMINI_API_KEY ||
    env.GOOGLE_API_KEY ||
    env.OPENAI_API_KEY ||
    env.OLLAMA_BASE_URL
  )
}

export function resolveOpenAICompatibleRuntime(options?: {
  env?: NodeJS.ProcessEnv
  model?: string
  baseUrl?: string
  fallbackModel?: string
}): ResolvedOpenAICompatibleRuntime {
  const env = options?.env ?? process.env
  const provider = resolveRuntimeProvider(env)
  const runtimeModel =
    options?.model ?? firstEnvValue(env, provider.modelEnv)
  const runtimeBaseUrl =
    options?.baseUrl ?? firstEnvValue(env, provider.baseUrlEnv)

  const request = resolveProviderRequest({
    model: runtimeModel,
    baseUrl: runtimeBaseUrl ?? provider.defaultBaseUrl,
    fallbackModel:
      options?.fallbackModel ??
      firstEnvValue(env, provider.modelEnv) ??
      provider.defaultModel,
  })

  if (request.transport === 'codex_responses') {
    const credentials = resolveCodexApiCredentials(env)
    const source: OpenAICompatibleApiKeySource =
      credentials.source === 'env'
        ? 'codex_env'
        : credentials.source === 'auth.json'
          ? 'codex_auth_json'
          : 'none'

    return {
      mode: 'codex',
      request: {
        ...request,
        baseUrl: runtimeBaseUrl ?? DEFAULT_CODEX_BASE_URL,
      },
      apiKey: credentials.apiKey,
      apiKeySource: source,
      codexCredentials: credentials,
    }
  }

  return {
    mode: provider.mode,
    request,
    ...resolveProviderApiKey(provider, env),
  }
}
