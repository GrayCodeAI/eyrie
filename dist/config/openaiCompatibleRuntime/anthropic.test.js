import test from 'node:test';
import assert from 'node:assert/strict';
import { resolveOpenAICompatibleRuntime } from '../openaiCompatibleRuntime.js';
test('runtime resolves Anthropic ahead of OPENAI_API_KEY', () => {
    const runtime = resolveOpenAICompatibleRuntime({
        env: {
            OPENAI_API_KEY: 'openai-key',
            ANTHROPIC_API_KEY: 'anthropic-key',
            ANTHROPIC_MODEL: 'claude-3-5-sonnet-latest',
            ANTHROPIC_BASE_URL: 'https://api.anthropic.com/v1',
        },
    });
    assert.equal(runtime.mode, 'anthropic');
    assert.equal(runtime.apiKeySource, 'anthropic');
    assert.equal(runtime.apiKey, 'anthropic-key');
});
