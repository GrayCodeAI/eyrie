/**
 * Multi-Provider LLM SDKs
 *
 * Re-exports from all major LLM provider SDKs.
 * Support: Anthropic, OpenAI, Google Gemini, Groq
 */
// ============================================================================
// Anthropic (Claude)
// ============================================================================
export { default as Anthropic } from '@anthropic-ai/sdk';
export { APIError as AnthropicAPIError, APIConnectionError as AnthropicAPIConnectionError, APIConnectionTimeoutError as AnthropicAPIConnectionTimeoutError, APIUserAbortError as AnthropicAPIUserAbortError, } from '@anthropic-ai/sdk';
// ============================================================================
// OpenAI (GPT-4, GPT-3.5)
// ============================================================================
export { default as OpenAI } from 'openai';
export { APIError as OpenAIAPIError, APIConnectionError as OpenAIAPIConnectionError, APIConnectionTimeoutError as OpenAIAPIConnectionTimeoutError, } from 'openai';
// ============================================================================
// Google (Gemini)
// ============================================================================
export { GoogleGenerativeAI as Gemini } from '@google/generative-ai';
// ============================================================================
// Groq (Fast inference)
// ============================================================================
export { default as Groq } from 'groq-sdk';
// ============================================================================
// Provider Registry
// ============================================================================
export const PROVIDERS = {
    ANTHROPIC: 'anthropic',
    OPENAI: 'openai',
    GEMINI: 'gemini',
    GROQ: 'groq',
};
