import type { RuntimeProviderProfile } from '../types.js'

export const OPENCODEGO_RUNTIME_PROFILE: RuntimeProviderProfile = {
  mode: 'opencodego',
  defaultBaseUrl: 'https://opencode.ai/zen/go/v1',
  defaultModel: 'kimi-k2.5',
  detectionEnv: ['OPENCODEGO_API_KEY'],
  modelEnv: ['OPENCODEGO_MODEL'],
  baseUrlEnv: ['OPENCODEGO_BASE_URL'],
  apiKeys: [
    { env: 'OPENCODEGO_API_KEY', source: 'opencodego' },
  ],
}
