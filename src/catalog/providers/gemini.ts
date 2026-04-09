import type { ModelCatalogEntry } from '../types.js'

export const GEMINI_MODELS: ModelCatalogEntry[] = [
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
]
