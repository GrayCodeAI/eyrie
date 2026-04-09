import type { APIProvider } from '../client/factory.js';
import type { ModelCatalog, ModelCatalogEntry } from './types.js';
export declare function defaultModelCatalog(): ModelCatalog;
export declare function loadModelCatalogSync(cachePath?: string): ModelCatalog;
export declare function fetchModelCatalog(cachePath?: string, sourceUrl?: string): Promise<ModelCatalog>;
export declare function modelsForProvider(catalog: ModelCatalog | null | undefined, provider: APIProvider): ModelCatalogEntry[];
export type { ModelCatalog, ModelCatalogEntry } from './types.js';
//# sourceMappingURL=modelCatalog.d.ts.map