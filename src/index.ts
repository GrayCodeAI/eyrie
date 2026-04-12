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
export {
  API_IMAGE_MAX_BASE64_SIZE,
  IMAGE_TARGET_RAW_SIZE,
  IMAGE_MAX_WIDTH,
  IMAGE_MAX_HEIGHT,
  PDF_TARGET_RAW_SIZE,
  API_PDF_MAX_PAGES,
  PDF_EXTRACT_SIZE_THRESHOLD,
  PDF_MAX_EXTRACT_SIZE,
  PDF_MAX_PAGES_PER_READ,
  PDF_AT_MENTION_INLINE_THRESHOLD,
  API_MAX_MEDIA_PER_REQUEST,
} from './constants/limits.js'

// Phase 2: Types
export type {
  SessionId,
  AgentId,
} from './types/ids.js'

export {
  asSessionId,
  asAgentId,
  toAgentId,
} from './types/ids.js'

export type {
  ConnectorTextBlock,
  ConnectorTextDelta,
} from './types/connector.js'

export {
  isConnectorTextBlock,
} from './types/connector.js'

// Phase 3: Provider Config
export {
  DEFAULT_OPENAI_BASE_URL,
  DEFAULT_CANOPYWAVE_OPENAI_BASE_URL,
  DEFAULT_OPENROUTER_OPENAI_BASE_URL,
  DEFAULT_GEMINI_OPENAI_BASE_URL,
  DEFAULT_ANTHROPIC_OPENAI_BASE_URL,
  DEFAULT_GROK_OPENAI_BASE_URL,
  isLocalProviderUrl,
  resolveProviderRequest,
} from './config/providers.js'

export {
  OPENAI_COMPATIBLE_RUNTIME_PROVIDERS,
  isOpenAICompatibleRuntimeEnabled,
  resolveOpenAICompatibleRuntime,
} from './config/openaiCompatibleRuntime.js'

export type {
  ProviderTransport,
  ResolvedProviderRequest,
} from './config/providers.js'
export type { APIProvider } from './config/providerProfiles.js'

export {
  API_PROVIDER_DETECTION_ORDER,
  OPENAI_COMPATIBLE_RUNTIME_PROFILES,
  OPENAI_COMPATIBLE_RUNTIME_PROFILE_ORDER,
  PROVIDER_MODEL_ENV_KEYS,
} from './config/providerProfiles.js'

export type {
  OpenAICompatibleRuntimeProvider,
  OpenAICompatibleRuntimeMode,
  OpenAICompatibleApiKeySource,
  ResolvedOpenAICompatibleRuntime,
} from './config/openaiCompatibleRuntime.js'

// Phase 4: Error Constants
export {
  // Error prefixes
  API_ERROR_MESSAGE_PREFIX,
  startsWithApiErrorPrefix,
  
  // Prompt too long errors
  PROMPT_TOO_LONG_ERROR_MESSAGE,
  parsePromptTooLongTokenCounts,
  
  // Auth errors
  CREDIT_BALANCE_TOO_LOW_ERROR_MESSAGE,
  INVALID_API_KEY_ERROR_MESSAGE,
  INVALID_API_KEY_ERROR_MESSAGE_EXTERNAL,
  ORG_DISABLED_ERROR_MESSAGE_ENV_KEY_WITH_OAUTH,
  ORG_DISABLED_ERROR_MESSAGE_ENV_KEY,
  TOKEN_REVOKED_ERROR_MESSAGE,
  CCR_AUTH_ERROR_MESSAGE,
  
  // Rate limit errors
  REPEATED_529_ERROR_MESSAGE,
  CUSTOM_OFF_SWITCH_MESSAGE,
  
  // Timeout errors
  API_TIMEOUT_ERROR_MESSAGE,
  
  // Media size errors
  isMediaSizeError,
  getPdfTooLargeErrorMessage,
  getPdfPasswordProtectedErrorMessage,
  getPdfInvalidErrorMessage,
  getImageTooLargeErrorMessage,
  getRequestTooLargeErrorMessage,
  
  // OAuth errors
  OAUTH_ORG_NOT_ALLOWED_ERROR_MESSAGE,
  getTokenRevokedErrorMessage,
  getOauthOrgNotAllowedErrorMessage,
} from './errors/index.js'

// Future phases will add more exports here
// Phase 5: Clients

// Phase 5: API Utilities
export {
  EMPTY_USAGE,
  type NonNullableUsage,
  type ServerToolUse,
  type CacheCreation,
} from './types/usage.js'

export {
  extractConnectionErrorDetails,
  getSSLErrorHint,
  sanitizeAPIError,
  type ConnectionErrorDetails,
} from './utils/errorUtils.js'

// Phase 6 & 7: All Message and SDK Types (Independent - no npm dependency)
export type {
  // Core message types
  Message,
  // StopReason alias (backward compat with graycode-ai/sdk naming)
  StopReason as BetaStopReason,
  UserMessage,
  AssistantMessage,
  SystemMessage,
  MessageOrigin,
  MessageSource,
  // Content blocks
  ContentBlock,
  ContentBlockParam,
  TextBlock,
  TextBlockParam,
  ImageBlock,
  ImageBlockParam,
  Base64ImageSource,
  ToolUseBlock,
  ToolUseBlockParam,
  ToolResultBlock,
  ToolResultBlockParam,
  ThinkingBlock,
  ThinkingBlockParam,
  RedactedThinkingBlock,
  RedactedThinkingBlockParam,
  // Messages
  MessageParam,
  BetaMessage,
  BetaMessageParam,
  BetaContentBlock,
  BetaContentBlockParam,
  BetaMessageStreamParams,
  // Tools
  Tool,
  ToolUnion,
  BetaToolUnion,
  // Other types
  Model,
  StopReason,
  MessageStreamEvent,
  MessageCreateParams,
  Usage,
  BetaUsage,
  Stream,
  // Client
  ClientOptions,
} from './types/sdk.js'

export {
  // Type guards
  isTextBlock,
  isImageBlock,
  isToolUseBlock,
  isToolResultBlock,
  // Creators
  createUserMessage,
  createAssistantMessage,
  createSystemMessage,
  toSDKContentBlocks,
  // Errors
  APIError,
  APIConnectionError,
  APIConnectionTimeoutError,
  APIUserAbortError,
  NotFoundError,
  AuthenticationError,
  // Client (stub kept for any remaining type references during migration)
  GrayCode,
} from './types/sdk.js'

// Version
export const EYRIE_VERSION = '1.0.2'

// Client exports
export {
  EyrieClient,
  createEyrie,
  type EyrieConfig,
  type EyrieMessage,
  type EyrieTool,
  type EyrieResponse,
  type EyrieStreamEvent,
  type EyrieUsage,
} from './client/index.js'

// Provider-aware client factory (the main entry point for hawk)
export {
  createAnthropicClient,
  detectProvider,
  resolveProviderModelEnvOverride,
  parseCustomHeaders,
  type AnthropicClientConfig,
} from './client/factory.js'

export {
  defaultModelCatalog,
  loadModelCatalogSync,
  fetchModelCatalog,
  modelsForProvider,
  type ModelCatalog,
  type ModelCatalogEntry,
} from './catalog/modelCatalog.js'

export {
  CORE_PROVIDERS,
  OPENAI_COMPATIBLE_PROVIDERS,
  type ProviderConfig,
  type ProviderType,
  type OpenAICompatConfig,
} from './providers/registry.js'
