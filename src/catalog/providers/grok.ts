import type { ModelCatalogEntry } from '../types.js'

export const GROK_MODELS: ModelCatalogEntry[] = [
  {
    id: 'grok-2',
    input_price_per_1m: 2,
    output_price_per_1m: 10,
    context_window: 128000,
    max_output: 8000,
    server_tools: ['web_search'],
  },
]
