export function isGrokConfigured(env: NodeJS.ProcessEnv): boolean {
  return !!(env.GROK_API_KEY || env.XAI_API_KEY)
}
