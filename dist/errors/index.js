/**
 * Error constants and utilities
 *
 * Safe, dependency-free error messages and utilities for API interactions.
 */
export { 
// Error prefixes
API_ERROR_MESSAGE_PREFIX, startsWithApiErrorPrefix, 
// Prompt too long errors
PROMPT_TOO_LONG_ERROR_MESSAGE, parsePromptTooLongTokenCounts, 
// Auth errors
CREDIT_BALANCE_TOO_LOW_ERROR_MESSAGE, INVALID_API_KEY_ERROR_MESSAGE, INVALID_API_KEY_ERROR_MESSAGE_EXTERNAL, ORG_DISABLED_ERROR_MESSAGE_ENV_KEY_WITH_OAUTH, ORG_DISABLED_ERROR_MESSAGE_ENV_KEY, TOKEN_REVOKED_ERROR_MESSAGE, CCR_AUTH_ERROR_MESSAGE, 
// Rate limit errors
REPEATED_529_ERROR_MESSAGE, CUSTOM_OFF_SWITCH_MESSAGE, 
// Timeout errors
API_TIMEOUT_ERROR_MESSAGE, 
// Media size errors
isMediaSizeError, getPdfTooLargeErrorMessage, getPdfPasswordProtectedErrorMessage, getPdfInvalidErrorMessage, getImageTooLargeErrorMessage, getRequestTooLargeErrorMessage, 
// OAuth errors
OAUTH_ORG_NOT_ALLOWED_ERROR_MESSAGE, getTokenRevokedErrorMessage, getOauthOrgNotAllowedErrorMessage, } from './constants.js';
