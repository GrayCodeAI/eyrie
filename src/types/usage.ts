/**
 * Usage tracking types
 * 
 * Types for tracking API usage metrics across providers.
 */

/**
 * Server-side tool usage counters
 */
export type ServerToolUse = {
  web_search_requests: number
  web_fetch_requests: number
}

/**
 * Cache creation token breakdown
 */
export type CacheCreation = {
  ephemeral_1h_input_tokens: number
  ephemeral_5m_input_tokens: number
}

/**
 * Non-nullable usage metrics
 * 
 * All fields are required (non-nullable) for consistent tracking.
 */
export type NonNullableUsage = {
  input_tokens: number
  cache_creation_input_tokens: number
  cache_read_input_tokens: number
  output_tokens: number
  server_tool_use: ServerToolUse
  service_tier: string
  cache_creation: CacheCreation
  inference_geo: string
  iterations: unknown[]
  speed: string
}

/**
 * Zero-initialized usage object.
 * Useful for starting fresh tracking or resetting counters.
 */
export const EMPTY_USAGE: Readonly<NonNullableUsage> = {
  input_tokens: 0,
  cache_creation_input_tokens: 0,
  cache_read_input_tokens: 0,
  output_tokens: 0,
  server_tool_use: { web_search_requests: 0, web_fetch_requests: 0 },
  service_tier: 'standard',
  cache_creation: {
    ephemeral_1h_input_tokens: 0,
    ephemeral_5m_input_tokens: 0,
  },
  inference_geo: '',
  iterations: [],
  speed: 'standard',
}
