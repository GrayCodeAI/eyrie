/**
 * eyrie - Core LLM client library for hawk
 *
 * This is the core library that handles:
 * - API provider configurations
 * - Model resolution
 * - API limits and constants
 * - Base types (messages, IDs, connectors)
 * - Error types
 *
 * @module @hawk/eyrie
 */
// Phase 1: Constants
export { API_IMAGE_MAX_BASE64_SIZE, IMAGE_TARGET_RAW_SIZE, IMAGE_MAX_WIDTH, IMAGE_MAX_HEIGHT, PDF_TARGET_RAW_SIZE, API_PDF_MAX_PAGES, PDF_EXTRACT_SIZE_THRESHOLD, PDF_MAX_EXTRACT_SIZE, PDF_MAX_PAGES_PER_READ, PDF_AT_MENTION_INLINE_THRESHOLD, API_MAX_MEDIA_PER_REQUEST, } from './constants/limits.js';
export { asSessionId, asAgentId, toAgentId, } from './types/ids.js';
export { isConnectorTextBlock, } from './types/connector.js';
// Phase 3: Provider Config
export { DEFAULT_OPENAI_BASE_URL, DEFAULT_CODEX_BASE_URL, isLocalProviderUrl, isCodexBaseUrl, resolveProviderRequest, resolveCodexAuthPath, parseChatgptAccountId, resolveCodexApiCredentials, } from './config/providers.js';
// Phase 4: Error Constants
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
OAUTH_ORG_NOT_ALLOWED_ERROR_MESSAGE, getTokenRevokedErrorMessage, getOauthOrgNotAllowedErrorMessage, } from './errors/index.js';
// Future phases will add more exports here
// Phase 5: Clients
// Phase 5: API Utilities
export { EMPTY_USAGE, } from './types/usage.js';
export { extractConnectionErrorDetails, getSSLErrorHint, sanitizeAPIError, } from './utils/errorUtils.js';
export { isTextBlock, isImageBlock, isToolUseBlock, isToolResultBlock, toSDKContentBlocks, createUserMessage, createAssistantMessage, createSystemMessage, } from './types/message.js';
// Phase 7: SDK Re-exports
// Re-export GrayCode SDK types so hawk doesn't need direct npm dependency
export * from './sdk/index.js';
// Version
export const EYRIE_VERSION = '0.3.0';
