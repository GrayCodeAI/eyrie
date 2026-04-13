import { DEFAULT_ANTHROPIC_OPENAI_BASE_URL } from '../../providers.js';
export const ANTHROPIC_RUNTIME_PROFILE = {
    mode: 'anthropic',
    defaultBaseUrl: DEFAULT_ANTHROPIC_OPENAI_BASE_URL,
    defaultModel: 'claude-3-5-sonnet-latest',
    detectionEnv: ['ANTHROPIC_API_KEY'],
    modelEnv: ['ANTHROPIC_MODEL', 'OPENAI_MODEL'],
    baseUrlEnv: ['ANTHROPIC_BASE_URL', 'OPENAI_BASE_URL', 'OPENAI_API_BASE'],
    apiKeys: [
        { env: 'ANTHROPIC_API_KEY', source: 'anthropic' },
        { env: 'OPENAI_API_KEY', source: 'openai' },
    ],
};
