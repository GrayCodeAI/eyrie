import type { APIProvider } from '../../client/factory.js'
import type { ModelCatalogEntry } from '../types.js'
import { ANTHROPIC_MODELS } from './anthropic.js'
import { GEMINI_MODELS } from './gemini.js'
import { GROK_MODELS } from './grok.js'
import { OLLAMA_MODELS } from './ollama.js'
import { OPENAI_MODELS } from './openai.js'

export const DEFAULT_PROVIDER_CATALOGS: Partial<
  Record<APIProvider, ModelCatalogEntry[]>
> = {
  anthropic: ANTHROPIC_MODELS,
  openai: OPENAI_MODELS,
  grok: GROK_MODELS,
  gemini: GEMINI_MODELS,
  ollama: OLLAMA_MODELS,
}
