import { ANTHROPIC_MODELS } from './anthropic.js';
import { CANOPYWAVE_MODELS } from './canopywave.js';
import { GEMINI_MODELS } from './gemini.js';
import { GROK_MODELS } from './grok.js';
import { OLLAMA_MODELS } from './ollama.js';
import { OPENAI_MODELS } from './openai.js';
import { OPENROUTER_MODELS } from './openrouter.js';
import { OPENCODEGO_MODELS } from './opencodego.js';
export const DEFAULT_PROVIDER_CATALOGS = {
    anthropic: ANTHROPIC_MODELS,
    canopywave: CANOPYWAVE_MODELS,
    openai: OPENAI_MODELS,
    openrouter: OPENROUTER_MODELS,
    grok: GROK_MODELS,
    gemini: GEMINI_MODELS,
    ollama: OLLAMA_MODELS,
    opencodego: OPENCODEGO_MODELS,
};
