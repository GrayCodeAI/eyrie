/**
 * API Error message constants and utility functions
 *
 * These are safe to use in any context as they don't depend on
 * hawk-specific types or functionality.
 */
/** Prefix used for all API error messages */
export declare const API_ERROR_MESSAGE_PREFIX = "API Error";
/** Check if text starts with API error prefix */
export declare function startsWithApiErrorPrefix(text: string): boolean;
/** Message indicating prompt is too long */
export declare const PROMPT_TOO_LONG_ERROR_MESSAGE = "Prompt is too long";
/**
 * Parse actual/limit token counts from a raw prompt-too-long API error
 * message like "prompt is too long: 137500 tokens > 135000 maximum".
 * The raw string may be wrapped in SDK prefixes or JSON envelopes, or
 * have different casing (Vertex), so this is intentionally lenient.
 */
export declare function parsePromptTooLongTokenCounts(rawMessage: string): {
    actualTokens: number | undefined;
    limitTokens: number | undefined;
};
/** Credit balance too low error */
export declare const CREDIT_BALANCE_TOO_LOW_ERROR_MESSAGE = "Credit balance is too low";
/** Invalid API key error message */
export declare const INVALID_API_KEY_ERROR_MESSAGE = "Not logged in \u00B7 Please run /login";
/** Invalid API key error for external contexts */
export declare const INVALID_API_KEY_ERROR_MESSAGE_EXTERNAL = "Invalid API key \u00B7 Please check your credentials";
/** Organization disabled error (with OAuth) */
export declare const ORG_DISABLED_ERROR_MESSAGE_ENV_KEY_WITH_OAUTH = "Organization disabled";
/** Organization disabled error (API key only) */
export declare const ORG_DISABLED_ERROR_MESSAGE_ENV_KEY = "Organization disabled";
/** Token revoked error message */
export declare const TOKEN_REVOKED_ERROR_MESSAGE = "Token has been revoked";
/** CCR (Code Completion Request) authentication error */
export declare const CCR_AUTH_ERROR_MESSAGE = "Authentication error";
/** Repeated 529 overloaded errors */
export declare const REPEATED_529_ERROR_MESSAGE = "Repeated 529 Overloaded errors";
/** Custom off switch message */
export declare const CUSTOM_OFF_SWITCH_MESSAGE = "Service temporarily disabled";
/** API timeout error message */
export declare const API_TIMEOUT_ERROR_MESSAGE = "Request timed out";
/**
 * Check if raw error string indicates a media size error
 */
export declare function isMediaSizeError(raw: string): boolean;
/**
 * Get PDF too large error message with helpful context
 */
export declare function getPdfTooLargeErrorMessage(): string;
/**
 * Get PDF password protected error message
 */
export declare function getPdfPasswordProtectedErrorMessage(): string;
/**
 * Get PDF invalid format error message
 */
export declare function getPdfInvalidErrorMessage(): string;
/**
 * Get image too large error message
 */
export declare function getImageTooLargeErrorMessage(): string;
/**
 * Get request too large error message
 */
export declare function getRequestTooLargeErrorMessage(): string;
/** OAuth organization not allowed error */
export declare const OAUTH_ORG_NOT_ALLOWED_ERROR_MESSAGE = "Organization not allowed for OAuth";
/**
 * Get token revoked error message with instructions
 */
export declare function getTokenRevokedErrorMessage(): string;
/**
 * Get OAuth organization not allowed error message
 */
export declare function getOauthOrgNotAllowedErrorMessage(): string;
//# sourceMappingURL=constants.d.ts.map