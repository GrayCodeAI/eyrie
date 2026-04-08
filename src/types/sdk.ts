/**
 * SDK Type Definitions
 * 
 * These types mirror the @graycode-ai/sdk types but are defined locally
 * to make eyrie completely independent of npm dependencies.
 */

// ============================================================================
// Base Types
// ============================================================================

export interface TextBlock {
  type: 'text'
  text: string
}

export interface TextBlockParam {
  type: 'text'
  text: string
}

export interface Base64ImageSource {
  type: 'base64'
  media_type: 'image/jpeg' | 'image/png' | 'image/gif' | 'image/webp'
  data: string
}

export interface ImageBlockParam {
  type: 'image'
  source: Base64ImageSource
}

// Alias for backward compatibility
export type ImageBlock = ImageBlockParam

export interface ToolUseBlock {
  type: 'tool_use'
  id: string
  name: string
  input: Record<string, unknown>
}

export interface ToolUseBlockParam {
  type: 'tool_use'
  id: string
  name: string
  input: Record<string, unknown>
}

export interface ToolResultBlockParam {
  type: 'tool_result'
  tool_use_id: string
  content: string | Array<TextBlockParam | ImageBlockParam>
  is_error?: boolean
}

// Alias for backward compatibility  
export type ToolResultBlock = ToolResultBlockParam

export interface ThinkingBlock {
  type: 'thinking'
  thinking: string
  signature: string
}

export interface ThinkingBlockParam {
  type: 'thinking'
  thinking: string
  signature?: string
}

export interface RedactedThinkingBlock {
  type: 'redacted_thinking'
  data: string
}

export interface RedactedThinkingBlockParam {
  type: 'redacted_thinking'
  data: string
}

// ============================================================================
// Content Blocks
// ============================================================================

export type ContentBlock = 
  | TextBlock 
  | ThinkingBlock 
  | RedactedThinkingBlock 
  | ToolUseBlock

export type ContentBlockParam = 
  | TextBlockParam 
  | ImageBlockParam 
  | ThinkingBlockParam 
  | RedactedThinkingBlockParam 
  | ToolUseBlockParam 
  | ToolResultBlockParam

// ============================================================================
// Messages
// ============================================================================

export interface Message {
  id?: string
  type?: 'message'
  role: 'user' | 'assistant' | 'system'
  content: string | ContentBlock[]
  model?: string
  stop_reason?: 'end_turn' | 'max_tokens' | 'stop_sequence' | 'tool_use' | null
  stop_sequence?: string | null
  usage?: Usage
}

export interface MessageParam {
  role: 'assistant' | 'user'
  content: string | ContentBlockParam[]
}

// ============================================================================
// Beta Types
// ============================================================================

export interface BetaMessage {
  id: string
  type: 'message'
  role: 'assistant' | 'user'
  content: BetaContentBlock[]
  model: string
  stop_reason: 'end_turn' | 'max_tokens' | 'stop_sequence' | 'tool_use' | null
  stop_sequence: string | null
  usage: BetaUsage
}

export interface BetaMessageParam {
  role: 'assistant' | 'user'
  content: string | BetaContentBlockParam[]
}

export type BetaContentBlock = ContentBlock
export type BetaContentBlockParam = ContentBlockParam

export interface BetaUsage {
  input_tokens: number
  output_tokens: number
}

export type BetaToolUnion = Record<string, unknown>

// ============================================================================
// Tools
// ============================================================================

export interface Tool {
  name: string
  description?: string
  input_schema: {
    type: 'object'
    properties?: Record<string, unknown>
    required?: string[]
  }
}

export type ToolUnion = Tool

// ============================================================================
// Other Types
// ============================================================================

export type Model = string

export type StopReason = 'end_turn' | 'max_tokens' | 'stop_sequence' | 'tool_use' | null

export interface MessageStreamEvent {
  type: string
}

export interface MessageCreateParams {
  model: string
  max_tokens: number
  messages: MessageParam[]
  tools?: Tool[]
  tool_choice?: { type: 'auto' | 'any' | 'tool'; name?: string }
  system?: string
  temperature?: number
  top_p?: number
  top_k?: number
  stop_sequences?: string[]
  stream?: boolean
}

// Alias for backward compatibility with Beta API
export type BetaMessageStreamParams = MessageCreateParams

export interface Usage {
  input_tokens: number
  output_tokens: number
}

// ============================================================================
// Streaming
// ============================================================================

export interface Stream<T> {
  [Symbol.asyncIterator](): AsyncIterator<T>
}

