/**
 * Re-exports from @graycode-ai/sdk
 * 
 * This allows hawk to import SDK types from @hawk/eyrie instead of
 * directly from npm, keeping all dependencies centralized.
 */

// Core SDK and errors
export { default as GrayCode } from '@graycode-ai/sdk'
export {
  APIError,
  APIConnectionError,
  APIConnectionTimeoutError,
  APIUserAbortError,
} from '@graycode-ai/sdk'

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
  
  // Other
  Model,
  StopReason,
  MessageStreamEvent,
  MessageCreateParams,
} from '@graycode-ai/sdk/resources/messages.mjs'

// Beta API types
export type {
  BetaMessage,
  BetaMessageParam,
  BetaContentBlock,
  BetaContentBlockParam,
  BetaUsage,
  BetaToolUnion,
} from '@graycode-ai/sdk/resources/beta/messages/messages.mjs'

// Streaming
export type { Stream } from '@graycode-ai/sdk/streaming.mjs'
