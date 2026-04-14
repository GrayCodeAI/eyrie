import { OPENCODEGO_RUNTIME_PROFILE } from '../../providerProfiles/providers/opencodego.js'
import type { OpenAICompatibleRuntimeProvider } from '../types.js'

export const OPENCODEGO_RUNTIME_PROVIDER: OpenAICompatibleRuntimeProvider = {
  ...OPENCODEGO_RUNTIME_PROFILE,
}
