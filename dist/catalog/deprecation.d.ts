/**
 * Model deprecation metadata and warning helpers.
 * All retirement dates live here — not in the consuming application.
 */
import type { APIProvider } from '../config/providerProfiles/types.js';
type DeprecationEntry = {
    /** Human-readable model name shown in warnings */
    modelName: string;
    /** Retirement dates keyed by provider; null = not deprecated for that provider */
    retirementDates: Record<APIProvider, string | null>;
};
/**
 * Deprecated models and their per-provider retirement dates.
 * Keys are canonical substrings matched against normalized model IDs.
 */
export declare const DEPRECATED_MODELS: Record<string, DeprecationEntry>;
/**
 * Returns a deprecation warning string for a model/provider pair, or null if
 * the model is not deprecated for that provider.
 *
 * @param modelId - Full model ID (any format; normalized internally)
 * @param provider - The active API provider
 */
export declare function getModelDeprecationWarning(modelId: string, provider: APIProvider): string | null;
export {};
//# sourceMappingURL=deprecation.d.ts.map