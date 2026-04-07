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
export declare const EYRIE_VERSION = "0.1.0";
//# sourceMappingURL=index.d.ts.map