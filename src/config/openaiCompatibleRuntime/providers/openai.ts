import { OPENAI_RUNTIME_PROFILE } from '../../providerProfiles/providers/openai.js'
import type { OpenAICompatibleRuntimeProvider } from '../types.js'

export const OPENAI_RUNTIME_PROVIDER: OpenAICompatibleRuntimeProvider = {
  ...OPENAI_RUNTIME_PROFILE,
}
