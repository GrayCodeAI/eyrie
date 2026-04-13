import { CANOPYWAVE_RUNTIME_PROFILE } from '../../providerProfiles/providers/canopywave.js'
import type { OpenAICompatibleRuntimeProvider } from '../types.js'

export const CANOPYWAVE_RUNTIME_PROVIDER: OpenAICompatibleRuntimeProvider = {
  ...CANOPYWAVE_RUNTIME_PROFILE,
}
