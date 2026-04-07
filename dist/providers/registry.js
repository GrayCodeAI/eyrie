/**
 * Unified Provider Registry
 *
 * Following the pi-mono pattern - one SDK, many providers via baseUrl
 */
import { OpenAI } from 'openai';
// ============================================================================
// Core SDK Providers (Dedicated SDKs)
// ============================================================================
export const CORE_PROVIDERS = {
    anthropic: {
        name: 'anthropic',
        type: 'anthropic',
        envKey: 'ANTHROPIC_API_KEY',
        defaultModel: 'claude-3-5-sonnet-20241022',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: true,
    },
    openai: {
        name: 'openai',
        type: 'openai',
        baseUrl: 'https://api.openai.com/v1',
        envKey: 'OPENAI_API_KEY',
        defaultModel: 'gpt-4o-mini',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: true,
        compat: {
            supportsStore: true,
            supportsDeveloperRole: true,
            supportsReasoningEffort: true,
        }
    },
    google: {
        name: 'google',
        type: 'google',
        envKey: 'GEMINI_API_KEY',
        defaultModel: 'gemini-1.5-flash',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: true,
    },
    mistral: {
        name: 'mistral',
        type: 'mistral',
        envKey: 'MISTRAL_API_KEY',
        defaultModel: 'mistral-large-latest',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: false,
    },
    bedrock: {
        name: 'bedrock',
        type: 'bedrock',
        envKey: 'AWS_ACCESS_KEY_ID',
        defaultModel: 'anthropic.claude-3-5-sonnet-20241022-v2:0',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: false,
    }
};
// ============================================================================
// OpenAI-Compatible Providers (Using OpenAI SDK with custom baseUrl)
// ============================================================================
export const OPENAI_COMPATIBLE_PROVIDERS = {
    // Grok (xAI)
    grok: {
        name: 'grok',
        type: 'openai-compatible',
        baseUrl: 'https://api.x.ai/v1',
        envKey: 'XAI_API_KEY',
        defaultModel: 'grok-2',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: true,
        compat: {
            supportsReasoningEffort: false,
            supportsStore: false,
            supportsDeveloperRole: false,
        }
    },
    // Groq
    groq: {
        name: 'groq',
        type: 'openai-compatible',
        baseUrl: 'https://api.groq.com/openai/v1',
        envKey: 'GROQ_API_KEY',
        defaultModel: 'llama-3.1-70b-versatile',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: false,
    },
    // DeepSeek
    deepseek: {
        name: 'deepseek',
        type: 'openai-compatible',
        baseUrl: 'https://api.deepseek.com/v1',
        envKey: 'DEEPSEEK_API_KEY',
        defaultModel: 'deepseek-chat',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: true,
    },
    // Z.ai (Zhipu AI / GLM)
    zai: {
        name: 'zai',
        type: 'openai-compatible',
        baseUrl: 'https://api.z.ai/v1',
        envKey: 'ZAI_API_KEY',
        defaultModel: 'glm-4-flash',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: true,
        compat: {
            supportsReasoningEffort: false,
            supportsStore: false,
            supportsDeveloperRole: false,
            thinkingFormat: 'zai',
        }
    },
    // Cerebras
    cerebras: {
        name: 'cerebras',
        type: 'openai-compatible',
        baseUrl: 'https://api.cerebras.ai/v1',
        envKey: 'CEREBRAS_API_KEY',
        defaultModel: 'llama3.1-70b',
        supportsStreaming: true,
        supportsTools: false,
        supportsReasoning: false,
        compat: {
            supportsStore: false,
            supportsDeveloperRole: false,
        }
    },
    // OpenCode
    opencode: {
        name: 'opencode',
        type: 'openai-compatible',
        baseUrl: 'https://api.opencode.ai/v1',
        envKey: 'OPENCODE_API_KEY',
        defaultModel: 'opencode-zen',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: false,
    },
    // MiniMax
    minimax: {
        name: 'minimax',
        type: 'openai-compatible',
        baseUrl: 'https://api.minimax.chat/v1',
        envKey: 'MINIMAX_API_KEY',
        defaultModel: 'MiniMax-Text-01',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: false,
    },
    // Kimi (Moonshot)
    kimi: {
        name: 'kimi',
        type: 'openai-compatible',
        baseUrl: 'https://api.moonshot.cn/v1',
        envKey: 'MOONSHOT_API_KEY',
        defaultModel: 'kimi-latest',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: true,
    },
    // OpenRouter
    openrouter: {
        name: 'openrouter',
        type: 'openai-compatible',
        baseUrl: 'https://openrouter.ai/api/v1',
        envKey: 'OPENROUTER_API_KEY',
        defaultModel: 'anthropic/claude-3.5-sonnet',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: true,
        compat: {
            thinkingFormat: 'openrouter',
        }
    },
    // Vercel AI Gateway
    vercel: {
        name: 'vercel',
        type: 'openai-compatible',
        baseUrl: 'https://ai-gateway.vercel.sh/v1',
        envKey: 'AI_GATEWAY_API_KEY',
        defaultModel: 'gpt-4o-mini',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: false,
    },
    // Local/Custom
    ollama: {
        name: 'ollama',
        type: 'openai-compatible',
        baseUrl: 'http://localhost:11434/v1',
        envKey: 'OLLAMA_API_KEY',
        defaultModel: 'llama3.1',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: false,
        compat: {
            supportsStore: false,
            supportsDeveloperRole: false,
            supportsReasoningEffort: false,
        }
    },
    // ===== NEW PROVIDERS (matching OpenCode's 75+) =====
    // Cloudflare AI Gateway
    cloudflare: {
        name: 'cloudflare',
        type: 'openai-compatible',
        baseUrl: 'https://gateway.ai.cloudflare.com/client/v4',
        envKey: 'CLOUDFLARE_API_TOKEN',
        defaultModel: 'gpt-4o',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: true,
    },
    // Azure OpenAI
    azure: {
        name: 'azure',
        type: 'openai-compatible',
        baseUrl: 'https://${AZURE_RESOURCE_NAME}.openai.azure.com',
        envKey: 'AZURE_OPENAI_API_KEY',
        defaultModel: 'gpt-4o',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: true,
    },
    // Deep Infra
    deepinfra: {
        name: 'deepinfra',
        type: 'openai-compatible',
        baseUrl: 'https://api.deepinfra.com/v1',
        envKey: 'DEEPINFRA_API_KEY',
        defaultModel: 'meta-llama/Llama-3.1-70B-Instruct',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: false,
    },
    // Fireworks AI
    fireworks: {
        name: 'fireworks',
        type: 'openai-compatible',
        baseUrl: 'https://api.fireworks.ai/v1',
        envKey: 'FIREWORKS_API_KEY',
        defaultModel: 'accounts/fireworks/models/llama-v3p1-70b-instruct',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: true,
    },
    // Together AI
    together: {
        name: 'together',
        type: 'openai-compatible',
        baseUrl: 'https://api.together.xyz/v1',
        envKey: 'TOGETHER_API_KEY',
        defaultModel: 'meta-llama/Llama-3.1-70B-Instruct',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: true,
    },
    // Venice AI
    venice: {
        name: 'venice',
        type: 'openai-compatible',
        baseUrl: 'https://api.venice.ai/v1',
        envKey: 'VENICE_API_KEY',
        defaultModel: 'llama-3.1-70b',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: false,
    },
    // Nebius
    nebius: {
        name: 'nebius',
        type: 'openai-compatible',
        baseUrl: 'https://api.nebius.ai/v1',
        envKey: 'NEBIUS_API_KEY',
        defaultModel: 'meta-llama/Llama-3.1-70B-Instruct',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: false,
    },
    // IO.NET
    ionet: {
        name: 'ionet',
        type: 'openai-compatible',
        baseUrl: 'https://api.io.net/v1',
        envKey: 'IONET_API_KEY',
        defaultModel: 'meta-llama/Llama-3.1-70B-Instruct',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: false,
    },
    // LM Studio (local)
    lmstudio: {
        name: 'lmstudio',
        type: 'openai-compatible',
        baseUrl: 'http://localhost:1234/v1',
        envKey: 'LMSTUDIO_API_KEY',
        defaultModel: 'gemma-3n-e4b',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: false,
        compat: {
            supportsStore: false,
            supportsDeveloperRole: false,
            supportsReasoningEffort: false,
        }
    },
    // llama.cpp (local)
    llamacpp: {
        name: 'llamacpp',
        type: 'openai-compatible',
        baseUrl: 'http://localhost:8080/v1',
        envKey: 'LLAMACPP_API_KEY',
        defaultModel: 'qwen3-coder-a3b',
        supportsStreaming: true,
        supportsTools: false,
        supportsReasoning: false,
        compat: {
            supportsStore: false,
            supportsDeveloperRole: false,
            supportsReasoningEffort: false,
        }
    },
    // Ollama Cloud
    ollamacloud: {
        name: 'ollamacloud',
        type: 'openai-compatible',
        baseUrl: 'https://api.ollama.com/v1',
        envKey: 'OLLAMA_CLOUD_API_KEY',
        defaultModel: 'llama3.1',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: false,
    },
    // Baseten
    baseten: {
        name: 'baseten',
        type: 'openai-compatible',
        baseUrl: 'https://app.baseten.co/v1',
        envKey: 'BASETEN_API_KEY',
        defaultModel: 'qwen-72b',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: false,
    },
    // STACKIT
    stackit: {
        name: 'stackit',
        type: 'openai-compatible',
        baseUrl: 'https://ai-api.stackit.de/v1',
        envKey: 'STACKIT_AUTH_TOKEN',
        defaultModel: 'meta-llama-3.1-70b-instruct',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: false,
    },
    // OVHcloud
    ovhcloud: {
        name: 'ovhcloud',
        type: 'openai-compatible',
        baseUrl: 'https://endpoints.ovh.com/ai/v1',
        envKey: 'OVHcloud_API_KEY',
        defaultModel: 'gpt-oss-120b',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: false,
    },
    // Scaleway
    scaleway: {
        name: 'scaleway',
        type: 'openai-compatible',
        baseUrl: 'https://api.scaleway.com/ai/v1',
        envKey: 'SCALEWAY_API_KEY',
        defaultModel: 'devstral-2-123b-instruct-2512',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: false,
    },
    // 302.AI
    ai302: {
        name: '302ai',
        type: 'openai-compatible',
        baseUrl: 'https://api.302.ai/v1',
        envKey: 'AI302_API_KEY',
        defaultModel: 'gpt-4o',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: true,
    },
    // ZenMux
    zenmux: {
        name: 'zenmux',
        type: 'openai-compatible',
        baseUrl: 'https://api.zenmux.com/v1',
        envKey: 'ZENMUX_API_KEY',
        defaultModel: 'gpt-4o',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: true,
    },
    // Cortecs
    cortecs: {
        name: 'cortecs',
        type: 'openai-compatible',
        baseUrl: 'https://api.cortecs.io/v1',
        envKey: 'CORTECS_API_KEY',
        defaultModel: 'kimi-k2-instruct',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: true,
    },
    // Firmware
    firmware: {
        name: 'firmware',
        type: 'openai-compatible',
        baseUrl: 'https://api.firmware.ai/v1',
        envKey: 'FIRMWARE_API_KEY',
        defaultModel: 'gpt-4o',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: true,
    },
    // Hugging Face
    huggingface: {
        name: 'huggingface',
        type: 'openai-compatible',
        baseUrl: 'https://router.huggingface.co/hf-inference/v1',
        envKey: 'HF_TOKEN',
        defaultModel: 'meta-llama/Llama-3.1-70B-Instruct',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: false,
    },
    // Helicone (LLM gateway/observability)
    helicone: {
        name: 'helicone',
        type: 'openai-compatible',
        baseUrl: 'https://ai-gateway.helicone.ai',
        envKey: 'HELICONE_API_KEY',
        defaultModel: 'gpt-4o',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: true,
    },
    // OpenCode Zen (their recommended models)
    opencodezen: {
        name: 'opencode-zen',
        type: 'openai-compatible',
        baseUrl: 'https://api.opencode.ai/v1',
        envKey: 'OPENCODE_ZEN_API_KEY',
        defaultModel: 'opencode-zen',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: true,
    },
    // Canopy Wave
    canopywave: {
        name: 'canopywave',
        type: 'openai-compatible',
        baseUrl: 'https://inference.canopywave.io/v1',
        envKey: 'CANOPYWAVE_API_KEY',
        defaultModel: 'zai/glm-4.6',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: true,
    },
    // ===== ADDITIONAL PROVIDERS TO REACH 75+ =====
    // Perplexity
    perplexity: {
        name: 'perplexity',
        type: 'openai-compatible',
        baseUrl: 'https://api.perplexity.ai',
        envKey: 'PERPLEXITY_API_KEY',
        defaultModel: 'llama-3.1-sonar-small-128k-online',
        supportsStreaming: true,
        supportsTools: false,
        supportsReasoning: true,
    },
    // Anyscale
    anyscale: {
        name: 'anyscale',
        type: 'openai-compatible',
        baseUrl: 'https://api.endpoints.anyscale.com/v1',
        envKey: 'ANYSCALE_API_KEY',
        defaultModel: 'meta-llama/Llama-3.1-70B-Instruct',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: false,
    },
    // Replicate
    replicate: {
        name: 'replicate',
        type: 'openai-compatible',
        baseUrl: 'https://api.replicate.com/v1',
        envKey: 'REPLICATE_API_KEY',
        defaultModel: 'meta/llama-3.1-70b-instruct',
        supportsStreaming: true,
        supportsTools: false,
        supportsReasoning: false,
    },
    // Lepton AI
    lepton: {
        name: 'lepton',
        type: 'openai-compatible',
        baseUrl: 'https://api.lepton.ai/api/v1',
        envKey: 'LEPTON_API_KEY',
        defaultModel: 'llama3-70b',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: false,
    },
    // Predibase
    predibase: {
        name: 'predibase',
        type: 'openai-compatible',
        baseUrl: 'https://serving.predibase.com/v3',
        envKey: 'PREDIBASE_API_KEY',
        defaultModel: 'meta-llama/Llama-3.1-70B-Instruct',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: false,
    },
    // Hyperbolic
    hyperbolic: {
        name: 'hyperbolic',
        type: 'openai-compatible',
        baseUrl: 'https://api.hyperbolic.xyz/v1',
        envKey: 'HYPERBOLIC_API_KEY',
        defaultModel: 'meta-llama/Llama-3.1-70B-Instruct',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: false,
    },
    // Novita AI
    novita: {
        name: 'novita',
        type: 'openai-compatible',
        baseUrl: 'https://api.novita.ai/v1',
        envKey: 'NOVITA_API_KEY',
        defaultModel: 'meta-llama/Llama-3.1-70B-Instruct',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: false,
    },
    // Nitro
    nitro: {
        name: 'nitro',
        type: 'openai-compatible',
        baseUrl: 'https://api.nitrolabs.ai/v1',
        envKey: 'NITRO_API_KEY',
        defaultModel: 'meta-llama/Llama-3.1-70B-Instruct',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: false,
    },
    // Abacus AI
    abacus: {
        name: 'abacus',
        type: 'openai-compatible',
        baseUrl: 'https://api.alpha.ac.ai/v1',
        envKey: 'ABACUS_API_KEY',
        defaultModel: 'meta-llama/Llama-3.1-70B-Instruct',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: false,
    },
    // Featherless AI
    featherless: {
        name: 'featherless',
        type: 'openai-compatible',
        baseUrl: 'https://api.featherless.ai/v1',
        envKey: 'FEATHERLESS_API_KEY',
        defaultModel: 'meta-llama/Llama-3.1-70B-Instruct',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: false,
    },
    // Simmer AI
    simmer: {
        name: 'simmer',
        type: 'openai-compatible',
        baseUrl: 'https://api.simmer.ai/v1',
        envKey: 'SIMMER_API_KEY',
        defaultModel: 'gpt-4o',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: false,
    },
    // ChainStack
    chainstack: {
        name: 'chainstack',
        type: 'openai-compatible',
        baseUrl: 'https://chat.chainstack.ai/v1',
        envKey: 'CHAINSTACK_API_KEY',
        defaultModel: 'meta-llama/Llama-3.1-70B-Instruct',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: false,
    },
    // Kata.ai
    kata: {
        name: 'kata',
        type: 'openai-compatible',
        baseUrl: 'https://api.kata.ai/v1',
        envKey: 'KATA_API_KEY',
        defaultModel: 'meta-llama/Llama-3.1-70B-Instruct',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: false,
    },
    // Mancer
    mancer: {
        name: 'mancer',
        type: 'openai-compatible',
        baseUrl: 'https://api.mancer.ai/v1',
        envKey: 'MANCER_API_KEY',
        defaultModel: 'meta-llama/Llama-3.1-70B-Instruct',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: false,
    },
    // Azuredata
    azuredata: {
        name: 'azuredata',
        type: 'openai-compatible',
        baseUrl: 'https://<resource>.cognitiveservices.azure.com/',
        envKey: 'AZURE_DATA_API_KEY',
        defaultModel: 'gpt-4o',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: true,
    },
    // Vertex AI (Google Cloud)
    vertex: {
        name: 'vertex',
        type: 'google',
        envKey: 'GOOGLE_VERTEX_PROJECT',
        defaultModel: 'gemini-1.5-pro',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: true,
    },
    // GitHub Copilot
    githubcopilot: {
        name: 'github-copilot',
        type: 'openai-compatible',
        baseUrl: 'https://api.github.com/copilot-models/v1',
        envKey: 'GITHUB_COPILOT_TOKEN',
        defaultModel: 'gpt-4o',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: true,
    },
    // GitLab Duo
    gitlab: {
        name: 'gitlab',
        type: 'openai-compatible',
        baseUrl: 'https://gitlab.com/api/ai',
        envKey: 'GITLAB_TOKEN',
        defaultModel: 'duo-chat-sonnet-4-5',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: true,
    },
    // SAP AI Core
    sap: {
        name: 'sap-ai-core',
        type: 'openai-compatible',
        baseUrl: 'https://ai-core.cfapps.<region>.hana.ondemand.com',
        envKey: 'AICORE_SERVICE_KEY',
        defaultModel: 'gpt-4o',
        supportsStreaming: true,
        supportsTools: true,
        supportsReasoning: true,
    },
};
// ============================================================================
// All Providers
// ============================================================================
export const ALL_PROVIDERS = {
    ...CORE_PROVIDERS,
    ...OPENAI_COMPATIBLE_PROVIDERS
};
export const PROVIDER_LIST = Object.keys(ALL_PROVIDERS);
// ============================================================================
// Helper Functions
// ============================================================================
export function getProvider(name) {
    return ALL_PROVIDERS[name];
}
export function getAllProviders() {
    return Object.values(ALL_PROVIDERS);
}
export function getOpenAICompatibleProviders() {
    return Object.values(OPENAI_COMPATIBLE_PROVIDERS);
}
export function isOpenAICompatible(provider) {
    const config = ALL_PROVIDERS[provider];
    return config?.type === 'openai-compatible';
}
export function createOpenAIClient(provider, apiKey, baseUrl) {
    const config = ALL_PROVIDERS[provider];
    if (!config) {
        throw new Error(`Unknown provider: ${provider}`);
    }
    if (config.type !== 'openai-compatible') {
        throw new Error(`Provider ${provider} is not OpenAI-compatible`);
    }
    const key = apiKey || process.env[config.envKey];
    if (!key) {
        throw new Error(`API key required for ${provider}. ` +
            `Set ${config.envKey} environment variable or pass apiKey option.`);
    }
    return new OpenAI({
        apiKey: key,
        baseURL: baseUrl || config.baseUrl,
        dangerouslyAllowBrowser: true,
    });
}
// ============================================================================
// Environment API Keys
// ============================================================================
export function getEnvApiKey(provider) {
    const config = ALL_PROVIDERS[provider];
    if (!config)
        return undefined;
    return process.env[config.envKey];
}
export function checkApiKey(provider) {
    return !!getEnvApiKey(provider);
}
