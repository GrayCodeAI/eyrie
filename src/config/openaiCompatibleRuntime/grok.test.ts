import test from 'node:test'
import assert from 'node:assert/strict'
import { resolveOpenAICompatibleRuntime } from '../openaiCompatibleRuntime.js'

test('runtime resolves Grok with XAI_API_KEY fallback', () => {
  const runtime = resolveOpenAICompatibleRuntime({
    env: {
      OPENAI_API_KEY: 'openai-key',
      XAI_API_KEY: 'xai-key',
      GROK_MODEL: 'grok-2-1212',
      GROK_BASE_URL: 'https://api.x.ai/v1',
    } as NodeJS.ProcessEnv,
  })

  assert.equal(runtime.mode, 'grok')
  assert.equal(runtime.apiKeySource, 'xai')
  assert.equal(runtime.apiKey, 'xai-key')
  assert.equal(runtime.request.resolvedModel, 'grok-2-1212')
})
