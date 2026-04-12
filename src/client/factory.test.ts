import test from 'node:test'
import assert from 'node:assert/strict'
import {
  detectProvider,
  resolveProviderModelEnvOverride,
} from './factory.js'

test('detectProvider prioritizes provider-scoped keys over OPENAI_API_KEY', () => {
  const env = process.env
  const prev = {
    ANTHROPIC_API_KEY: env.ANTHROPIC_API_KEY,
    OPENAI_API_KEY: env.OPENAI_API_KEY,
    CANOPYWAVE_API_KEY: env.CANOPYWAVE_API_KEY,
    OPENROUTER_API_KEY: env.OPENROUTER_API_KEY,
    GROK_API_KEY: env.GROK_API_KEY,
    XAI_API_KEY: env.XAI_API_KEY,
    GEMINI_API_KEY: env.GEMINI_API_KEY,
    OLLAMA_BASE_URL: env.OLLAMA_BASE_URL,
  }

  try {
    env.ANTHROPIC_API_KEY = ''
    env.OPENAI_API_KEY = 'openai-key'
    env.CANOPYWAVE_API_KEY = ''
    env.OPENROUTER_API_KEY = 'openrouter-key'
    env.GROK_API_KEY = ''
    env.XAI_API_KEY = ''
    env.GEMINI_API_KEY = ''
    env.OLLAMA_BASE_URL = ''

    assert.equal(detectProvider(), 'openrouter')
  } finally {
    env.ANTHROPIC_API_KEY = prev.ANTHROPIC_API_KEY
    env.OPENAI_API_KEY = prev.OPENAI_API_KEY
    env.CANOPYWAVE_API_KEY = prev.CANOPYWAVE_API_KEY
    env.OPENROUTER_API_KEY = prev.OPENROUTER_API_KEY
    env.GROK_API_KEY = prev.GROK_API_KEY
    env.XAI_API_KEY = prev.XAI_API_KEY
    env.GEMINI_API_KEY = prev.GEMINI_API_KEY
    env.OLLAMA_BASE_URL = prev.OLLAMA_BASE_URL
  }
})

test('resolveProviderModelEnvOverride keeps OpenRouter model provider-scoped', () => {
  const model = resolveProviderModelEnvOverride('openrouter', {
    OPENROUTER_MODEL: 'openai/gpt-4o-mini',
    OPENAI_MODEL: 'gpt-4o',
  } as NodeJS.ProcessEnv)
  assert.equal(model, 'openai/gpt-4o-mini')
})

test('detectProvider resolves CanopyWave ahead of OPENAI', () => {
  const env = process.env
  const prev = {
    ANTHROPIC_API_KEY: env.ANTHROPIC_API_KEY,
    OPENROUTER_API_KEY: env.OPENROUTER_API_KEY,
    GROK_API_KEY: env.GROK_API_KEY,
    XAI_API_KEY: env.XAI_API_KEY,
    GEMINI_API_KEY: env.GEMINI_API_KEY,
    CANOPYWAVE_API_KEY: env.CANOPYWAVE_API_KEY,
    OPENAI_API_KEY: env.OPENAI_API_KEY,
    OLLAMA_BASE_URL: env.OLLAMA_BASE_URL,
  }

  try {
    env.ANTHROPIC_API_KEY = ''
    env.OPENROUTER_API_KEY = ''
    env.GROK_API_KEY = ''
    env.XAI_API_KEY = ''
    env.GEMINI_API_KEY = ''
    env.CANOPYWAVE_API_KEY = 'cw-key'
    env.OPENAI_API_KEY = 'openai-key'
    env.OLLAMA_BASE_URL = ''
    assert.equal(detectProvider(), 'canopywave')
  } finally {
    env.ANTHROPIC_API_KEY = prev.ANTHROPIC_API_KEY
    env.OPENROUTER_API_KEY = prev.OPENROUTER_API_KEY
    env.GROK_API_KEY = prev.GROK_API_KEY
    env.XAI_API_KEY = prev.XAI_API_KEY
    env.GEMINI_API_KEY = prev.GEMINI_API_KEY
    env.CANOPYWAVE_API_KEY = prev.CANOPYWAVE_API_KEY
    env.OPENAI_API_KEY = prev.OPENAI_API_KEY
    env.OLLAMA_BASE_URL = prev.OLLAMA_BASE_URL
  }
})

test('resolveProviderModelEnvOverride keeps CanopyWave model provider-scoped', () => {
  const model = resolveProviderModelEnvOverride('canopywave', {
    CANOPYWAVE_MODEL: 'zai/glm-4.6',
    OPENAI_MODEL: 'gpt-4o',
  } as NodeJS.ProcessEnv)
  assert.equal(model, 'zai/glm-4.6')
})
