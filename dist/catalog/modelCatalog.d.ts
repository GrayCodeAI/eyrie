import type { APIProvider } from '../client/factory.js';
export type ModelCatalogEntry = {
    id: string;
    input_price_per_1m: number;
    output_price_per_1m: number;
    context_window: number;
    max_output: number;
    server_tools?: string[];
};
export type ModelCatalog = {
    updated_at: string;
    source: string;
    providers: Partial<Record<APIProvider, ModelCatalogEntry[]>>;
};
export declare function defaultModelCatalog(): ModelCatalog;
export declare function loadModelCatalogSync(cachePath?: string): ModelCatalog;
export declare function fetchModelCatalog(cachePath?: string, sourceUrl?: string): Promise<ModelCatalog>;
export declare function modelsForProvider(catalog: ModelCatalog | null | undefined, provider: APIProvider): ModelCatalogEntry[];
//# sourceMappingURL=modelCatalog.d.ts.map