import test from 'node:test';
import assert from 'node:assert/strict';
import { detectProvider } from '../factory.js';
test('detectProvider resolves ollama when only OLLAMA_BASE_URL is set', () => {
    const env = process.env;
    const prev = {
        ANTHROPIC_API_KEY: env.ANTHROPIC_API_KEY,
        OPENROUTER_API_KEY: env.OPENROUTER_API_KEY,
        GROK_API_KEY: env.GROK_API_KEY,
        XAI_API_KEY: env.XAI_API_KEY,
        GEMINI_API_KEY: env.GEMINI_API_KEY,
        CANOPYWAVE_API_KEY: env.CANOPYWAVE_API_KEY,
        OPENAI_API_KEY: env.OPENAI_API_KEY,
        OLLAMA_BASE_URL: env.OLLAMA_BASE_URL,
    };
    try {
        env.ANTHROPIC_API_KEY = '';
        env.OPENROUTER_API_KEY = '';
        env.GROK_API_KEY = '';
        env.XAI_API_KEY = '';
        env.GEMINI_API_KEY = '';
        env.CANOPYWAVE_API_KEY = '';
        env.OPENAI_API_KEY = '';
        env.OLLAMA_BASE_URL = 'http://localhost:11434';
        assert.equal(detectProvider(), 'ollama');
    }
    finally {
        env.ANTHROPIC_API_KEY = prev.ANTHROPIC_API_KEY;
        env.OPENROUTER_API_KEY = prev.OPENROUTER_API_KEY;
        env.GROK_API_KEY = prev.GROK_API_KEY;
        env.XAI_API_KEY = prev.XAI_API_KEY;
        env.GEMINI_API_KEY = prev.GEMINI_API_KEY;
        env.CANOPYWAVE_API_KEY = prev.CANOPYWAVE_API_KEY;
        env.OPENAI_API_KEY = prev.OPENAI_API_KEY;
        env.OLLAMA_BASE_URL = prev.OLLAMA_BASE_URL;
    }
});
