import type { ModelCatalogEntry } from '../types.js'

export const OPENROUTER_MODELS: ModelCatalogEntry[] = [
  {
    id: 'openai/gpt-4o',
    input_price_per_1m: 5,
    output_price_per_1m: 15,
    context_window: 128000,
    max_output: 16000,
    server_tools: ['web_search'],
  },
  {
    id: 'openai/gpt-4o-mini',
    input_price_per_1m: 0.15,
    output_price_per_1m: 0.6,
    context_window: 128000,
    max_output: 16000,
    server_tools: ['web_search'],
  },
  {
    id: 'anthropic/claude-sonnet-4-6',
    input_price_per_1m: 3,
    output_price_per_1m: 15,
    context_window: 200000,
    max_output: 32000,
    server_tools: ['web_search'],
  },
]
