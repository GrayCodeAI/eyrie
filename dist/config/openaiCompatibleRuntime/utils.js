export function asTrimmedString(value) {
    return typeof value === 'string' && value.trim() ? value.trim() : undefined;
}
export function firstEnvValue(env, keys) {
    for (const key of keys) {
        const value = asTrimmedString(env[key]);
        if (value)
            return value;
    }
    return undefined;
}
