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

export type {
  ContentBlock as AnthropicContentBlock,
  ContentBlockParam as AnthropicContentBlockParam,
  TextBlock as AnthropicTextBlock,
  TextBlockParam as AnthropicTextBlockParam,
  ImageBlockParam as AnthropicImageBlockParam,
  ToolUseBlock as AnthropicToolUseBlock,
  ToolUseBlockParam as AnthropicToolUseBlockParam,
  ToolResultBlockParam as AnthropicToolResultBlockParam,
  Message as AnthropicMessage,
  MessageParam as AnthropicMessageParam,
  Tool as AnthropicTool,
  ToolUnion as AnthropicToolUnion,
  ToolChoice as AnthropicToolChoice,
  Model as AnthropicModel,
  Usage as AnthropicUsage,
} from '@anthropic-ai/sdk/resources/messages.mjs'

export type {
  BetaMessage as AnthropicBetaMessage,
  BetaMessageParam as AnthropicBetaMessageParam,
  BetaContentBlock as AnthropicBetaContentBlock,
  BetaContentBlockParam as AnthropicBetaContentBlockParam,
  BetaUsage as AnthropicBetaUsage,
  BetaToolUnion as AnthropicBetaToolUnion,
} from '@anthropic-ai/sdk/resources/beta/messages/messages.mjs'

export type { Stream as AnthropicStream } from '@anthropic-ai/sdk/streaming.mjs'
export type { ClientOptions as AnthropicClientOptions } from '@anthropic-ai/sdk'

// ============================================================================
// OpenAI (GPT-4, GPT-3.5)
// ============================================================================
export { default as OpenAI } from 'openai'
export {
  APIError as OpenAIAPIError,
  APIConnectionError as OpenAIAPIConnectionError,
  APIConnectionTimeoutError as OpenAIAPIConnectionTimeoutError,
} from 'openai'

export type {
  ChatCompletion as OpenAIChatCompletion,
  ChatCompletionMessage as OpenAIChatCompletionMessage,
  ChatCompletionMessageParam as OpenAIChatCompletionMessageParam,
  ChatCompletionChunk as OpenAIChatCompletionChunk,
  ChatCompletionTool as OpenAIChatCompletionTool,
  ChatCompletionToolChoiceOption as OpenAIChatCompletionToolChoiceOption,
  ChatCompletionCreateParams as OpenAIChatCompletionCreateParams,
} from 'openai/resources/chat/completions'

export type { Stream as OpenAIStream } from 'openai/streaming'
export type { ClientOptions as OpenAIClientOptions } from 'openai'

// ============================================================================
// Google (Gemini)
// ============================================================================
export { GoogleGenerativeAI as Gemini } from '@google/generative-ai'

export type {
  GenerativeModel as GeminiGenerativeModel,
  GenerateContentRequest as GeminiGenerateContentRequest,
  GenerateContentResult as GeminiGenerateContentResult,
  Content as GeminiContent,
  Part as GeminiPart,
  TextPart as GeminiTextPart,
  InlineDataPart as GeminiInlineDataPart,
  FunctionCallPart as GeminiFunctionCallPart,
  FunctionResponsePart as GeminiFunctionResponsePart,
  Tool as GeminiTool,
  ToolConfig as GeminiToolConfig,
  GenerationConfig as GeminiGenerationConfig,
  SafetySetting as GeminiSafetySetting,
} from '@google/generative-ai'

// ============================================================================
// Groq (Fast inference)
// ============================================================================
export { default as Groq } from 'groq-sdk'

export type {
  ChatCompletion as GroqChatCompletion,
  ChatCompletionMessage as GroqChatCompletionMessage,
  ChatCompletionMessageParam as GroqChatCompletionMessageParam,
  ChatCompletionChunk as GroqChatCompletionChunk,
  ChatCompletionCreateParams as GroqChatCompletionCreateParams,
  ChatCompletionTool as GroqChatCompletionTool,
} from 'groq-sdk/resources/chat/completions'

export type { Stream as GroqStream } from 'groq-sdk/streaming'
export type { ClientOptions as GroqClientOptions } from 'groq-sdk'

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
} as const

export type Provider = typeof PROVIDERS[keyof typeof PROVIDERS]

export interface ProviderConfig {
  name: Provider
  client: Anthropic | OpenAI | Gemini | Groq
  defaultModel: string
  supportsStreaming: boolean
  supportsTools: boolean
}
