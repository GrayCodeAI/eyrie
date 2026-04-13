export function isGrokConfigured(env) {
    return !!(env.GROK_API_KEY || env.XAI_API_KEY);
}
