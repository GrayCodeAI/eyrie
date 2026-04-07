/**
 * Usage tracking types
 *
 * Types for tracking API usage metrics across providers.
 */
/**
 * Zero-initialized usage object.
 * Useful for starting fresh tracking or resetting counters.
 */
export const EMPTY_USAGE = {
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
};
