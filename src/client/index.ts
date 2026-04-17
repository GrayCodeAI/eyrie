/**
 * eyrie - Universal LLM Client
 * 
 * Wraps standard SDKs (OpenAI, Anthropic, etc.)
 * Supports 54+ providers via baseUrl config
 * Simple API for hawk to use
 */

import OpenAI from 'openai'
import { Anthropic } from '@anthropic-ai/sdk'
import {
  CORE_PROVIDERS,
  OPENAI_COMPATIBLE_PROVIDERS,
  type ProviderConfig,
  type ProviderType,
} from '../providers/registry.js'
import { loadModelCatalogSync, modelsForProvider } from '../catalog/modelCatalog.js'
import type { APIProvider } from '../config/providerProfiles.js'

// ============================================================================
// Types
// ============================================================================

export interface EyrieConfig {
  provider?: string
  apiKey?: string
  baseUrl?: string
  model?: string
  maxRetries?: number
}

export interface EyrieMessage {
  role: 'user' | 'assistant' | 'system'
  content: string
  images?: string[]
  tools?: EyrieTool[]
}

export interface EyrieTool {
  name: string
  description: string
  parameters: Record<string, unknown>
}

export interface EyrieUsage {
  promptTokens: number
  completionTokens: number
  totalTokens: number
}

export interface EyrieResponse {
  content: string
  usage?: EyrieUsage
  toolCalls?: Array<{
    name: string
    arguments: Record<string, unknown>
  }>
  finishReason: string
}

export interface EyrieStreamEvent {
  type: 'content' | 'tool_call' | 'done' | 'error'
  content?: string
  toolCall?: {
    name: string
    arguments: string
  }
  error?: string
}

// ============================================================================
// Provider Wrappers
// ============================================================================

interface ProviderClient {
  chat: (messages: EyrieMessage[], options?: {
    model?: string
    temperature?: number
    maxTokens?: number
    tools?: EyrieTool[]
    stream?: boolean
  }) => Promise<EyrieResponse>
  
  streamChat?: (messages: EyrieMessage[], options?: {
    model?: string
    temperature?: number
    maxTokens?: number
    tools?: EyrieTool[]
  }) => AsyncGenerator<EyrieStreamEvent>
}

// ============================================================================
// Main Client
// ============================================================================

export class EyrieClient {
  private clients: Map<string, ProviderClient> = new Map()
  private defaultProvider: string = 'openai'
  private apiKeys: Map<string, string> = new Map()

  constructor(config?: EyrieConfig) {
    if (config?.provider) {
      this.defaultProvider = config.provider
    }
    if (config?.apiKey) {
      this.apiKeys.set(config.provider || this.defaultProvider, config.apiKey)
    }
  }

  /**
   * Set API key for a provider
   */
  setApiKey(provider: string, apiKey: string): void {
    this.apiKeys.set(provider, apiKey)
  }

  /**
   * Get client for a provider
   */
  private getClient(provider: string): ProviderClient {
    if (this.clients.has(provider)) {
      return this.clients.get(provider)!
    }
    
    const apiKey = this.apiKeys.get(provider) || process.env[this.getProviderConfig(provider).envKey]
    if (!apiKey) {
      throw new Error(`No API key found for provider: ${provider}. Set ${this.getProviderConfig(provider).envKey} or call setApiKey()`)
    }
    
    const client = this.createClient(provider, apiKey)
    this.clients.set(provider, client)
    return client
  }

  /**
   * Get provider config
   */
  private getProviderConfig(provider: string): ProviderConfig {
    const core = CORE_PROVIDERS[provider as keyof typeof CORE_PROVIDERS]
    if (core) return core
    
    const openai = OPENAI_COMPATIBLE_PROVIDERS[provider as keyof typeof OPENAI_COMPATIBLE_PROVIDERS]
    if (openai) return openai
    
    throw new Error(`Unknown provider: ${provider}`)
  }

  // ============================================================================
  // Client Factories
  // ============================================================================

  private readonly clientFactories: Partial<Record<
    ProviderType,
    (apiKey: string, config: ProviderConfig & { defaultModel?: string }) => ProviderClient
  >> = {
    anthropic: (apiKey, config) => this.createAnthropicClient(apiKey, config.defaultModel),
    openai: (apiKey, config) => this.createOpenAIClient(apiKey, config.baseUrl, config.defaultModel),
    'openai-compatible': (apiKey, config) => this.createOpenAICompatibleClient(apiKey, config.baseUrl, config.defaultModel),
  }

  /**
   * Resolve the default model for a provider from the eyrie runtime catalog.
   * Returns undefined for providers not tracked in the catalog (e.g. groq, ollama)
   * — callers must supply a model explicitly in that case.
   */
  private resolveDefaultModel(provider: string): string | undefined {
    const catalog = loadModelCatalogSync()
    const models = modelsForProvider(catalog, provider as APIProvider)
    return models.length > 0 ? models[0].id : undefined
  }

  /**
   * Create provider client
   */
  private createClient(provider: string, apiKey: string): ProviderClient {
    const config = this.getProviderConfig(provider)
    const defaultModel = this.resolveDefaultModel(provider)
    const factory = this.clientFactories[config.type]
    if (factory) {
      return factory(apiKey, { ...config, defaultModel } as ProviderConfig & { defaultModel?: string })
    }
    return this.createOpenAICompatibleClient(apiKey, config.baseUrl, defaultModel)
  }

