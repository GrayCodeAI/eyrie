import test from 'node:test';
import assert from 'node:assert/strict';
import { resolveOpenAICompatibleRuntime } from '../openaiCompatibleRuntime.js';
test('runtime resolves Ollama when OLLAMA_BASE_URL is set', () => {
    const runtime = resolveOpenAICompatibleRuntime({
        env: {
            OLLAMA_BASE_URL: 'http://localhost:11434/v1',
            OLLAMA_MODEL: 'llama3.1:8b',
        },
    });
    assert.equal(runtime.mode, 'openai');
    assert.equal(runtime.apiKeySource, 'none');
    assert.equal(runtime.apiKey, '');
    assert.equal(runtime.request.baseUrl, 'http://localhost:11434/v1');
    assert.equal(runtime.request.resolvedModel, 'llama3.1:8b');
});
