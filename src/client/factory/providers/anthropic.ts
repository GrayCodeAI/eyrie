export function isAnthropicConfigured(env: NodeJS.ProcessEnv): boolean {
  return !!env.ANTHROPIC_API_KEY
}
