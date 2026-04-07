/**
 * Core message types for LLM conversations
 *
 * Base types used across the system for message handling.
 * These are intentionally minimal and can be extended in hawk.
 */
import type { ContentBlockParam } from './sdk.js';
/**
 * Text content block
 */
export interface TextBlock {
    type: 'text';
    text: string;
}
/**
 * Image content block
 */
export interface ImageBlock {
    type: 'image';
    source: {
        type: 'base64';
        media_type: 'image/jpeg' | 'image/png' | 'image/gif' | 'image/webp';
        data: string;
    };
}
/**
 * Tool use block - when assistant uses a tool
 */
export interface ToolUseBlock {
    type: 'tool_use';
    id: string;
    name: string;
    input: Record<string, unknown>;
}
/**
 * Tool result block - result from tool execution
 */
export interface ToolResultBlock {
    type: 'tool_result';
    tool_use_id: string;
    content: string | Array<TextBlock | ImageBlock>;
    is_error?: boolean;
}
/**
 * Union type for all content blocks
 */
export type ContentBlock = TextBlock | ImageBlock | ToolUseBlock | ToolResultBlock;
/**
 * Base message interface
 */
export interface Message {
    role: 'user' | 'assistant' | 'system';
    content: string | ContentBlock[];
}
/**
 * User message
 */
export interface UserMessage extends Message {
    role: 'user';
}
/**
 * Assistant message
 */
export interface AssistantMessage extends Message {
    role: 'assistant';
    /**
     * Indicates this message represents an API error
     */
    isApiErrorMessage?: boolean;
    /**
     * Error details if this is an error message
     */
    errorDetails?: {
        actualTokens?: number;
        limitTokens?: number;
    };
}
/**
 * System message
 */
export interface SystemMessage extends Message {
    role: 'system';
}
/**
 * Where a message originated from
 */
export type MessageOrigin = 'user' | 'api' | 'tool' | 'system' | 'compact' | 'recovery';
/**
 * Source of the message (for mailbox/pipeline)
 */
export type MessageSource = 'user' | 'teammate' | 'system' | 'tick' | 'task';
/**
 * Type guard for text blocks
 */
export declare function isTextBlock(block: unknown): block is TextBlock;
/**
 * Type guard for image blocks
 */
export declare function isImageBlock(block: unknown): block is ImageBlock;
/**
 * Type guard for tool use blocks
 */
export declare function isToolUseBlock(block: unknown): block is ToolUseBlock;
/**
 * Type guard for tool result blocks
 */
export declare function isToolResultBlock(block: unknown): block is ToolResultBlock;
/**
 * Convert content blocks to SDK format
 */
export declare function toSDKContentBlocks(blocks: ContentBlock[]): ContentBlockParam[];
/**
 * Create a user message with text content
 */
export declare function createUserMessage(text: string): UserMessage;
/**
 * Create an assistant message with text content
 */
export declare function createAssistantMessage(text: string): AssistantMessage;
/**
 * Create a system message
 */
export declare function createSystemMessage(text: string): SystemMessage;
//# sourceMappingURL=message.d.ts.map