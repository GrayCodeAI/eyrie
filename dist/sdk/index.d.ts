/**
 * Re-exports from @graycode-ai/sdk
 *
 * This allows hawk to import everything from @hawk/eyrie
 * without directly depending on @graycode-ai/sdk
 */
export { default as GrayCode } from '@graycode-ai/sdk';
export { APIError, APIConnectionError, APIConnectionTimeoutError, APIUserAbortError, } from '@graycode-ai/sdk';
export type { ContentBlock, ContentBlockParam, TextBlock, TextBlockParam, ImageBlockParam, ToolUseBlock, ToolUseBlockParam, ToolResultBlockParam, ThinkingBlock, ThinkingBlockParam, RedactedThinkingBlock, RedactedThinkingBlockParam, Message, MessageParam, Tool, ToolUnion, Model, StopReason, MessageStreamEvent, MessageCreateParams, Usage, } from '@graycode-ai/sdk/resources/messages.mjs';
export type { BetaMessage, BetaMessageParam, BetaContentBlock, BetaContentBlockParam, BetaUsage, BetaToolUnion, } from '@graycode-ai/sdk/resources/beta/messages/messages.mjs';
export type { Base64ImageSource, } from '@graycode-ai/sdk/resources/index.mjs';
export type { Stream } from '@graycode-ai/sdk/streaming.mjs';
export type { ClientOptions } from '@graycode-ai/sdk';
//# sourceMappingURL=index.d.ts.map