import { mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { dirname } from 'node:path';
import { DEFAULT_PROVIDER_CATALOGS } from './providers/index.js';
const DEFAULT_MODEL_CATALOG = {
    updated_at: '2026-04-09T00:00:00.000Z',
    source: 'embedded',
    providers: DEFAULT_PROVIDER_CATALOGS,
};
const DEFAULT_CATALOG_URL = 'https://raw.githubusercontent.com/aduermael/langdag/main/internal/models/catalog.json';
function isCatalog(value) {
    if (!value || typeof value !== 'object')
        return false;
    const v = value;
    return !!v.providers && typeof v.providers === 'object';
}
export function defaultModelCatalog() {
    return DEFAULT_MODEL_CATALOG;
}
export function loadModelCatalogSync(cachePath) {
    if (cachePath) {
        try {
            const parsed = JSON.parse(readFileSync(cachePath, 'utf8'));
            if (isCatalog(parsed)) {
                return parsed;
            }
        }
        catch {
            // Fall back to embedded default catalog.
        }
    }
    return defaultModelCatalog();
}
export async function fetchModelCatalog(cachePath, sourceUrl = DEFAULT_CATALOG_URL) {
    const res = await fetch(sourceUrl, {
        headers: {
            'User-Agent': 'eyrie-model-catalog/1.0',
            Accept: 'application/json',
        },
    });
    if (!res.ok) {
        throw new Error(`model catalog fetch failed (${res.status})`);
    }
    const parsed = (await res.json());
    if (!isCatalog(parsed)) {
        throw new Error('model catalog payload is invalid');
    }
    const normalized = {
        ...parsed,
        providers: {
            ...DEFAULT_MODEL_CATALOG.providers,
            ...parsed.providers,
        },
    };
    if (cachePath) {
        mkdirSync(dirname(cachePath), { recursive: true });
        writeFileSync(cachePath, `${JSON.stringify(normalized, null, 2)}\n`, 'utf8');
    }
    return normalized;
}
export function modelsForProvider(catalog, provider) {
    if (!catalog?.providers?.[provider])
        return [];
    return catalog.providers[provider] ?? [];
}
