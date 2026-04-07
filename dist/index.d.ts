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
export { API_IMAGE_MAX_BASE64_SIZE, IMAGE_TARGET_RAW_SIZE, IMAGE_MAX_WIDTH, IMAGE_MAX_HEIGHT, PDF_TARGET_RAW_SIZE, API_PDF_MAX_PAGES, PDF_EXTRACT_SIZE_THRESHOLD, PDF_MAX_EXTRACT_SIZE, PDF_MAX_PAGES_PER_READ, PDF_AT_MENTION_INLINE_THRESHOLD, API_MAX_MEDIA_PER_REQUEST, } from './constants/limits.js';
export type { SessionId, AgentId, } from './types/ids.js';
export { asSessionId, asAgentId, toAgentId, } from './types/ids.js';
export type { ConnectorTextBlock, ConnectorTextDelta, } from './types/connector.js';
export { isConnectorTextBlock, } from './types/connector.js';
export { DEFAULT_OPENAI_BASE_URL, DEFAULT_CODEX_BASE_URL, isLocalProviderUrl, isCodexBaseUrl, resolveProviderRequest, resolveCodexAuthPath, parseChatgptAccountId, resolveCodexApiCredentials, } from './config/providers.js';
export type { ProviderTransport, ResolvedProviderRequest, ResolvedCodexCredentials, } from './config/providers.js';
export { API_ERROR_MESSAGE_PREFIX, startsWithApiErrorPrefix, PROMPT_TOO_LONG_ERROR_MESSAGE, parsePromptTooLongTokenCounts, CREDIT_BALANCE_TOO_LOW_ERROR_MESSAGE, INVALID_API_KEY_ERROR_MESSAGE, INVALID_API_KEY_ERROR_MESSAGE_EXTERNAL, ORG_DISABLED_ERROR_MESSAGE_ENV_KEY_WITH_OAUTH, ORG_DISABLED_ERROR_MESSAGE_ENV_KEY, TOKEN_REVOKED_ERROR_MESSAGE, CCR_AUTH_ERROR_MESSAGE, REPEATED_529_ERROR_MESSAGE, CUSTOM_OFF_SWITCH_MESSAGE, API_TIMEOUT_ERROR_MESSAGE, isMediaSizeError, getPdfTooLargeErrorMessage, getPdfPasswordProtectedErrorMessage, getPdfInvalidErrorMessage, getImageTooLargeErrorMessage, getRequestTooLargeErrorMessage, OAUTH_ORG_NOT_ALLOWED_ERROR_MESSAGE, getTokenRevokedErrorMessage, getOauthOrgNotAllowedErrorMessage, } from './errors/index.js';
export { EMPTY_USAGE, type NonNullableUsage, type ServerToolUse, type CacheCreation, } from './types/usage.js';
export { extractConnectionErrorDetails, getSSLErrorHint, sanitizeAPIError, type ConnectionErrorDetails, } from './utils/errorUtils.js';
export type { Message, UserMessage, AssistantMessage, SystemMessage, ContentBlock, TextBlock, ImageBlock, ToolUseBlock, ToolResultBlock, MessageOrigin, MessageSource, } from './types/message.js';
export { isTextBlock, isImageBlock, isToolUseBlock, isToolResultBlock, toSDKContentBlocks, createUserMessage, createAssistantMessage, createSystemMessage, } from './types/message.js';
export * from './sdk/index.js';
export declare const EYRIE_VERSION = "0.3.0";
//# sourceMappingURL=index.d.ts.map