export function isCanopyWaveConfigured(env: NodeJS.ProcessEnv): boolean {
  return !!env.CANOPYWAVE_API_KEY
}
