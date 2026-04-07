/**
 * Base types for eyrie (core LLM library)
 * 
 * These are the fundamental types used across the system.
 * Hawk-specific extensions are kept in the hawk repo.
 */

// ID types
export type { SessionId, AgentId } from './ids.js'

// Connector types
export type {
  ConnectorTextBlock,
  ConnectorTextDelta,
} from './connector.js'
