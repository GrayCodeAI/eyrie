import test from 'node:test'
import assert from 'node:assert/strict'
import {
  resolveOpenAICompatibleRuntime,
  isOpenAICompatibleRuntimeEnabled,
} from './openaiCompatibleRuntime.js'

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

test('runtime resolves Gemini ahead of OPENAI_API_KEY', () => {
  const runtime = resolveOpenAICompatibleRuntime({
    env: {
      OPENAI_API_KEY: 'openai-key',
      GEMINI_API_KEY: 'gemini-key',
      GEMINI_MODEL: 'gemini-2.5-pro',
    } as NodeJS.ProcessEnv,
  })

  assert.equal(runtime.mode, 'gemini')
  assert.equal(runtime.apiKeySource, 'gemini')
  assert.equal(runtime.request.resolvedModel, 'gemini-2.5-pro')
})

test('runtime enabled check accepts provider-scoped keys', () => {
  assert.equal(
    isOpenAICompatibleRuntimeEnabled({
      OPENROUTER_API_KEY: 'key',
    } as NodeJS.ProcessEnv),
    true,
  )
  assert.equal(
    isOpenAICompatibleRuntimeEnabled({
      ANTHROPIC_API_KEY: 'key',
    } as NodeJS.ProcessEnv),
    true,
  )
  assert.equal(
    isOpenAICompatibleRuntimeEnabled({} as NodeJS.ProcessEnv),
    false,
  )
})

test('runtime resolves Anthropic ahead of OPENAI_API_KEY', () => {
  const runtime = resolveOpenAICompatibleRuntime({
    env: {
      OPENAI_API_KEY: 'openai-key',
      ANTHROPIC_API_KEY: 'anthropic-key',
      ANTHROPIC_MODEL: 'claude-3-5-sonnet-latest',
      ANTHROPIC_BASE_URL: 'https://api.anthropic.com/v1',
    } as NodeJS.ProcessEnv,
  })

  assert.equal(runtime.mode, 'anthropic')
  assert.equal(runtime.apiKeySource, 'anthropic')
  assert.equal(runtime.apiKey, 'anthropic-key')
})
