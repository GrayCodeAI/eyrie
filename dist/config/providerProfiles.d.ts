export type { APIProvider, RuntimeProviderProfile } from './providerProfiles/types.js';
import type { APIProvider, RuntimeProviderProfile } from './providerProfiles/types.js';
export declare const API_PROVIDER_DETECTION_ORDER: APIProvider[];
export declare const PROVIDER_MODEL_ENV_KEYS: Record<APIProvider, string[]>;
export declare const OLLAMA_DEFAULT_BASE_URL = "http://localhost:11434/v1";
export declare const OLLAMA_DEFAULT_MODEL = "llama3.1:8b";
export declare const OPENCODEGO_DEFAULT_BASE_URL = "https://api.opencode.ai/v1";
export declare const OPENCODEGO_DEFAULT_MODEL = "opencode-go";
export declare const OPENAI_COMPATIBLE_RUNTIME_PROFILE_ORDER: readonly ["openrouter", "grok", "gemini", "anthropic", "canopywave", "openai", "opencodego"];
export type OpenAICompatibleRuntimeProfileKey = (typeof OPENAI_COMPATIBLE_RUNTIME_PROFILE_ORDER)[number];
export declare const OPENAI_COMPATIBLE_RUNTIME_PROFILES: Record<OpenAICompatibleRuntimeProfileKey, RuntimeProviderProfile>;
//# sourceMappingURL=providerProfiles.d.ts.map