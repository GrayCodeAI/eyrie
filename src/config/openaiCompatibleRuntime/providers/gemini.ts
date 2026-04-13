import { GEMINI_RUNTIME_PROFILE } from '../../providerProfiles/providers/gemini.js'
import type { OpenAICompatibleRuntimeProvider } from '../types.js'

export const GEMINI_RUNTIME_PROVIDER: OpenAICompatibleRuntimeProvider = {
  ...GEMINI_RUNTIME_PROFILE,
}
