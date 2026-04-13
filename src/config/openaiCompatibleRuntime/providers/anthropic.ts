import { OPENAI_COMPATIBLE_RUNTIME_PROFILES } from '../../providerProfiles.js'
import type { OpenAICompatibleRuntimeProvider } from '../types.js'

export const ANTHROPIC_RUNTIME_PROVIDER: OpenAICompatibleRuntimeProvider = {
  ...OPENAI_COMPATIBLE_RUNTIME_PROFILES.anthropic,
}
