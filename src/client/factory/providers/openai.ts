export function isOpenAIConfigured(env: NodeJS.ProcessEnv): boolean {
  return !!env.OPENAI_API_KEY
}
