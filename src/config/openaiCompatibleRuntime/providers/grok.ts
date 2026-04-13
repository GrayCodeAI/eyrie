import { GROK_RUNTIME_PROFILE } from '../../providerProfiles/providers/grok.js'
import type { OpenAICompatibleRuntimeProvider } from '../types.js'

export const GROK_RUNTIME_PROVIDER: OpenAICompatibleRuntimeProvider = {
  ...GROK_RUNTIME_PROFILE,
}
