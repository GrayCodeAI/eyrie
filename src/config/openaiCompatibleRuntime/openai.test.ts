import test from 'node:test'
import assert from 'node:assert/strict'
import { resolveOpenAICompatibleRuntime } from '../openaiCompatibleRuntime.js'

test('runtime resolves OpenAI when only OPENAI_API_KEY is set', () => {
  const runtime = resolveOpenAICompatibleRuntime({
    env: {
      OPENAI_API_KEY: 'openai-key',
      OPENAI_MODEL: 'gpt-4o-mini',
    } as NodeJS.ProcessEnv,
  })

  assert.equal(runtime.mode, 'openai')
  assert.equal(runtime.apiKeySource, 'openai')
  assert.equal(runtime.apiKey, 'openai-key')
  assert.equal(runtime.request.resolvedModel, 'gpt-4o-mini')
})
