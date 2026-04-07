/**
 * Core message types for LLM conversations
 * 
 * Base types used across the system for message handling.
 * These are intentionally minimal and can be extended in hawk.
 */

import type { ContentBlockParam } from './sdk.js'

// ============================================================================
// Content Blocks
// ============================================================================

/**
 * Text content block
 */
export interface TextBlock {
  type: 'text'
  text: string
}

/**
 * Image content block
 */
export interface ImageBlock {
  type: 'image'
  source: {
    type: 'base64'
    media_type: 'image/jpeg' | 'image/png' | 'image/gif' | 'image/webp'
    data: string
  }
}

/**
 * Tool use block - when assistant uses a tool
 */
export interface ToolUseBlock {
  type: 'tool_use'
  id: string
  name: string
  input: Record<string, unknown>
}

/**
 * Tool result block - result from tool execution
 */
export interface ToolResultBlock {
  type: 'tool_result'
  tool_use_id: string
  content: string | Array<TextBlock | ImageBlock>
  is_error?: boolean
}

/**
 * Union type for all content blocks
 */
export type ContentBlock = TextBlock | ImageBlock | ToolUseBlock | ToolResultBlock

// ============================================================================
// Base Message Types
// ============================================================================

/**
 * Base message interface
 */
export interface Message {
  role: 'user' | 'assistant' | 'system'
  content: string | ContentBlock[]
}

/**
 * User message
 */
export interface UserMessage extends Message {
  role: 'user'
}

/**
 * Assistant message
 */
export interface AssistantMessage extends Message {
  role: 'assistant'
  /**
   * Indicates this message represents an API error
   */
  isApiErrorMessage?: boolean
  /**
   * Error details if this is an error message
   */
  errorDetails?: {
    actualTokens?: number
    limitTokens?: number
  }
}

/**
 * System message
 */
export interface SystemMessage extends Message {
  role: 'system'
}

// ============================================================================
// Message Origins & Sources
// ============================================================================

/**
 * Where a message originated from
 */
export type MessageOrigin = 
  | 'user' 
  | 'api' 
  | 'tool' 
  | 'system' 
  | 'compact'
  | 'recovery'

/**
 * Source of the message (for mailbox/pipeline)
 */
export type MessageSource = 
  | 'user' 
  | 'teammate' 
  | 'system' 
  | 'tick' 
  | 'task'

// ============================================================================
// Utility Types
// ============================================================================

/**
 * Type guard for text blocks
 */
export function isTextBlock(block: unknown): block is TextBlock {
  return (
    typeof block === 'object' &&
    block !== null &&
    'type' in block &&
    block.type === 'text' &&
    'text' in block &&
    typeof block.text === 'string'
  )
}

/**
 * Type guard for image blocks
 */
export function isImageBlock(block: unknown): block is ImageBlock {
  return (
    typeof block === 'object' &&
    block !== null &&
    'type' in block &&
    block.type === 'image' &&
    'source' in block
  )
}

/**
 * Type guard for tool use blocks
 */
export function isToolUseBlock(block: unknown): block is ToolUseBlock {
  return (
    typeof block === 'object' &&
    block !== null &&
    'type' in block &&
    block.type === 'tool_use' &&
    'id' in block &&
    'name' in block
  )
}

/**
 * Type guard for tool result blocks
 */
export function isToolResultBlock(block: unknown): block is ToolResultBlock {
  return (
    typeof block === 'object' &&
    block !== null &&
    'type' in block &&
    block.type === 'tool_result' &&
    'tool_use_id' in block
  )
}

// ============================================================================
// Conversion Utilities
// ============================================================================

/**
 * Convert content blocks to SDK format
 */
export function toSDKContentBlocks(
  blocks: ContentBlock[],
): ContentBlockParam[] {
  return blocks.map(block => {
    if (block.type === 'text') {
      return { type: 'text', text: block.text }
    }
    if (block.type === 'image') {
      return {
        type: 'image',
        source: block.source,
      }
    }
    if (block.type === 'tool_use') {
      return {
        type: 'tool_use',
        id: block.id,
        name: block.name,
        input: block.input,
      }
    }
    if (block.type === 'tool_result') {
      return {
        type: 'tool_result',
        tool_use_id: block.tool_use_id,
        content: block.content,
        is_error: block.is_error,
      }
    }
    throw new Error(`Unknown block type: ${(block as { type: string }).type}`)
  })
}

/**
 * Create a user message with text content
 */
export function createUserMessage(text: string): UserMessage {
  return {
    role: 'user',
    content: text,
  }
}

/**
 * Create an assistant message with text content
 */
export function createAssistantMessage(text: string): AssistantMessage {
  return {
    role: 'assistant',
    content: text,
  }
}

/**
 * Create a system message
 */
export function createSystemMessage(text: string): SystemMessage {
  return {
    role: 'system',
    content: text,
  }
}
