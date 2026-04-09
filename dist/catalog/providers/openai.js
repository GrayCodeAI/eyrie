export const OPENAI_MODELS = [
    {
        id: 'gpt-4o',
        input_price_per_1m: 5,
        output_price_per_1m: 15,
        context_window: 128000,
        max_output: 16000,
        server_tools: ['web_search'],
    },
    {
        id: 'gpt-4o-mini',
        input_price_per_1m: 0.15,
        output_price_per_1m: 0.6,
        context_window: 128000,
        max_output: 16000,
        server_tools: ['web_search'],
    },
];
