import { mkdirSync, readFileSync, writeFileSync } from 'node:fs'
import { dirname } from 'node:path'
import type { APIProvider } from '../client/factory.js'
import { DEFAULT_PROVIDER_CATALOGS } from './providers/index.js'
import type { ModelCatalog, ModelCatalogEntry } from './types.js'

const DEFAULT_MODEL_CATALOG: ModelCatalog = {
  updated_at: '2026-04-09T00:00:00.000Z',
  source: 'embedded',
  providers: DEFAULT_PROVIDER_CATALOGS,
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

export type { ModelCatalog, ModelCatalogEntry } from './types.js'
