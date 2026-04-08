import { mkdirSync, readFileSync, writeFileSync } from 'node:fs'
import { dirname } from 'node:path'
import type { APIProvider } from '../client/factory.js'

export type ModelCatalogEntry = {
  id: string
  input_price_per_1m: number
  output_price_per_1m: number
  context_window: number
  max_output: number
  server_tools?: string[]
}

export type ModelCatalog = {
  updated_at: string
  source: string
  providers: Partial<Record<APIProvider, ModelCatalogEntry[]>>
}

const DEFAULT_MODEL_CATALOG: ModelCatalog = {
  updated_at: '2026-04-09T00:00:00.000Z',
  source: 'embedded',
  providers: {
    anthropic: [
      {
        id: 'claude-opus-4-6',
        input_price_per_1m: 15,
        output_price_per_1m: 75,
        context_window: 200000,
        max_output: 32000,
        server_tools: ['web_search'],
      },
      {
        id: 'claude-sonnet-4-6',
        input_price_per_1m: 3,
        output_price_per_1m: 15,
        context_window: 200000,
        max_output: 32000,
        server_tools: ['web_search'],
      },
      {
        id: 'claude-haiku-4-5-20251001',
        input_price_per_1m: 1,
        output_price_per_1m: 5,
        context_window: 200000,
        max_output: 16000,
        server_tools: ['web_search'],
      },
    ],
    openai: [
      {
        id: 'gpt-4o',
        input_price_per_1m: 5,
        output_price_per_1m: 15,
        context_window: 128000,
        max_output: 16000,
        server_tools: ['web_search'],
      },
      {
        id: 'gpt-4o-mini',
        input_price_per_1m: 0.15,
        output_price_per_1m: 0.6,
        context_window: 128000,
        max_output: 16000,
        server_tools: ['web_search'],
      },
    ],
    grok: [
      {
        id: 'grok-2',
        input_price_per_1m: 2,
        output_price_per_1m: 10,
        context_window: 128000,
        max_output: 8000,
        server_tools: ['web_search'],
      },
    ],
    gemini: [
      {
        id: 'gemini-2.5-pro-preview-03-25',
        input_price_per_1m: 1.25,
        output_price_per_1m: 5,
        context_window: 1000000,
        max_output: 65536,
        server_tools: ['web_search'],
      },
      {
        id: 'gemini-2.0-flash',
        input_price_per_1m: 0.1,
        output_price_per_1m: 0.4,
        context_window: 1000000,
        max_output: 8192,
        server_tools: ['web_search'],
      },
      {
        id: 'gemini-2.0-flash-lite',
        input_price_per_1m: 0.075,
        output_price_per_1m: 0.3,
        context_window: 1000000,
        max_output: 8192,
        server_tools: ['web_search'],
      },
    ],
    ollama: [],
  },
}

const DEFAULT_CATALOG_URL =
  'https://raw.githubusercontent.com/aduermael/langdag/main/internal/models/catalog.json'

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

  if (cachePath) {
    mkdirSync(dirname(cachePath), { recursive: true })
    writeFileSync(cachePath, `${JSON.stringify(parsed, null, 2)}\n`, 'utf8')
  }

  return parsed
}

export function modelsForProvider(
  catalog: ModelCatalog | null | undefined,
  provider: APIProvider,
): ModelCatalogEntry[] {
  if (!catalog?.providers?.[provider]) return []
  return catalog.providers[provider] ?? []
}
