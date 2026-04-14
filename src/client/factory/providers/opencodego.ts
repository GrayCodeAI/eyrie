export function isOpenCodeGoConfigured(env: NodeJS.ProcessEnv): boolean {
  return !!env.OPENCODEGO_API_KEY
}
