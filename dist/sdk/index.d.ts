/**
 * Re-exports from @anthropic-ai/sdk
 *
 * This allows hawk to import everything from @hawk/eyrie
 * following the langdag pattern (using Anthropic SDK)
 */
export { default as Anthropic } from '@anthropic-ai/sdk';
export { APIError, APIConnectionError, APIConnectionTimeoutError, APIUserAbortError, } from '@anthropic-ai/sdk';
export type { ContentBlock, ContentBlockParam, TextBlock, TextBlockParam, ImageBlockParam, ToolUseBlock, ToolUseBlockParam, ToolResultBlockParam, ThinkingBlock, ThinkingBlockParam, RedactedThinkingBlock, RedactedThinkingBlockParam, Message, MessageParam, Tool, ToolUnion, ToolChoice, Model, Metadata, Usage, } from '@anthropic-ai/sdk/resources/messages.mjs';
export type { BetaMessage, BetaMessageParam, BetaContentBlock, BetaContentBlockParam, BetaUsage, BetaToolUnion, } from '@anthropic-ai/sdk/resources/beta/messages/messages.mjs';
export type { Stream } from '@anthropic-ai/sdk/streaming.mjs';
export type { ClientOptions } from '@anthropic-ai/sdk';
//# sourceMappingURL=index.d.ts.map