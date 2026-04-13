import { asTrimmedString } from './utils.js';
export function resolveProviderApiKey(provider, env) {
    for (const key of provider.apiKeys) {
        const value = asTrimmedString(env[key.env]);
        if (value) {
            return {
                apiKey: value,
                apiKeySource: key.source,
            };
        }
    }
    return {
        apiKey: '',
        apiKeySource: 'none',
    };
}
