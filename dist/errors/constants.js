/**
 * API Error message constants and utility functions
 *
 * These are safe to use in any context as they don't depend on
 * hawk-specific types or functionality.
 */
// ============================================================================
// Error Message Prefixes
// ============================================================================
/** Prefix used for all API error messages */
export const API_ERROR_MESSAGE_PREFIX = 'API Error';
/** Check if text starts with API error prefix */
export function startsWithApiErrorPrefix(text) {
    return (text.startsWith(API_ERROR_MESSAGE_PREFIX) ||
        text.startsWith(`Please run /login · ${API_ERROR_MESSAGE_PREFIX}`));
}
// ============================================================================
// Prompt Too Long Errors
// ============================================================================
/** Message indicating prompt is too long */
export const PROMPT_TOO_LONG_ERROR_MESSAGE = 'Prompt is too long';
/**
 * Parse actual/limit token counts from a raw prompt-too-long API error
 * message like "prompt is too long: 137500 tokens > 135000 maximum".
 * The raw string may be wrapped in SDK prefixes or JSON envelopes, or
 * have different casing (Vertex), so this is intentionally lenient.
 */
export function parsePromptTooLongTokenCounts(rawMessage) {
    const match = rawMessage.match(/prompt is too long[^0-9]*(\d+)\s*tokens?\s*>\s*(\d+)/i);
    return {
        actualTokens: match ? parseInt(match[1], 10) : undefined,
        limitTokens: match ? parseInt(match[2], 10) : undefined,
    };
}
// ============================================================================
// Authentication & Authorization Errors
// ============================================================================
/** Credit balance too low error */
export const CREDIT_BALANCE_TOO_LOW_ERROR_MESSAGE = 'Credit balance is too low';
/** Invalid API key error message */
export const INVALID_API_KEY_ERROR_MESSAGE = 'Not logged in · Please run /login';
/** Invalid API key error for external contexts */
export const INVALID_API_KEY_ERROR_MESSAGE_EXTERNAL = 'Invalid API key · Please check your credentials';
/** Organization disabled error (with OAuth) */
export const ORG_DISABLED_ERROR_MESSAGE_ENV_KEY_WITH_OAUTH = 'Organization disabled';
/** Organization disabled error (API key only) */
export const ORG_DISABLED_ERROR_MESSAGE_ENV_KEY = 'Organization disabled';
/** Token revoked error message */
export const TOKEN_REVOKED_ERROR_MESSAGE = 'Token has been revoked';
/** CCR (Code Completion Request) authentication error */
export const CCR_AUTH_ERROR_MESSAGE = 'Authentication error';
// ============================================================================
// Rate Limit & Overload Errors
// ============================================================================
/** Repeated 529 overloaded errors */
export const REPEATED_529_ERROR_MESSAGE = 'Repeated 529 Overloaded errors';
/** Custom off switch message */
export const CUSTOM_OFF_SWITCH_MESSAGE = 'Service temporarily disabled';
// ============================================================================
// Timeout Errors
// ============================================================================
/** API timeout error message */
export const API_TIMEOUT_ERROR_MESSAGE = 'Request timed out';
// ============================================================================
// Media Size Errors
// ============================================================================
/**
 * Check if raw error string indicates a media size error
 */
export function isMediaSizeError(raw) {
    return (raw.includes('image is too large') ||
        raw.includes('pdf is too large') ||
        raw.includes('file is too large') ||
        raw.includes('request is too large'));
}
// ============================================================================
// Error Message Getters
// ============================================================================
/**
 * Get PDF too large error message with helpful context
 */
export function getPdfTooLargeErrorMessage() {
    return 'PDF is too large · Try splitting into smaller files or reducing quality';
}
/**
 * Get PDF password protected error message
 */
export function getPdfPasswordProtectedErrorMessage() {
    return 'PDF is password protected · Please remove the password and try again';
}
/**
 * Get PDF invalid format error message
 */
export function getPdfInvalidErrorMessage() {
    return 'PDF format is invalid · Please check the file is not corrupted';
}
/**
 * Get image too large error message
 */
export function getImageTooLargeErrorMessage() {
    return 'Image is too large · Try resizing or compressing the image';
}
/**
 * Get request too large error message
 */
export function getRequestTooLargeErrorMessage() {
    return 'Request is too large · Try reducing the amount of content';
}
// ============================================================================
// OAuth Errors
// ============================================================================
/** OAuth organization not allowed error */
export const OAUTH_ORG_NOT_ALLOWED_ERROR_MESSAGE = 'Organization not allowed for OAuth';
/**
 * Get token revoked error message with instructions
 */
export function getTokenRevokedErrorMessage() {
    return 'Token has been revoked · Please run /login to reconnect';
}
/**
 * Get OAuth organization not allowed error message
 */
export function getOauthOrgNotAllowedErrorMessage() {
    return 'Organization not allowed · Please check your organization settings';
}
