import { ANTHROPIC_RUNTIME_PROFILE } from '../../providerProfiles/providers/anthropic.js'
import type { OpenAICompatibleRuntimeProvider } from '../types.js'

export const ANTHROPIC_RUNTIME_PROVIDER: OpenAICompatibleRuntimeProvider = {
  ...ANTHROPIC_RUNTIME_PROFILE,
}
