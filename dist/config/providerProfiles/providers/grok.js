import { DEFAULT_GROK_OPENAI_BASE_URL } from '../../providers.js';
export const GROK_RUNTIME_PROFILE = {
    mode: 'grok',
    defaultBaseUrl: DEFAULT_GROK_OPENAI_BASE_URL,
    defaultModel: 'grok-2',
    detectionEnv: ['GROK_API_KEY', 'XAI_API_KEY'],
    modelEnv: ['GROK_MODEL', 'XAI_MODEL', 'OPENAI_MODEL'],
    baseUrlEnv: ['GROK_BASE_URL', 'XAI_BASE_URL', 'OPENAI_BASE_URL', 'OPENAI_API_BASE'],
    apiKeys: [
        { env: 'GROK_API_KEY', source: 'grok' },
        { env: 'XAI_API_KEY', source: 'xai' },
        { env: 'OPENAI_API_KEY', source: 'openai' },
    ],
};
