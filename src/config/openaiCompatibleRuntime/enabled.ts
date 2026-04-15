export function isOpenAICompatibleRuntimeEnabled(
  env: NodeJS.ProcessEnv = process.env,
): boolean {
  return !!(
    env.OPENROUTER_API_KEY ||
    env.GROK_API_KEY ||
    env.XAI_API_KEY ||
    env.GEMINI_API_KEY ||
    env.ANTHROPIC_API_KEY ||
    env.CANOPYWAVE_API_KEY ||
    env.OPENAI_API_KEY ||
    env.OPENCODEGO_API_KEY ||
    env.OLLAMA_BASE_URL
  )
}
