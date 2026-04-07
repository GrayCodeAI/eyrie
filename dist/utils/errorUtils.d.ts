/**
 * API Error Utilities
 *
 * Helper functions for working with API errors across different providers.
 */
import type { APIError } from '@graycode-ai/sdk';
export type ConnectionErrorDetails = {
    code: string;
    message: string;
    isSSLError: boolean;
};
/**
 * Extracts connection error details from the error cause chain.
 * The GrayCode SDK wraps underlying errors in the `cause` property.
 * This function walks the cause chain to find the root error code/message.
 */
export declare function extractConnectionErrorDetails(error: unknown): ConnectionErrorDetails | null;
/**
 * Returns an actionable hint for SSL/TLS errors, intended for contexts outside
 * the main API client (OAuth token exchange, preflight connectivity checks)
 * where `formatAPIError` doesn't apply.
 *
 * Motivation: enterprise users behind TLS-intercepting proxies (Zscaler et al.)
 * see OAuth complete in-browser but the CLI's token exchange silently fails
 * with a raw SSL code. Surfacing the likely fix saves a support round-trip.
 */
export declare function getSSLErrorHint(error: unknown): string | null;
/**
 * Detects if an error message contains HTML content (e.g., CloudFlare error pages)
 * and returns a user-friendly message instead
 */
export declare function sanitizeAPIError(apiError: APIError): string;
//# sourceMappingURL=errorUtils.d.ts.map