import { DEFAULT_OPENAI_BASE_URL } from '../../providers.js';
export const OPENAI_RUNTIME_PROFILE = {
    mode: 'openai',
    defaultBaseUrl: DEFAULT_OPENAI_BASE_URL,
    defaultModel: 'gpt-4o',
    detectionEnv: ['OPENAI_API_KEY'],
    modelEnv: ['OPENAI_MODEL'],
    baseUrlEnv: ['OPENAI_BASE_URL', 'OPENAI_API_BASE'],
    apiKeys: [{ env: 'OPENAI_API_KEY', source: 'openai' }],
};