  /**
   * Anthropic (Claude) client
   */
  private createAnthropicClient(apiKey: string, defaultModel: string | undefined): ProviderClient {
    const client = new Anthropic({ apiKey })

    return {
      chat: async (messages, options) => {
        const model = options?.model ?? defaultModel
        if (!model) throw new Error('No model specified for anthropic. Pass model in options or ensure the catalog has an entry for this provider.')
        const response = await client.messages.create({
          model,
          messages: messages.map(m => ({
            role: m.role as 'user' | 'assistant',
            content: m.content
          })),
          max_tokens: options?.maxTokens || 4096,
          temperature: options?.temperature,
          stream: false
        })
        
        return {
          content: response.content[0].type === 'text' ? response.content[0].text : '',
          usage: {
            promptTokens: response.usage.input_tokens,
            completionTokens: response.usage.output_tokens,
            totalTokens: response.usage.input_tokens + response.usage.output_tokens
          },
          finishReason: response.stop_reason || 'unknown'
        }
      }
    }
  }

  /**
   * OpenAI client
   */
  private createOpenAIClient(apiKey: string, baseUrl: string | undefined, defaultModel: string | undefined): ProviderClient {
    const client = new OpenAI({
      apiKey,
      baseURL: baseUrl,
    })

    return {
      chat: async (messages, options) => {
        const model = options?.model ?? defaultModel
        if (!model) throw new Error('No model specified for openai. Pass model in options or ensure the catalog has an entry for this provider.')
        const response = await client.chat.completions.create({
          model,
          messages: messages as any,
          temperature: options?.temperature,
          max_tokens: options?.maxTokens,
          stream: false
        })
        
        const choice = response.choices[0]
        return {
          content: choice?.message?.content || '',
          usage: response.usage ? {
            promptTokens: response.usage.prompt_tokens || 0,
            completionTokens: response.usage.completion_tokens || 0,
            totalTokens: response.usage.total_tokens || 0
          } : undefined,
          toolCalls: choice?.message?.tool_calls
            ?.filter((tc): tc is typeof tc & { function: { name: string; arguments: string } } => 'function' in tc)
            .map(tc => ({
              name: tc.function.name,
              arguments: JSON.parse(tc.function.arguments),
            })),
          finishReason: (choice?.finish_reason as string) || 'unknown'
        }
      }
    }
  }

  /**
   * OpenAI-compatible client (for all other providers, including groq)
   */
  private createOpenAICompatibleClient(apiKey: string, baseUrl: string | undefined, defaultModel: string | undefined): ProviderClient {
    const client = new OpenAI({
      apiKey,
      baseURL: baseUrl,
    })

    return {
      chat: async (messages, options) => {
        const model = options?.model ?? defaultModel
        if (!model) throw new Error(`No model specified for provider at ${baseUrl ?? 'unknown'}. Pass model in options or add this provider to the eyrie catalog.`)
        const response = await client.chat.completions.create({
          model,
          messages: messages as any,
          temperature: options?.temperature,
          max_tokens: options?.maxTokens,
          stream: false
        })
        
        const choice = response.choices[0]
        return {
          content: choice?.message?.content || '',
          usage: response.usage ? {
            promptTokens: response.usage.prompt_tokens || 0,
            completionTokens: response.usage.completion_tokens || 0,
            totalTokens: response.usage.total_tokens || 0
          } : undefined,
          finishReason: (choice?.finish_reason as string) || 'unknown'
        }
      }
    }
  }

  // ============================================================================
  // Public API
  // ============================================================================

  /**
   * Generate content with the specified provider
   */
  async chat(
    messages: EyrieMessage[],
    options?: {
      provider?: string
      model?: string
      temperature?: number
      maxTokens?: number
      tools?: EyrieTool[]
      stream?: boolean
    }
  ): Promise<EyrieResponse> {
    const provider = options?.provider || this.defaultProvider
    const client = this.getClient(provider)
    
    return await client.chat(messages, options)
  }

  /**
   * Stream content with the specified provider
   */
  async *streamChat(
    messages: EyrieMessage[],
    options?: {
      provider?: string
      model?: string
      temperature?: number
      maxTokens?: number
      tools?: EyrieTool[]
    }
  ): AsyncGenerator<EyrieStreamEvent> {
    const provider = options?.provider || this.defaultProvider
    const client = this.getClient(provider)
    
    if (!client.streamChat) {
      throw new Error('Streaming not supported for this provider')
    }
    
    yield* client.streamChat(messages, options)
  }

  /**
   * List available providers
   */
  getProviders(): string[] {
    return Object.keys(CORE_PROVIDERS).concat(Object.keys(OPENAI_COMPATIBLE_PROVIDERS))
  }

  /**
   * Get provider info
   */
  getProviderInfo(provider: string): ProviderConfig | undefined {
    return { ...CORE_PROVIDERS[provider], ...OPENAI_COMPATIBLE_PROVIDERS[provider] }
  }
}

// ============================================================================
// Convenience Functions
// ============================================================================

export function createEyrie(config?: EyrieConfig): EyrieClient {
  return new EyrieClient(config)
}

// ============================================================================
// Re-export provider registry
// ============================================================================

export { CORE_PROVIDERS, OPENAI_COMPATIBLE_PROVIDERS } from '../providers/registry.js'
export type { ProviderConfig, ProviderType } from '../providers/registry.js'
