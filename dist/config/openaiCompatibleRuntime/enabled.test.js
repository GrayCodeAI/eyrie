import test from 'node:test';
import assert from 'node:assert/strict';
import { isOpenAICompatibleRuntimeEnabled } from '../openaiCompatibleRuntime.js';
test('runtime enabled check accepts provider-scoped keys', () => {
    assert.equal(isOpenAICompatibleRuntimeEnabled({ OPENROUTER_API_KEY: 'key' }), true);
    assert.equal(isOpenAICompatibleRuntimeEnabled({ ANTHROPIC_API_KEY: 'key' }), true);
    assert.equal(isOpenAICompatibleRuntimeEnabled({ CANOPYWAVE_API_KEY: 'key' }), true);
    assert.equal(isOpenAICompatibleRuntimeEnabled({}), false);
});
