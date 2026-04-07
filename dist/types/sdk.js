/**
 * SDK Type Definitions
 *
 * These types mirror the @graycode-ai/sdk types but are defined locally
 * to make eyrie completely independent of npm dependencies.
 */
// ============================================================================
// API Errors
// ============================================================================
export class APIError extends Error {
    status;
    headers;
    error;
    constructor(status, error, message, headers) {
        super(message || 'API Error');
        this.status = status;
        this.error = error;
        this.headers = headers;
    }
}
export class APIConnectionError extends APIError {
    constructor(message) {
        super(undefined, undefined, message || 'Connection error', undefined);
    }
}
export class APIConnectionTimeoutError extends APIError {
    constructor(message) {
        super(undefined, undefined, message || 'Connection timeout', undefined);
    }
}
export class APIUserAbortError extends APIError {
    constructor(message) {
        super(undefined, undefined, message, undefined);
    }
}
export class NotFoundError extends APIError {
    constructor(message) {
        super(undefined, undefined, message, undefined);
    }
}
export class AuthenticationError extends APIError {
    constructor(message) {
        super(undefined, undefined, message, undefined);
    }
}
// ============================================================================
// Type Guards
// ============================================================================
export function isTextBlock(block) {
    return (typeof block === 'object' &&
        block !== null &&
        'type' in block &&
        block.type === 'text' &&
        'text' in block &&
        typeof block.text === 'string');
}
export function isImageBlock(block) {
    return (typeof block === 'object' &&
        block !== null &&
        'type' in block &&
        block.type === 'image' &&
        'source' in block);
}
export function isToolUseBlock(block) {
    return (typeof block === 'object' &&
        block !== null &&
        'type' in block &&
        block.type === 'tool_use' &&
        'id' in block &&
        'name' in block);
}
export function isToolResultBlock(block) {
    return (typeof block === 'object' &&
        block !== null &&
        'type' in block &&
        block.type === 'tool_result' &&
        'tool_use_id' in block);
}
// ============================================================================
// Creator Functions
// ============================================================================
export function createUserMessage(text) {
    return {
        role: 'user',
        content: text,
    };
}
export function createAssistantMessage(text) {
    return {
        role: 'assistant',
        content: text,
    };
}
export function createSystemMessage(text) {
    return {
        role: 'system',
        content: text,
    };
}
export function toSDKContentBlocks(blocks) {
    return blocks.map(block => {
        if (block.type === 'text') {
            return { type: 'text', text: block.text };
        }
        throw new Error(`Block type conversion not implemented: ${block.type}`);
    });
}
export class GrayCode {
    constructor(options) { }
    messages = {
        create: async (_params) => {
            throw new Error('GrayCode client is a type stub. Use actual SDK for runtime.');
        }
    };
}
