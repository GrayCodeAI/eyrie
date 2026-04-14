import type { ModelCatalogEntry } from '../types.js'

export const OPENCODEGO_MODELS: ModelCatalogEntry[] = [
  {
    id: 'opencode-go/glm-5.1',
    input_price_per_1m: 5,
    output_price_per_1m: 15,
    context_window: 128000,
    max_output: 8000,
  },
  {
    id: 'opencode-go/glm-5',
    input_price_per_1m: 5,
    output_price_per_1m: 15,
    context_window: 128000,
    max_output: 8000,
  },
  {
    id: 'opencode-go/kimi-k2.5',
    input_price_per_1m: 3,
    output_price_per_1m: 10,
    context_window: 256000,
    max_output: 8000,
  },
  {
    id: 'opencode-go/mimo-v2-pro',
    input_price_per_1m: 3,
    output_price_per_1m: 10,
    context_window: 128000,
    max_output: 8000,
  },
  {
    id: 'opencode-go/mimo-v2-omni',
    input_price_per_1m: 2,
    output_price_per_1m: 8,
    context_window: 128000,
    max_output: 8000,
  },
  {
    id: 'opencode-go/minimax-m2.7',
    input_price_per_1m: 1,
    output_price_per_1m: 3,
    context_window: 1000000,
    max_output: 8000,
  },
  {
    id: 'opencode-go/minimax-m2.5',
    input_price_per_1m: 0.5,
    output_price_per_1m: 1.5,
    context_window: 1000000,
    max_output: 8000,
  },
]
