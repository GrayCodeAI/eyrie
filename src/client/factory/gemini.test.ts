import test from 'node:test'
import assert from 'node:assert/strict'
import { detectProvider } from '../factory.js'

test('detectProvider resolves gemini when GEMINI_API_KEY is set', () => {
  const env = process.env
  const prev = {
    ANTHROPIC_API_KEY: env.ANTHROPIC_API_KEY,
    OPENROUTER_API_KEY: env.OPENROUTER_API_KEY,
    GROK_API_KEY: env.GROK_API_KEY,
    XAI_API_KEY: env.XAI_API_KEY,
    GEMINI_API_KEY: env.GEMINI_API_KEY,
  }

  try {
    env.ANTHROPIC_API_KEY = ''
    env.OPENROUTER_API_KEY = ''
    env.GROK_API_KEY = ''
    env.XAI_API_KEY = ''
    env.GEMINI_API_KEY = 'gem-key'
    assert.equal(detectProvider(), 'gemini')
  } finally {
    env.ANTHROPIC_API_KEY = prev.ANTHROPIC_API_KEY
    env.OPENROUTER_API_KEY = prev.OPENROUTER_API_KEY
    env.GROK_API_KEY = prev.GROK_API_KEY
    env.XAI_API_KEY = prev.XAI_API_KEY
    env.GEMINI_API_KEY = prev.GEMINI_API_KEY
  }
})
