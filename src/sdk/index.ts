/**
 * Re-exports from @anthropic-ai/sdk
 * 
 * This allows hawk to import everything from @hawk/eyrie
 * following the langdag pattern (using Anthropic SDK)
 */

// Core SDK
export { default as Anthropic } from '@anthropic-ai/sdk'
export {
  APIError,
  APIConnectionError,
  APIConnectionTimeoutError,
  APIUserAbortError,
} from '@anthropic-ai/sdk'

// Re-export all message-related types from the SDK
export type {
  // Content blocks
  ContentBlock,
  ContentBlockParam,
  TextBlock,
  TextBlockParam,
  ImageBlockParam,
  ToolUseBlock,
  ToolUseBlockParam,
  ToolResultBlockParam,
  ThinkingBlock,
  ThinkingBlockParam,
  RedactedThinkingBlock,
  RedactedThinkingBlockParam,
  
  // Messages
  Message,
  MessageParam,
  
  // Tools
  Tool,
  ToolUnion,
  ToolChoice,
  
  // Other
  Model,
  Metadata,
  Usage,
} from '@anthropic-ai/sdk/resources/messages.mjs'

// Beta API types
export type {
  BetaMessage,
  BetaMessageParam,
  BetaContentBlock,
  BetaContentBlockParam,
  BetaUsage,
  BetaToolUnion,
} from '@anthropic-ai/sdk/resources/beta/messages/messages.mjs'

// Streaming
export type { Stream } from '@anthropic-ai/sdk/streaming.mjs'

// Client options
export type { ClientOptions } from '@anthropic-ai/sdk'
