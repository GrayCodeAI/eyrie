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
// AWS Bedrock (Amazon Titan, Claude via AWS)
// ============================================================================
export { BedrockRuntimeClient as Bedrock, InvokeModelCommand, InvokeModelWithResponseStreamCommand, ConverseCommand, ConverseStreamCommand, } from '@aws-sdk/client-bedrock-runtime';
// ============================================================================
// Vercel AI SDK (Universal AI SDK)
// ============================================================================
export { generateText, streamText, generateObject, streamObject, embed, embedMany, } from 'ai';
// ============================================================================
// Google Vertex AI
// ============================================================================
export { VertexAI } from '@google-cloud/vertexai';
// ============================================================================
// Provider Registry
// ============================================================================
export const PROVIDERS = {
    ANTHROPIC: 'anthropic',
    OPENAI: 'openai',
    GEMINI: 'gemini',
    GROQ: 'groq',
    VERCEL: 'vercel',
    VERTEX: 'vertex',
    BEDROCK: 'bedrock',
};
