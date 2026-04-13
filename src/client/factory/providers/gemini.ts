export function isGeminiConfigured(env: NodeJS.ProcessEnv): boolean {
  return !!env.GEMINI_API_KEY
}
