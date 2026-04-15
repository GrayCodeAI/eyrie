import test from 'node:test';
import assert from 'node:assert/strict';
import { resolveOpenAICompatibleRuntime, isOpenAICompatibleRuntimeEnabled } from '../openaiCompatibleRuntime.js';
test('runtime resolves opencodego when OPENCODEGO_API_KEY is set', () => {
    const runtime = resolveOpenAICompatibleRuntime({
        env: {
            OPENCODEGO_API_KEY: 'ocg-test-key',
        },
    });
    assert.equal(runtime.mode, 'opencodego');
    assert.equal(runtime.apiKeySource, 'opencodego');
    assert.equal(runtime.apiKey, 'ocg-test-key');
    assert.equal(runtime.request.baseUrl, 'https://opencode.ai/zen/go/v1');
    assert.equal(runtime.request.resolvedModel, 'kimi-k2.5');
});
test('runtime resolves opencodego with custom model and base URL', () => {
    const runtime = resolveOpenAICompatibleRuntime({
        env: {
            OPENCODEGO_API_KEY: 'ocg-test-key',
            OPENCODEGO_MODEL: 'glm-5.1',
            OPENCODEGO_BASE_URL: 'https://custom.opencode.ai/v1',
        },
    });
    assert.equal(runtime.mode, 'opencodego');
    assert.equal(runtime.request.resolvedModel, 'glm-5.1');
    assert.equal(runtime.request.baseUrl, 'https://custom.opencode.ai/v1');
});
test('isOpenAICompatibleRuntimeEnabled returns true for OPENCODEGO_API_KEY', () => {
    assert.equal(isOpenAICompatibleRuntimeEnabled({ OPENCODEGO_API_KEY: 'ocg-key' }), true);
});
