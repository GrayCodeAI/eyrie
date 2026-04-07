/**
 * Core message types for LLM conversations
 *
 * Base types used across the system for message handling.
 * These are intentionally minimal and can be extended in hawk.
 */
// ============================================================================
// Utility Types
// ============================================================================
/**
 * Type guard for text blocks
 */
export function isTextBlock(block) {
    return (typeof block === 'object' &&
        block !== null &&
        'type' in block &&
        block.type === 'text' &&
        'text' in block &&
        typeof block.text === 'string');
}
/**
 * Type guard for image blocks
 */
export function isImageBlock(block) {
    return (typeof block === 'object' &&
        block !== null &&
        'type' in block &&
        block.type === 'image' &&
        'source' in block);
}
/**
 * Type guard for tool use blocks
 */
export function isToolUseBlock(block) {
    return (typeof block === 'object' &&
        block !== null &&
        'type' in block &&
        block.type === 'tool_use' &&
        'id' in block &&
        'name' in block);
}
/**
 * Type guard for tool result blocks
 */
export function isToolResultBlock(block) {
    return (typeof block === 'object' &&
        block !== null &&
        'type' in block &&
        block.type === 'tool_result' &&
        'tool_use_id' in block);
}
// ============================================================================
// Conversion Utilities
// ============================================================================
/**
 * Convert content blocks to SDK format
 */
export function toSDKContentBlocks(blocks) {
    return blocks.map(block => {
        if (block.type === 'text') {
            return { type: 'text', text: block.text };
        }
        if (block.type === 'image') {
            return {
                type: 'image',
                source: block.source,
            };
        }
        if (block.type === 'tool_use') {
            return {
                type: 'tool_use',
                id: block.id,
                name: block.name,
                input: block.input,
            };
        }
        if (block.type === 'tool_result') {
            return {
                type: 'tool_result',
                tool_use_id: block.tool_use_id,
                content: block.content,
                is_error: block.is_error,
            };
        }
        throw new Error(`Unknown block type: ${block.type}`);
    });
}
/**
 * Create a user message with text content
 */
export function createUserMessage(text) {
    return {
        role: 'user',
        content: text,
    };
}
/**
 * Create an assistant message with text content
 */
export function createAssistantMessage(text) {
    return {
        role: 'assistant',
        content: text,
    };
}
/**
 * Create a system message
 */
export function createSystemMessage(text) {
    return {
        role: 'system',
        content: text,
    };
}
