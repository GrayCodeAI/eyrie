/**
 * API Error Utilities
 *
 * Helper functions for working with API errors across different providers.
 */
// SSL/TLS error codes from OpenSSL (used by both Node.js and Bun)
// See: https://www.openssl.org/docs/man3.1/man3/X509_STORE_CTX_get_error.html
const SSL_ERROR_CODES = new Set([
    // Certificate verification errors
    'UNABLE_TO_VERIFY_LEAF_SIGNATURE',
    'UNABLE_TO_GET_ISSUER_CERT',
    'UNABLE_TO_GET_ISSUER_CERT_LOCALLY',
    'CERT_SIGNATURE_FAILURE',
    'CERT_NOT_YET_VALID',
    'CERT_HAS_EXPIRED',
    'CERT_REVOKED',
    'CERT_REJECTED',
    'CERT_UNTRUSTED',
    // Self-signed certificate errors
    'DEPTH_ZERO_SELF_SIGNED_CERT',
    'SELF_SIGNED_CERT_IN_CHAIN',
    // Chain errors
    'CERT_CHAIN_TOO_LONG',
    'PATH_LENGTH_EXCEEDED',
    // Hostname/altname errors
    'ERR_TLS_CERT_ALTNAME_INVALID',
    'HOSTNAME_MISMATCH',
    // TLS handshake errors
    'ERR_TLS_HANDSHAKE_TIMEOUT',
    'ERR_SSL_WRONG_VERSION_NUMBER',
    'ERR_SSL_DECRYPTION_FAILED_OR_BAD_RECORD_MAC',
]);
/**
 * Extracts connection error details from the error cause chain.
 * The GrayCode SDK wraps underlying errors in the `cause` property.
 * This function walks the cause chain to find the root error code/message.
 */
export function extractConnectionErrorDetails(error) {
    if (!error || typeof error !== 'object') {
        return null;
    }
    // Walk the cause chain to find the root error with a code
    let current = error;
    const maxDepth = 5; // Prevent infinite loops
    let depth = 0;
    while (current && depth < maxDepth) {
        if (current instanceof Error &&
            'code' in current &&
            typeof current.code === 'string') {
            const code = current.code;
            const isSSLError = SSL_ERROR_CODES.has(code);
            return {
                code,
                message: current.message,
                isSSLError,
            };
        }
        // Move to the next cause in the chain
        if (current instanceof Error &&
            'cause' in current &&
            current.cause !== current) {
            current = current.cause;
            depth++;
        }
        else {
            break;
        }
    }
    return null;
}
/**
 * Returns an actionable hint for SSL/TLS errors, intended for contexts outside
 * the main API client (OAuth token exchange, preflight connectivity checks)
 * where `formatAPIError` doesn't apply.
 *
 * Motivation: enterprise users behind TLS-intercepting proxies (Zscaler et al.)
 * see OAuth complete in-browser but the CLI's token exchange silently fails
 * with a raw SSL code. Surfacing the likely fix saves a support round-trip.
 */
export function getSSLErrorHint(error) {
    const details = extractConnectionErrorDetails(error);
    if (!details?.isSSLError) {
        return null;
    }
    return `SSL certificate error (${details.code}). If you are behind a corporate proxy or TLS-intercepting firewall, set NODE_EXTRA_CA_CERTS to your CA bundle path, or ask IT to allowlist *.graycode.com. Run /doctor for details.`;
}
/**
 * Strips HTML content (e.g., CloudFlare error pages) from a message string,
 * returning a user-friendly title or empty string if HTML is detected.
 * Returns the original message unchanged if no HTML is found.
 */
function sanitizeMessageHTML(message) {
    if (message.includes('<!DOCTYPE html') || message.includes('<html')) {
        const titleMatch = message.match(/<title>([^<]+)<\/title>/);
        if (titleMatch && titleMatch[1]) {
            return titleMatch[1].trim();
        }
        return '';
    }
    return message;
}
/**
 * Detects if an error message contains HTML content (e.g., CloudFlare error pages)
 * and returns a user-friendly message instead
 */
export function sanitizeAPIError(apiError) {
    const message = apiError.message;
    if (!message) {
        // Sometimes message is undefined
        // TODO: figure out why
        return '';
    }
    return sanitizeMessageHTML(message);
}
