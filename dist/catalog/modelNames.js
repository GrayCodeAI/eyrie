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
export function anthropicNameToCanonical(name) {
    name = name.toLowerCase();
    // Claude 4+ — check more specific versions first (4-5 before 4, etc.)
    if (name.includes('claude-opus-4-6'))
        return 'claude-opus-4-6';
    if (name.includes('claude-opus-4-5'))
        return 'claude-opus-4-5';
    if (name.includes('claude-opus-4-1'))
        return 'claude-opus-4-1';
    if (name.includes('claude-opus-4'))
        return 'claude-opus-4';
    if (name.includes('claude-sonnet-4-6'))
        return 'claude-sonnet-4-6';
    if (name.includes('claude-sonnet-4-5'))
        return 'claude-sonnet-4-5';
    if (name.includes('claude-sonnet-4'))
        return 'claude-sonnet-4';
    if (name.includes('claude-haiku-4-5'))
        return 'claude-haiku-4-5';
    // Claude 3.x
    if (name.includes('claude-3-7-sonnet'))
        return 'claude-3-7-sonnet';
    if (name.includes('claude-3-5-sonnet'))
        return 'claude-3-5-sonnet';
    if (name.includes('claude-3-5-haiku'))
        return 'claude-3-5-haiku';
    if (name.includes('claude-3-opus'))
        return 'claude-3-opus';
    if (name.includes('claude-3-sonnet'))
        return 'claude-3-sonnet';
    if (name.includes('claude-3-haiku'))
        return 'claude-3-haiku';
    const match = name.match(/(claude-(\d+-\d+-)?\w+)/);
    if (match?.[1])
        return match[1];
    return name;
}
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
export function getModelMarketingName(modelId) {
    const lower = modelId.toLowerCase();
    const has1m = lower.includes('[1m]');
    const base = lower.replace(/\[(1|2)m\]/gi, '').trim();
    const canonical = anthropicNameToCanonical(base);
    if (canonical.includes('claude-opus-4-6'))
        return has1m ? 'Opus 4.6 (1M context)' : 'Opus 4.6';
    if (canonical.includes('claude-opus-4-5'))
        return 'Opus 4.5';
    if (canonical.includes('claude-opus-4-1'))
        return 'Opus 4.1';
    if (canonical.includes('claude-opus-4'))
        return 'Opus 4';
    if (canonical.includes('claude-sonnet-4-6'))
        return has1m ? 'Sonnet 4.6 (1M context)' : 'Sonnet 4.6';
    if (canonical.includes('claude-sonnet-4-5'))
        return has1m ? 'Sonnet 4.5 (1M context)' : 'Sonnet 4.5';
    if (canonical.includes('claude-sonnet-4'))
        return has1m ? 'Sonnet 4 (1M context)' : 'Sonnet 4';
    if (canonical.includes('claude-3-7-sonnet'))
        return 'Sonnet 3.7';
    if (canonical.includes('claude-3-5-sonnet'))
        return 'Sonnet 3.5';
    if (canonical.includes('claude-haiku-4-5'))
        return 'Haiku 4.5';
    if (canonical.includes('claude-3-5-haiku'))
        return 'Haiku 3.5';
    return undefined;
}
