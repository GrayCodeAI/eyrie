/**
 * Unified Provider Registry
 * 
 * Following the pi-mono pattern - one SDK, many providers via baseUrl
 */

import { OpenAI } from 'openai'
import { Anthropic } from '@anthropic-ai/sdk'

// ============================================================================
// Provider Types
// ============================================================================

export type ProviderType = 
  | 'anthropic'
  | 'openai'
  | 'openai-compatible'

export interface ProviderConfig {
  name: string
  type: ProviderType
  baseUrl?: string
  envKey: string
  supportsStreaming: boolean
  supportsTools: boolean
  supportsReasoning: boolean
  compat?: OpenAICompatConfig
}

export interface OpenAICompatConfig {
  supportsStore?: boolean
  supportsDeveloperRole?: boolean
  supportsReasoningEffort?: boolean
  supportsUsageInStreaming?: boolean
  supportsStrictMode?: boolean
  maxTokensField?: 'max_completion_tokens' | 'max_tokens'
  requiresToolResultName?: boolean
  requiresAssistantAfterToolResult?: boolean
  requiresThinkingAsText?: boolean
  thinkingFormat?: 'openai' | 'zai' | 'qwen' | 'openrouter'
  reasoningEffortMap?: Partial<Record<string, string>>
}

// ============================================================================
// Core SDK Providers (Dedicated SDKs)
// ============================================================================

export const CORE_PROVIDERS: Record<string, ProviderConfig> = {
  anthropic: {
    name: 'anthropic',
    type: 'anthropic',
    envKey: 'ANTHROPIC_API_KEY',
    supportsStreaming: true,
    supportsTools: true,
    supportsReasoning: true,
  },
  
  openai: {
    name: 'openai',
    type: 'openai',
    baseUrl: 'https://api.openai.com/v1',
    envKey: 'OPENAI_API_KEY',
    supportsStreaming: true,
    supportsTools: true,
    supportsReasoning: true,
    compat: {
      supportsStore: true,
      supportsDeveloperRole: true,
      supportsReasoningEffort: true,
    }
  },
}

// ============================================================================
// OpenAI-Compatible Providers (Using OpenAI SDK with custom baseUrl)
// ============================================================================

export const OPENAI_COMPATIBLE_PROVIDERS: Record<string, ProviderConfig> = {
  // Grok (xAI)
  grok: {
    name: 'grok',
    type: 'openai-compatible',
    baseUrl: 'https://api.x.ai/v1',
    envKey: 'XAI_API_KEY',
    supportsStreaming: true,
    supportsTools: true,
    supportsReasoning: true,
    compat: {
      supportsReasoningEffort: false,
      supportsStore: false,
      supportsDeveloperRole: false,
    }
  },
  
  // OpenRouter
  openrouter: {
    name: 'openrouter',
    type: 'openai-compatible',
    baseUrl: 'https://openrouter.ai/api/v1',
    envKey: 'OPENROUTER_API_KEY',
    supportsStreaming: true,
    supportsTools: true,
    supportsReasoning: true,
    compat: {
      thinkingFormat: 'openrouter',
    }
  },
  
  // Canopy Wave
  canopywave: {
    name: 'canopywave',
    type: 'openai-compatible',
    baseUrl: 'https://inference.canopywave.io/v1',
    envKey: 'CANOPYWAVE_API_KEY',
    supportsStreaming: true,
    supportsTools: true,
    supportsReasoning: true,
  },
  
  // Gemini
  gemini: {
    name: 'gemini',
    type: 'openai-compatible',
    baseUrl: 'https://api.gemini.google.com/v1/forward',
    envKey: 'GEMINI_API_KEY',
    supportsStreaming: true,
    supportsTools: true,
    supportsReasoning: true,
  },
  
  // Ollama
  ollama: {
    name: 'ollama',
    type: 'openai-compatible',
    baseUrl: 'http://localhost:11434/v1',
    envKey: 'OLLAMA_API_KEY',
    supportsStreaming: true,
    supportsTools: true,
    supportsReasoning: false,
    compat: {
      supportsStore: false,
      supportsDeveloperRole: false,
      supportsReasoningEffort: false,
    }
  },
}

// ============================================================================
// All Providers
// ============================================================================

export const ALL_PROVIDERS: Record<string, ProviderConfig> = {
  ...CORE_PROVIDERS,
  ...OPENAI_COMPATIBLE_PROVIDERS
}

export const PROVIDER_LIST = Object.keys(ALL_PROVIDERS)

// ============================================================================
// Helper Functions
// ============================================================================

export function getProvider(name: string): ProviderConfig | undefined {
  return ALL_PROVIDERS[name]
}

export function getAllProviders(): ProviderConfig[] {
  return Object.values(ALL_PROVIDERS)
}

export function getOpenAICompatibleProviders(): ProviderConfig[] {
  return Object.values(OPENAI_COMPATIBLE_PROVIDERS)
}

export function isOpenAICompatible(provider: string): boolean {
  const config = ALL_PROVIDERS[provider]
  return config?.type === 'openai-compatible'
}

export function createOpenAIClient(
  provider: string,
  apiKey?: string,
  baseUrl?: string
): OpenAI {
  const config = ALL_PROVIDERS[provider]
  if (!config) {
    throw new Error(`Unknown provider: ${provider}`)
  }
  
  if (config.type !== 'openai-compatible') {
    throw new Error(`Provider ${provider} is not OpenAI-compatible`)
  }
  
  const key = apiKey || process.env[config.envKey]
  if (!key) {
    throw new Error(
      `API key required for ${provider}. ` +
      `Set ${config.envKey} environment variable or pass apiKey option.`
    )
  }
  
  return new OpenAI({
    apiKey: key,
    baseURL: baseUrl || config.baseUrl,
    dangerouslyAllowBrowser: true,
  })
}

// ============================================================================
// Environment API Keys
// ============================================================================

export function getEnvApiKey(provider: string): string | undefined {
  const config = ALL_PROVIDERS[provider]
  if (!config) return undefined
  return process.env[config.envKey]
}

export function checkApiKey(provider: string): boolean {
  return !!getEnvApiKey(provider)
}