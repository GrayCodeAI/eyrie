/**
 * Model name canonicalization and marketing display names.
 * Pure functions — no state, no disk I/O.
 */
/**
 * Maps an Anthropic-format model ID (with date/region/version suffixes)
 * to its canonical short name. Input is normalized to lowercase internally.
 *
 * e.g. 'claude-sonnet-4-6-20250814'          → 'claude-sonnet-4-6'
 *      'us.graycode.claude-opus-4-6-v1:0'    → 'claude-opus-4-6'
 *      'claude-3-5-haiku-20241022'            → 'claude-3-5-haiku'
 *
 * For non-Anthropic model IDs the input is returned unchanged (lowercased).
 */
export declare function anthropicNameToCanonical(name: string): string;
/**
 * Returns the marketing display name for a model ID, or undefined if the model
 * is not a recognized public Anthropic model.
 *
 * Handles [1m] / [2m] context-window suffixes transparently.
 *
 * e.g. 'claude-opus-4-6'              → 'Opus 4.6'
 *      'claude-sonnet-4-6[1m]'        → 'Sonnet 4.6 (1M context)'
 *      'claude-3-7-sonnet-20250219'    → 'Sonnet 3.7'
 */
export declare function getModelMarketingName(modelId: string): string | undefined;
//# sourceMappingURL=modelNames.d.ts.map