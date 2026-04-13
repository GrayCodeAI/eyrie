import {
  API_PROVIDER_DETECTION_ORDER,
  type APIProvider,
} from '../../config/providerProfiles.js'
import { PROVIDER_PRESENCE_CHECKS } from './providers/index.js'

export function detectProviderFromEnv(
  env: NodeJS.ProcessEnv = process.env,
): APIProvider {
  for (const provider of API_PROVIDER_DETECTION_ORDER) {
    if (PROVIDER_PRESENCE_CHECKS[provider](env)) {
      return provider
    }
  }
  return 'anthropic'
}
