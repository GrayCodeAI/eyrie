import { OPENROUTER_RUNTIME_PROFILE } from '../../providerProfiles/providers/openrouter.js'
import type { OpenAICompatibleRuntimeProvider } from '../types.js'

export const OPENROUTER_RUNTIME_PROVIDER: OpenAICompatibleRuntimeProvider = {
  ...OPENROUTER_RUNTIME_PROFILE,
}
