import test from 'node:test';
import assert from 'node:assert/strict';
import { resolveOpenAICompatibleRuntime } from '../openaiCompatibleRuntime.js';
test('runtime resolves Gemini ahead of OPENAI_API_KEY', () => {
    const runtime = resolveOpenAICompatibleRuntime({
        env: {
            OPENAI_API_KEY: 'openai-key',
            GEMINI_API_KEY: 'gemini-key',
            GEMINI_MODEL: 'gemini-2.5-pro',
        },
    });
    assert.equal(runtime.mode, 'gemini');
    assert.equal(runtime.apiKeySource, 'gemini');
    assert.equal(runtime.request.resolvedModel, 'gemini-2.5-pro');
});
