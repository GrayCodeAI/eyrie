import test from 'node:test';
import assert from 'node:assert/strict';
import { detectProvider } from '../factory.js';
test('detectProvider resolves anthropic when ANTHROPIC_API_KEY is set', () => {
    const env = process.env;
    const prev = {
        ANTHROPIC_API_KEY: env.ANTHROPIC_API_KEY,
        OPENAI_API_KEY: env.OPENAI_API_KEY,
    };
    try {
        env.ANTHROPIC_API_KEY = 'anthropic-key';
        env.OPENAI_API_KEY = 'openai-key';
        assert.equal(detectProvider(), 'anthropic');
    }
    finally {
        env.ANTHROPIC_API_KEY = prev.ANTHROPIC_API_KEY;
        env.OPENAI_API_KEY = prev.OPENAI_API_KEY;
    }
});
