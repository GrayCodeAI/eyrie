/**
 * eyrie - Universal LLM Client
 * 
 * Following the langdag pattern:
 * - Wraps standard SDKs (OpenAI, Anthropic, etc.)
 * - Supports 54+ providers via baseUrl config
 * - Simple API for hawk to use
 */

import OpenAI from 'openai'
import { Anthropic } from '@anthropic-ai/sdk'
import { GoogleGenerativeAI } from '@google/generative-ai'
import { Mistral } from '@mistralai/mistralai'
import { Groq } from 'groq-sdk'
import { 
  CORE_PROVIDERS, 
  OPENAI_COMPATIBLE_PROVIDERS, 
  type ProviderConfig,
  type ProviderType 
} from '../providers/registry.js'

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
  
  /**
   * Create provider client
   */
  private createClient(provider: string, apiKey: string): ProviderClient {
    const config = this.getProviderConfig(provider)
    const type = config.type
    
    // Handle special cases by provider name
    if (provider === 'groq') {
      return this.createGroqClient(apiKey)
    }
    
    switch (type) {
      case 'anthropic':
        return this.createAnthropicClient(apiKey)
      case 'openai':
        return this.createOpenAIClient(apiKey, config.baseUrl)
      case 'openai-compatible':
        return this.createOpenAICompatibleClient(apiKey, config.baseUrl)
      case 'google':
        return this.createGoogleClient(apiKey)
      case 'mistral':
        return this.createMistralClient(apiKey)
      default:
        return this.createOpenAICompatibleClient(apiKey, config.baseUrl)
    }
  }
  
  /**
   * Anthropic (Claude) client
   */
  private createAnthropicClient(apiKey: string): ProviderClient {
    const client = new Anthropic({ apiKey })
    
    return {
      chat: async (messages, options) => {
        const response = await client.messages.create({
          model: options?.model || 'claude-3-5-sonnet-20241022',
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
  private createOpenAIClient(apiKey: string, baseUrl?: string): ProviderClient {
    const client = new OpenAI({ 
      apiKey, 
      baseURL: baseUrl 
    })
    
    return {
      chat: async (messages, options) => {
        const response = await client.chat.completions.create({
          model: options?.model || 'gpt-4o-mini',
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
          toolCalls: choice?.message?.tool_calls?.map(tc => ({
            name: tc.function.name,
            arguments: JSON.parse(tc.function.arguments)
          })),
          finishReason: (choice?.finish_reason as string) || 'unknown'
        }
      }
    }
  }
  
  /**
   * OpenAI-compatible client (for all other providers)
   */
  private createOpenAICompatibleClient(apiKey: string, baseUrl?: string): ProviderClient {
    const client = new OpenAI({ 
      apiKey, 
      baseURL: baseUrl 
    })
    
    return {
      chat: async (messages, options) => {
        const response = await client.chat.completions.create({
          model: options?.model || 'gpt-4o-mini',
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
  
  /**
   * Google Gemini client
   */
  private createGoogleClient(apiKey: string): ProviderClient {
    const client = new GoogleGenerativeAI(apiKey)
    
    return {
      chat: async (messages, options) => {
        const model = client.getGenerativeModel({ 
          model: options?.model || 'gemini-1.5-flash' 
        })
        
        const chat = model.startChat({
          history: messages.slice(0, -1).map(m => ({
            role: m.role === 'user' ? 'user' : 'model',
            parts: [{ text: m.content }]
          })),
          generationConfig: {
            temperature: options?.temperature,
            maxOutputTokens: options?.maxTokens
          }
        })
        
        const result = await chat.sendMessage(messages[messages.length - 1].content)
        
        return {
          content: result.response.text(),
          finishReason: 'stop'
        }
      }
    }
  }
  
  /**
   * Mistral client
   */
  private createMistralClient(apiKey: string): ProviderClient {
    const client = new Mistral({ apiKey })
    
    return {
      chat: async (messages, options) => {
        const response = await client.chat.complete({
          model: options?.model || 'mistral-small-latest',
          messages: messages.map(m => ({
            role: m.role,
            content: m.content
          })) as any,
          temperature: options?.temperature,
          maxTokens: options?.maxTokens
        }) as any
        
        const choice = response.choices?.[0]
        return {
          content: choice?.message?.content || '',
          usage: response.usage ? {
            promptTokens: Number(response.usage.prompt_tokens) || 0,
            completionTokens: Number(response.usage.completion_tokens) || 0,
            totalTokens: Number(response.usage.total_tokens) || 0
          } : undefined,
          finishReason: (choice?.finish_reason as string) || 'stop'
        }
      }
    }
  }
  
  /**
   * Groq client
   */
  private createGroqClient(apiKey: string): ProviderClient {
    const client = new Groq({ apiKey })
    
    return {
      chat: async (messages, options) => {
        const response = await client.chat.completions.create({
          model: options?.model || 'llama-3.1-70b-versatile',
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
   * Send a chat message
   */
  async chat(messages: EyrieMessage[], options?: {
    provider?: string
    model?: string
    temperature?: number
    maxTokens?: number
  }): Promise<EyrieResponse> {
    const provider = options?.provider || this.defaultProvider
    const client = this.getClient(provider)
    
    return client.chat(messages, {
      model: options?.model,
      temperature: options?.temperature,
      maxTokens: options?.maxTokens
    })
  }
  
  /**
   * Stream chat response
   */
  async *stream(messages: EyrieMessage[], options?: {
    provider?: string
    model?: string
    temperature?: number
    maxTokens?: number
  }): AsyncGenerator<EyrieStreamEvent> {
    const provider = options?.provider || this.defaultProvider
    const client = this.getClient(provider)
    
    if (client.streamChat) {
      yield* client.streamChat(messages, options)
    } else {
      const response = await this.chat(messages, options)
      yield { type: 'content', content: response.content }
      yield { type: 'done' }
    }
  }
  
  /**
   * List available providers
   */
  listProviders(): string[] {
    const coreProviders = Object.keys(CORE_PROVIDERS)
    const openaiProviders = Object.keys(OPENAI_COMPATIBLE_PROVIDERS)
    return [...coreProviders, ...openaiProviders]
  }
  
  /**
   * Get provider info
   */
  getProviderInfo(provider: string): ProviderConfig {
    return this.getProviderConfig(provider)
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

export { CORE_PROVIDERS, OPENAI_COMPATIBLE_PROVIDERS, type ProviderConfig, type ProviderType }
