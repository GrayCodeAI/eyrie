import { DEFAULT_OPENROUTER_OPENAI_BASE_URL } from '../../providers.js'
import type { RuntimeProviderProfile } from '../types.js'

export const OPENROUTER_RUNTIME_PROFILE: RuntimeProviderProfile = {
  mode: 'openrouter',
  defaultBaseUrl: DEFAULT_OPENROUTER_OPENAI_BASE_URL,
  defaultModel: 'openai/gpt-4o-mini',
  detectionEnv: ['OPENROUTER_API_KEY'],
  modelEnv: ['OPENROUTER_MODEL', 'OPENAI_MODEL'],
  baseUrlEnv: ['OPENROUTER_BASE_URL', 'OPENAI_BASE_URL', 'OPENAI_API_BASE'],
  apiKeys: [
    { env: 'OPENROUTER_API_KEY', source: 'openrouter' },
    { env: 'OPENAI_API_KEY', source: 'openai' },
  ],
}
