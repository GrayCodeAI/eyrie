/**
 * Multi-Provider LLM SDKs
 * 
 * Re-exports from all major LLM provider SDKs.
 * Support: Anthropic, OpenAI, Google Gemini, Groq
 */

// ============================================================================
// Anthropic (Claude)
// ============================================================================
export { default as Anthropic } from '@anthropic-ai/sdk'
export {
  APIError as AnthropicAPIError,
  APIConnectionError as AnthropicAPIConnectionError,
  APIConnectionTimeoutError as AnthropicAPIConnectionTimeoutError,
  APIUserAbortError as AnthropicAPIUserAbortError,
} from '@anthropic-ai/sdk'

// Mistral types are not exported as named exports - use the Mistral class directly
// // OpenCode types not available in the package - using local definitions instead
// export type { ... } from '@opencode-ai/sdk/client'

// ============================================================================
// Common Types (Provider-agnostic)
// ============================================================================

export interface Message {
  role: 'user' | 'assistant' | 'system'
  content: string
}

export interface CompletionRequest {
  model: string
  messages: Message[]
  max_tokens?: number
  temperature?: number
  top_p?: number
  stream?: boolean
  tools?: Tool[]
}

export interface CompletionResponse {
  content: string
  usage?: {
    input_tokens: number
    output_tokens: number
  }
}

export interface Tool {
  type: 'function'
  function: {
    name: string
    description: string
    parameters: {
      type: 'object'
      properties?: Record<string, unknown>
      required?: string[]
    }
  }
}

export interface StreamEvent {
  type: 'content' | 'tool_call' | 'tool_result' | 'done' | 'error'
  content?: string
  tool_call?: {
    name: string
    arguments: Record<string, unknown>
  }
  error?: string
}

// ============================================================================
// Provider Registry
// ============================================================================

export const PROVIDERS = {
  ANTHROPIC: 'anthropic',
  OPENAI: 'openai',
  GEMINI: 'gemini',
  GROQ: 'groq',
  VERCEL: 'vercel',
  VERTEX: 'vertex',
  BEDROCK: 'bedrock',
  MISTRAL: 'mistral',
  OPENCODE: 'opencode',
} as const

export type Provider = typeof PROVIDERS[keyof typeof PROVIDERS]

export interface ProviderConfig {
  name: Provider
  client: unknown
  defaultModel: string
  supportsStreaming: boolean
  supportsTools: boolean
}
