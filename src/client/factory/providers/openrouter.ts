export function isOpenRouterConfigured(env: NodeJS.ProcessEnv): boolean {
  return !!env.OPENROUTER_API_KEY
}
