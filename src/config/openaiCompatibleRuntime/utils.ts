export function asTrimmedString(value: unknown): string | undefined {
  return typeof value === 'string' && value.trim() ? value.trim() : undefined
}

export function firstEnvValue(
  env: NodeJS.ProcessEnv,
  keys: string[],
): string | undefined {
  for (const key of keys) {
    const value = asTrimmedString(env[key])
    if (value) return value
  }
  return undefined
}
