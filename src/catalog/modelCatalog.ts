import { mkdirSync, readFileSync, writeFileSync } from 'node:fs'
import { dirname } from 'node:path'
import type { APIProvider } from '../client/factory.js'
import { DEFAULT_OPENROUTER_OPENAI_BASE_URL } from '../config/providers.js'
import { DEFAULT_PROVIDER_CATALOGS } from './providers/index.js'
import type { ModelCatalog, ModelCatalogEntry } from './types.js'

const DEFAULT_MODEL_CATALOG: ModelCatalog = {
  updated_at: '2026-04-09T00:00:00.000Z',
  source: 'embedded',
  providers: DEFAULT_PROVIDER_CATALOGS,
}

const DEFAULT_CATALOG_URL =
  'https://raw.githubusercontent.com/aduermael/langdag/main/internal/models/catalog.json'

function asNumber(value: unknown): number | undefined {
  if (typeof value === 'number' && Number.isFinite(value)) return value
  if (typeof value === 'string' && value.trim()) {
    const parsed = Number(value)
    if (Number.isFinite(parsed)) return parsed
  }
  return undefined
}

type OpenRouterModelPayload = {
  id?: string
  context_length?: number
  top_provider?: {
    context_length?: number
    max_completion_tokens?: number
  }
  pricing?: {
    prompt?: string | number
    completion?: string | number
  }
}

async function fetchOpenRouterCatalog(
  env: NodeJS.ProcessEnv = process.env,
): Promise<ModelCatalogEntry[] | null> {
  const apiKey = env.OPENROUTER_API_KEY?.trim()
  if (!apiKey) return null

  const baseUrl = (env.OPENROUTER_BASE_URL?.trim() || DEFAULT_OPENROUTER_OPENAI_BASE_URL).replace(/\/+$/, '')
  const res = await fetch(`${baseUrl}/models`, {
    headers: {
      Authorization: `Bearer ${apiKey}`,
      Accept: 'application/json',
      'User-Agent': 'eyrie-model-catalog/1.0',
    },
  })
  if (!res.ok) {
    throw new Error(`openrouter model fetch failed (${res.status})`)
  }

  const payload = (await res.json()) as { data?: OpenRouterModelPayload[] } | unknown
  if (!payload || typeof payload !== 'object' || !Array.isArray((payload as { data?: unknown }).data)) {
    return null
  }

  const entries: ModelCatalogEntry[] = []
  for (const raw of (payload as { data: OpenRouterModelPayload[] }).data) {
    const id = typeof raw.id === 'string' ? raw.id.trim() : ''
    if (!id) continue
    const contextWindow =
      asNumber(raw.context_length) ??
      asNumber(raw.top_provider?.context_length) ??
      128000
    const maxOutput = asNumber(raw.top_provider?.max_completion_tokens) ?? 16384
    const inputPrice = asNumber(raw.pricing?.prompt) ?? 0
    const outputPrice = asNumber(raw.pricing?.completion) ?? 0
    entries.push({
      id,
      input_price_per_1m: inputPrice * 1_000_000,
      output_price_per_1m: outputPrice * 1_000_000,
      context_window: contextWindow,
      max_output: maxOutput,
    })
  }

  return entries.length > 0 ? entries : null
}

function isCatalog(value: unknown): value is ModelCatalog {
  if (!value || typeof value !== 'object') return false
  const v = value as Partial<ModelCatalog>
  return !!v.providers && typeof v.providers === 'object'
}

export function defaultModelCatalog(): ModelCatalog {
  return DEFAULT_MODEL_CATALOG
}

export function loadModelCatalogSync(cachePath?: string): ModelCatalog {
  if (cachePath) {
    try {
      const parsed = JSON.parse(readFileSync(cachePath, 'utf8')) as unknown
      if (isCatalog(parsed)) {
        return parsed
      }
    } catch {
      // Fall back to embedded default catalog.
    }
  }
  return defaultModelCatalog()
}

export async function fetchModelCatalog(
  cachePath?: string,
  sourceUrl = DEFAULT_CATALOG_URL,
): Promise<ModelCatalog> {
  const res = await fetch(sourceUrl, {
    headers: {
      'User-Agent': 'eyrie-model-catalog/1.0',
      Accept: 'application/json',
    },
  })
  if (!res.ok) {
    throw new Error(`model catalog fetch failed (${res.status})`)
  }
  const parsed = (await res.json()) as unknown
  if (!isCatalog(parsed)) {
    throw new Error('model catalog payload is invalid')
  }
  const normalized: ModelCatalog = {
    ...parsed,
    providers: {
      ...DEFAULT_MODEL_CATALOG.providers,
      ...parsed.providers,
    },
  }

  try {
    const openrouterModels = await fetchOpenRouterCatalog()
    if (openrouterModels) {
      normalized.providers.openrouter = openrouterModels
    }
  } catch {
    // Keep default or fetched catalog entries on OpenRouter fetch failure.
  }

  if (cachePath) {
    mkdirSync(dirname(cachePath), { recursive: true })
    writeFileSync(cachePath, `${JSON.stringify(normalized, null, 2)}\n`, 'utf8')
  }

  return normalized
}

export function modelsForProvider(
  catalog: ModelCatalog | null | undefined,
  provider: APIProvider,
): ModelCatalogEntry[] {
  if (!catalog?.providers?.[provider]) return []
  return catalog.providers[provider] ?? []
}

export type { ModelCatalog, ModelCatalogEntry } from './types.js'
