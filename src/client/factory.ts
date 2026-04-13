/**
 * Eyrie client factory mirrors the herm/langdag pattern.
 *
 * Provider is detected by which API key is set.
 */

import Anthropic from '@anthropic-ai/sdk'
import {
  PROVIDER_MODEL_ENV_KEYS,
  type APIProvider,
} from '../config/providerProfiles.js'
import { detectProviderFromEnv } from './factory/detection.js'

export interface AnthropicClientConfig {
  apiKey?: string
  defaultHeaders?: Record<string, string>
  timeout?: number
  maxRetries?: number
  fetch?: typeof globalThis.fetch
  provider?: APIProvider
  baseURL?: string
}

export function detectProvider(): APIProvider {
  return detectProviderFromEnv(process.env)
}

export function resolveProviderModelEnvOverride(
  provider: APIProvider = detectProvider(),
  env: NodeJS.ProcessEnv = process.env,
): string | undefined {
  for (const key of PROVIDER_MODEL_ENV_KEYS[provider]) {
    const value = env[key]
    if (typeof value === 'string' && value.trim().length > 0) {
      return value.trim()
    }
  }
  return undefined
}

export function parseCustomHeaders(): Record<string, string> {
  const result: Record<string, string> = {}
  const raw = process.env.GRAYCODE_CUSTOM_HEADERS

  if (typeof raw === 'string' && raw.length > 0) {
    for (const line of raw.split(/\r?\n/)) {
      const trimmed = line.trim()
      if (trimmed.length === 0) continue
      const colon = trimmed.indexOf(':')
      if (colon === -1) continue
      const name = trimmed.slice(0, colon).trim()
      const value = trimmed.slice(colon + 1).trim()
      if (name.length > 0) {
        result[name] = value
      }
    }
  }

  return result
}

export async function createAnthropicClient(
  config: AnthropicClientConfig = {},
): Promise<Anthropic> {
  return new Anthropic({
    apiKey: config.apiKey ?? process.env.ANTHROPIC_API_KEY,
    defaultHeaders: config.defaultHeaders,
    maxRetries: config.maxRetries,
    timeout: config.timeout ?? 600_000,
    dangerouslyAllowBrowser: true,
    ...(config.fetch && { fetch: config.fetch }),
    ...(config.baseURL && { baseURL: config.baseURL }),
  })
}
