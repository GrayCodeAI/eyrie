export const OPENCODEGO_RUNTIME_PROFILE = {
    mode: 'opencodego',
    defaultBaseUrl: 'https://api.opencode.ai/v1',
    defaultModel: 'opencode-go',
    detectionEnv: ['OPENCODEGO_API_KEY'],
    modelEnv: ['OPENCODEGO_MODEL'],
    baseUrlEnv: ['OPENCODEGO_BASE_URL'],
    apiKeys: [
        { env: 'OPENCODEGO_API_KEY', source: 'opencodego' },
    ],
};
