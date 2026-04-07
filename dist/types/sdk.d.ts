/**
 * SDK Type Definitions
 *
 * These types mirror the @graycode-ai/sdk types but are defined locally
 * to make eyrie completely independent of npm dependencies.
 */
export interface TextBlock {
    type: 'text';
    text: string;
}
export interface TextBlockParam {
    type: 'text';
    text: string;
}
export interface ImageBlockParam {
    type: 'image';
    source: {
        type: 'base64';
        media_type: 'image/jpeg' | 'image/png' | 'image/gif' | 'image/webp';
        data: string;
    };
}
export type ImageBlock = ImageBlockParam;
export interface ToolUseBlock {
    type: 'tool_use';
    id: string;
    name: string;
    input: Record<string, unknown>;
}
export interface ToolUseBlockParam {
    type: 'tool_use';
    id: string;
    name: string;
    input: Record<string, unknown>;
}
export interface ToolResultBlockParam {
    type: 'tool_result';
    tool_use_id: string;
    content: string | Array<TextBlockParam | ImageBlockParam>;
    is_error?: boolean;
}
export type ToolResultBlock = ToolResultBlockParam;
export interface ThinkingBlock {
    type: 'thinking';
    thinking: string;
    signature: string;
}
export interface ThinkingBlockParam {
    type: 'thinking';
    thinking: string;
    signature?: string;
}
export interface RedactedThinkingBlock {
    type: 'redacted_thinking';
    data: string;
}
export interface RedactedThinkingBlockParam {
    type: 'redacted_thinking';
    data: string;
}
export type ContentBlock = TextBlock | ThinkingBlock | RedactedThinkingBlock | ToolUseBlock;
export type ContentBlockParam = TextBlockParam | ImageBlockParam | ThinkingBlockParam | RedactedThinkingBlockParam | ToolUseBlockParam | ToolResultBlockParam;
export interface Message {
    id?: string;
    type?: 'message';
    role: 'user' | 'assistant' | 'system';
    content: string | ContentBlock[];
    model?: string;
    stop_reason?: 'end_turn' | 'max_tokens' | 'stop_sequence' | 'tool_use' | null;
    stop_sequence?: string | null;
    usage?: Usage;
}
export interface MessageParam {
    role: 'assistant' | 'user';
    content: string | ContentBlockParam[];
}
export interface BetaMessage {
    id: string;
    type: 'message';
    role: 'assistant' | 'user';
    content: BetaContentBlock[];
    model: string;
    stop_reason: 'end_turn' | 'max_tokens' | 'stop_sequence' | 'tool_use' | null;
    stop_sequence: string | null;
    usage: BetaUsage;
}
export interface BetaMessageParam {
    role: 'assistant' | 'user';
    content: string | BetaContentBlockParam[];
}
export type BetaContentBlock = ContentBlock;
export type BetaContentBlockParam = ContentBlockParam;
export interface BetaUsage {
    input_tokens: number;
    output_tokens: number;
}
export type BetaToolUnion = Record<string, unknown>;
export interface Tool {
    name: string;
    description?: string;
    input_schema: {
        type: 'object';
        properties?: Record<string, unknown>;
        required?: string[];
    };
}
export type ToolUnion = Tool;
export type Model = string;
export type StopReason = 'end_turn' | 'max_tokens' | 'stop_sequence' | 'tool_use' | null;
export interface MessageStreamEvent {
    type: string;
}
export interface MessageCreateParams {
    model: string;
    max_tokens: number;
    messages: MessageParam[];
    tools?: Tool[];
    tool_choice?: {
        type: 'auto' | 'any' | 'tool';
        name?: string;
    };
    system?: string;
    temperature?: number;
    top_p?: number;
    top_k?: number;
    stop_sequences?: string[];
    stream?: boolean;
}
export interface Usage {
    input_tokens: number;
    output_tokens: number;
}
export interface Stream<T> {
    [Symbol.asyncIterator](): AsyncIterator<T>;
}
export declare class APIError extends Error {
    readonly status: number | undefined;
    readonly headers: Record<string, string> | undefined;
    readonly error: Record<string, unknown> | undefined;
    constructor(status: number | undefined, error: Record<string, unknown> | undefined, message: string | undefined, headers: Record<string, string> | undefined);
}
export declare class APIConnectionError extends APIError {
    constructor(message?: string);
}
export declare class APIConnectionTimeoutError extends APIError {
    constructor(message?: string);
}
export declare class APIUserAbortError extends APIError {
    constructor(message?: string);
}
export type MessageOrigin = 'user' | 'api' | 'tool' | 'system' | 'compact' | 'recovery';
export type MessageSource = 'user' | 'teammate' | 'system' | 'tick' | 'task';
export interface UserMessage extends Message {
    role: 'user';
}
export interface AssistantMessage extends Message {
    role: 'assistant';
    isApiErrorMessage?: boolean;
    errorDetails?: {
        actualTokens?: number;
        limitTokens?: number;
    };
}
export interface SystemMessage extends Message {
    role: 'system';
}
export declare function isTextBlock(block: unknown): block is TextBlock;
export declare function isImageBlock(block: unknown): block is ImageBlockParam;
export declare function isToolUseBlock(block: unknown): block is ToolUseBlock;
export declare function isToolResultBlock(block: unknown): block is ToolResultBlockParam;
export declare function createUserMessage(text: string): UserMessage;
export declare function createAssistantMessage(text: string): AssistantMessage;
export declare function createSystemMessage(text: string): SystemMessage;
export declare function toSDKContentBlocks(blocks: ContentBlock[]): ContentBlockParam[];
export interface ClientOptions {
    apiKey?: string;
    baseURL?: string;
    timeout?: number;
    httpAgent?: unknown;
}
export declare class GrayCode {
    constructor(options?: ClientOptions);
    messages: {
        create: (_params: MessageCreateParams) => Promise<Message>;
    };
}
//# sourceMappingURL=sdk.d.ts.map