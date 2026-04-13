import { API_PROVIDER_DETECTION_ORDER, } from '../../config/providerProfiles.js';
import { PROVIDER_PRESENCE_CHECKS } from './providers/index.js';
export function detectProviderFromEnv(env = process.env) {
    for (const provider of API_PROVIDER_DETECTION_ORDER) {
        if (PROVIDER_PRESENCE_CHECKS[provider](env)) {
            return provider;
        }
    }
    return 'anthropic';
}
