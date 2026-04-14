import type { ModelCatalogEntry } from '../types.js'

export const OPENCODEGO_MODELS: ModelCatalogEntry[] = [
  {
    id: 'opencode-go',
    input_price_per_1m: 2,
    output_price_per_1m: 8,
    context_window: 128000,
    max_output: 8000,
    server_tools: ['web_search'],
  },
]
