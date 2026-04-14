import { anthropicNameToCanonical } from './modelNames.js';
/**
 * Deprecated models and their per-provider retirement dates.
 * Keys are canonical substrings matched against normalized model IDs.
 */
export const DEPRECATED_MODELS = {
    'claude-3-opus': {
        modelName: 'Claude 3 Opus',
        retirementDates: {
            anthropic: 'January 5, 2026',
            openai: null,
            canopywave: null,
            openrouter: null,
            grok: null,
            gemini: null,
            ollama: null,
            opencodego: null,
        },
    },
    'claude-3-7-sonnet': {
        modelName: 'Claude 3.7 Sonnet',
        retirementDates: {
            anthropic: 'February 19, 2026',
            openai: null,
            canopywave: null,
            openrouter: null,
            grok: null,
            gemini: null,
            ollama: null,
            opencodego: null,
        },
    },
    'claude-3-5-haiku': {
        modelName: 'Claude 3.5 Haiku',
        retirementDates: {
            anthropic: 'February 19, 2026',
            openai: null,
            canopywave: null,
            openrouter: null,
            grok: null,
            gemini: null,
            ollama: null,
            opencodego: null,
        },
    },
};
/**
 * Returns a deprecation warning string for a model/provider pair, or null if
 * the model is not deprecated for that provider.
 *
 * @param modelId - Full model ID (any format; normalized internally)
 * @param provider - The active API provider
 */
export function getModelDeprecationWarning(modelId, provider) {
    const canonical = anthropicNameToCanonical(modelId.toLowerCase());
    for (const [key, entry] of Object.entries(DEPRECATED_MODELS)) {
        const retirementDate = entry.retirementDates[provider];
        if (!canonical.includes(key) || !retirementDate)
            continue;
        return `⚠ ${entry.modelName} will be retired on ${retirementDate}. Consider switching to a newer model.`;
    }
    return null;
}
