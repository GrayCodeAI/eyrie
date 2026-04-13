import test from 'node:test';
import assert from 'node:assert/strict';
import { detectProvider, resolveProviderModelEnvOverride } from '../factory.js';
test('detectProvider prioritizes openrouter over OPENAI_API_KEY', () => {
    const env = process.env;
    const prev = {
        ANTHROPIC_API_KEY: env.ANTHROPIC_API_KEY,
        OPENROUTER_API_KEY: env.OPENROUTER_API_KEY,
        OPENAI_API_KEY: env.OPENAI_API_KEY,
    };
    try {
        env.ANTHROPIC_API_KEY = '';
        env.OPENROUTER_API_KEY = 'openrouter-key';
        env.OPENAI_API_KEY = 'openai-key';
        assert.equal(detectProvider(), 'openrouter');
    }
    finally {
        env.ANTHROPIC_API_KEY = prev.ANTHROPIC_API_KEY;
        env.OPENROUTER_API_KEY = prev.OPENROUTER_API_KEY;
        env.OPENAI_API_KEY = prev.OPENAI_API_KEY;
    }
});
test('resolveProviderModelEnvOverride keeps OpenRouter model provider-scoped', () => {
    const model = resolveProviderModelEnvOverride('openrouter', {
        OPENROUTER_MODEL: 'openai/gpt-4o-mini',
        OPENAI_MODEL: 'gpt-4o',
    });
    assert.equal(model, 'openai/gpt-4o-mini');
});
