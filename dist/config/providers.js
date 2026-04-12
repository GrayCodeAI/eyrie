export const DEFAULT_OPENAI_BASE_URL = 'https://api.openai.com/v1';
export const DEFAULT_OPENROUTER_OPENAI_BASE_URL = 'https://openrouter.ai/api/v1';
export const DEFAULT_CANOPYWAVE_OPENAI_BASE_URL = 'https://inference.canopywave.io/v1';
export const DEFAULT_GEMINI_OPENAI_BASE_URL = 'https://generativelanguage.googleapis.com/v1beta/openai';
export const DEFAULT_ANTHROPIC_OPENAI_BASE_URL = 'https://api.anthropic.com/v1';
export const DEFAULT_GROK_OPENAI_BASE_URL = 'https://api.x.ai/v1';
const LOCALHOST_HOSTNAMES = new Set(['localhost', '127.0.0.1', '::1']);
function asTrimmedString(value) {
    return typeof value === 'string' && value.trim() ? value.trim() : undefined;
}
function parseReasoningEffort(value) {
    if (!value)
        return undefined;
    const normalized = value.trim().toLowerCase();
    if (normalized === 'low' || normalized === 'medium' || normalized === 'high') {
        return normalized;
    }
    return undefined;
}
function parseModelDescriptor(model) {
    const trimmed = model.trim();
    const queryIndex = trimmed.indexOf('?');
    if (queryIndex === -1) {
        return {
            raw: trimmed,
            baseModel: trimmed,
        };
    }
    const baseModel = trimmed.slice(0, queryIndex).trim();
    const params = new URLSearchParams(trimmed.slice(queryIndex + 1));
    const reasoning = parseReasoningEffort(params.get('reasoning') ?? undefined);
    return {
        raw: trimmed,
        baseModel,
        reasoning: typeof reasoning === 'string' ? { effort: reasoning } : reasoning,
    };
}
export function isLocalProviderUrl(baseUrl) {
    if (!baseUrl)
        return false;
    try {
        return LOCALHOST_HOSTNAMES.has(new URL(baseUrl).hostname);
    }
    catch {
        return false;
    }
}
export function resolveProviderRequest(options) {
    const requestedModel = options?.model?.trim() ||
        process.env.OPENAI_MODEL?.trim() ||
        options?.fallbackModel?.trim() ||
        'gpt-4o';
    const descriptor = parseModelDescriptor(requestedModel);
    const rawBaseUrl = options?.baseUrl ??
        process.env.OPENAI_BASE_URL ??
        process.env.OPENAI_API_BASE ??
        undefined;
    const transport = 'chat_completions';
    return {
        transport,
        requestedModel,
        resolvedModel: descriptor.baseModel,
        baseUrl: (rawBaseUrl ?? DEFAULT_OPENAI_BASE_URL).replace(/\/+$/, ''),
        reasoning: descriptor.reasoning,
    };
}
