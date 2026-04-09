import type { ModelCatalogEntry } from '../types.js'

export const ANTHROPIC_MODELS: ModelCatalogEntry[] = [
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
]
