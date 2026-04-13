export function isOllamaConfigured(env: NodeJS.ProcessEnv): boolean {
  return !!env.OLLAMA_BASE_URL
}
