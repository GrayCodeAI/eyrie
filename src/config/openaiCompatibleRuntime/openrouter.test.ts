import test from 'node:test'
import assert from 'node:assert/strict'
import { resolveOpenAICompatibleRuntime } from '../openaiCompatibleRuntime.js'

test('runtime resolves OpenRouter when both OPENROUTER_API_KEY and OPENAI_API_KEY are set', () => {
  const runtime = resolveOpenAICompatibleRuntime({
    env: {
      OPENAI_API_KEY: 'openai-key',
      OPENROUTER_API_KEY: 'openrouter-key',
      OPENROUTER_MODEL: 'google/gemma-4-26b-a4b-it:free',
    } as NodeJS.ProcessEnv,
  })

  assert.equal(runtime.mode, 'openrouter')
  assert.equal(runtime.apiKeySource, 'openrouter')
  assert.equal(runtime.request.resolvedModel, 'google/gemma-4-26b-a4b-it:free')
})
