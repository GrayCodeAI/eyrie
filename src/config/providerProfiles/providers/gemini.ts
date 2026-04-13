import { DEFAULT_GEMINI_OPENAI_BASE_URL } from '../../providers.js'
import type { RuntimeProviderProfile } from '../types.js'

export const GEMINI_RUNTIME_PROFILE: RuntimeProviderProfile = {
  mode: 'gemini',
  defaultBaseUrl: DEFAULT_GEMINI_OPENAI_BASE_URL,
  defaultModel: 'gemini-2.0-flash',
  detectionEnv: ['GEMINI_API_KEY'],
  modelEnv: ['GEMINI_MODEL', 'OPENAI_MODEL'],
  baseUrlEnv: ['GEMINI_BASE_URL', 'OPENAI_BASE_URL', 'OPENAI_API_BASE'],
  apiKeys: [
    { env: 'GEMINI_API_KEY', source: 'gemini' },
    { env: 'OPENAI_API_KEY', source: 'openai' },
  ],
}
