import test from 'node:test'
import assert from 'node:assert/strict'
import { resolveOpenAICompatibleRuntime } from '../openaiCompatibleRuntime.js'

test('runtime resolves CanopyWave ahead of OPENAI_API_KEY', () => {
  const runtime = resolveOpenAICompatibleRuntime({
    env: {
      OPENAI_API_KEY: 'openai-key',
      CANOPYWAVE_API_KEY: 'canopywave-key',
      CANOPYWAVE_MODEL: 'zai/glm-4.6',
      CANOPYWAVE_BASE_URL: 'https://inference.canopywave.io/v1',
    } as NodeJS.ProcessEnv,
  })

  assert.equal(runtime.mode, 'openai')
  assert.equal(runtime.apiKeySource, 'canopywave')
  assert.equal(runtime.apiKey, 'canopywave-key')
  assert.equal(runtime.request.baseUrl, 'https://inference.canopywave.io/v1')
  assert.equal(runtime.request.resolvedModel, 'zai/glm-4.6')
})
