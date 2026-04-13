import test from 'node:test'
import assert from 'node:assert/strict'
import { detectProvider } from '../factory.js'

test('detectProvider resolves grok when only XAI_API_KEY is set', () => {
  const env = process.env
  const prev = {
    ANTHROPIC_API_KEY: env.ANTHROPIC_API_KEY,
    OPENROUTER_API_KEY: env.OPENROUTER_API_KEY,
    GROK_API_KEY: env.GROK_API_KEY,
    XAI_API_KEY: env.XAI_API_KEY,
  }

  try {
    env.ANTHROPIC_API_KEY = ''
    env.OPENROUTER_API_KEY = ''
    env.GROK_API_KEY = ''
    env.XAI_API_KEY = 'xai-key'
    assert.equal(detectProvider(), 'grok')
  } finally {
    env.ANTHROPIC_API_KEY = prev.ANTHROPIC_API_KEY
    env.OPENROUTER_API_KEY = prev.OPENROUTER_API_KEY
    env.GROK_API_KEY = prev.GROK_API_KEY
    env.XAI_API_KEY = prev.XAI_API_KEY
  }
})