// ============================================================================
// API Errors
// ============================================================================

export class APIError extends Error {
  readonly status: number | undefined
  readonly headers: Record<string, string> | undefined
  readonly error: Record<string, unknown> | undefined
  
  constructor(
    status: number | undefined,
    error: Record<string, unknown> | undefined,
    message: string | undefined,
    headers: Record<string, string> | undefined
  ) {
    super(message || 'API Error')
    this.status = status
    this.error = error
    this.headers = headers
  }
}

export class APIConnectionError extends APIError {
  constructor(message?: string) {
    super(undefined, undefined, message || 'Connection error', undefined)
  }
}

export class APIConnectionTimeoutError extends APIError {
  constructor(message?: string) {
    super(undefined, undefined, message || 'Connection timeout', undefined)
  }
}

export class APIUserAbortError extends APIError {
  constructor(message: string) {
    super(undefined, undefined, message, undefined)
  }
}

export class NotFoundError extends APIError {
  constructor(message: string) {
    super(undefined, undefined, message, undefined)
  }
}

export class AuthenticationError extends APIError {
  constructor(message: string) {
    super(undefined, undefined, message, undefined)
  }
}


// ============================================================================
// Helper Types
// ============================================================================

export type MessageOrigin = 'user' | 'api' | 'tool' | 'system' | 'compact' | 'recovery'
export type MessageSource = 'user' | 'teammate' | 'system' | 'tick' | 'task'

// ============================================================================
// Extended Message Types
// ============================================================================

export interface UserMessage extends Message {
  role: 'user'
}

export interface AssistantMessage extends Message {
  role: 'assistant'
  isApiErrorMessage?: boolean
  errorDetails?: {
    actualTokens?: number
    limitTokens?: number
  }
}

export interface SystemMessage extends Message {
  role: 'system'
}

// ============================================================================
// Type Guards
// ============================================================================

export function isTextBlock(block: unknown): block is TextBlock {
  return (
    typeof block === 'object' &&
    block !== null &&
    'type' in block &&
    (block as { type: string }).type === 'text' &&
    'text' in block &&
    typeof (block as { text: string }).text === 'string'
  )
}

export function isImageBlock(block: unknown): block is ImageBlockParam {
  return (
    typeof block === 'object' &&
    block !== null &&
    'type' in block &&
    (block as { type: string }).type === 'image' &&
    'source' in block
  )
}

export function isToolUseBlock(block: unknown): block is ToolUseBlock {
  return (
    typeof block === 'object' &&
    block !== null &&
    'type' in block &&
    (block as { type: string }).type === 'tool_use' &&
    'id' in block &&
    'name' in block
  )
}

export function isToolResultBlock(block: unknown): block is ToolResultBlockParam {
  return (
    typeof block === 'object' &&
    block !== null &&
    'type' in block &&
    (block as { type: string }).type === 'tool_result' &&
    'tool_use_id' in block
  )
}

// ============================================================================
// Creator Functions
// ============================================================================

export function createUserMessage(text: string): UserMessage {
  return {
    role: 'user',
    content: text,
  } as UserMessage
}

export function createAssistantMessage(text: string): AssistantMessage {
  return {
    role: 'assistant',
    content: text,
  } as AssistantMessage
}

export function createSystemMessage(text: string): SystemMessage {
  return {
    role: 'system',
    content: text,
  } as SystemMessage
}

export function toSDKContentBlocks(blocks: ContentBlock[]): ContentBlockParam[] {
  return blocks.map(block => {
    if (block.type === 'text') {
      return { type: 'text', text: block.text }
    }
    throw new Error(`Block type conversion not implemented: ${(block as { type: string }).type}`)
  })
}

// ============================================================================
// GrayCode Client (Stub)
// ============================================================================

export interface ClientOptions {
  apiKey?: string
  baseURL?: string
  timeout?: number
  httpAgent?: unknown
  maxRetries?: number
  fetch?: typeof globalThis.fetch
  fetchOptions?: RequestInit
  logger?: {
    error: (msg: string, ...args: unknown[]) => void
    warn: (msg: string, ...args: unknown[]) => void
    info: (msg: string, ...args: unknown[]) => void
    debug: (msg: string, ...args: unknown[]) => void
  }
}

export class GrayCode {
  constructor(options?: ClientOptions) {}
  
  messages = {
    create: async (_params: MessageCreateParams): Promise<Message> => {
      throw new Error('GrayCode client is a type stub. Use actual SDK for runtime.')
    }
  }
}
