import { DEFAULT_CANOPYWAVE_OPENAI_BASE_URL } from '../../providers.js'
import type { RuntimeProviderProfile } from '../types.js'

export const CANOPYWAVE_RUNTIME_PROFILE: RuntimeProviderProfile = {
  mode: 'openai',
  defaultBaseUrl: DEFAULT_CANOPYWAVE_OPENAI_BASE_URL,
  defaultModel: 'zai/glm-4.6',
  detectionEnv: ['CANOPYWAVE_API_KEY'],
  modelEnv: ['CANOPYWAVE_MODEL', 'OPENAI_MODEL'],
  baseUrlEnv: ['CANOPYWAVE_BASE_URL', 'OPENAI_BASE_URL', 'OPENAI_API_BASE'],
  apiKeys: [
    { env: 'CANOPYWAVE_API_KEY', source: 'canopywave' },
    { env: 'OPENAI_API_KEY', source: 'openai' },
  ],
}
