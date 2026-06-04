/**
 * AI provider metadata, model lists, and real monochrome SVG logos.
 * All logos use fill="currentColor" — same visual style as OpenAI.
 */

export const AI_PROVIDERS = [
    {
        id: 'disabled',
        label: 'Disabled',
        defaultModel: '',
        defaultBaseUrl: '',
        defaultKeyEnv: '',
        models: []
    },
    {
        id: 'openai',
        label: 'OpenAI',
        defaultModel: 'gpt-4o-mini',
        defaultBaseUrl: 'https://api.openai.com/v1',
        defaultKeyEnv: 'OPENAI_API_KEY',
        models: [
            { id: 'gpt-4o',        label: 'GPT-4o' },
            { id: 'gpt-4o-mini',   label: 'GPT-4o mini' },
            { id: 'gpt-4-turbo',   label: 'GPT-4 Turbo' },
            { id: 'gpt-3.5-turbo', label: 'GPT-3.5 Turbo' },
            { id: 'o1',            label: 'o1' },
            { id: 'o1-mini',       label: 'o1-mini' },
            { id: 'o3-mini',       label: 'o3-mini' },
        ]
    },
    {
        id: 'anthropic',
        label: 'Anthropic',
        defaultModel: 'claude-3-5-haiku-20241022',
        defaultBaseUrl: 'https://api.anthropic.com',
        defaultKeyEnv: 'ANTHROPIC_API_KEY',
        models: [
            { id: 'claude-opus-4-5',            label: 'Claude Opus 4.5' },
            { id: 'claude-sonnet-4-5',          label: 'Claude Sonnet 4.5' },
            { id: 'claude-3-5-sonnet-20241022', label: 'Claude 3.5 Sonnet' },
            { id: 'claude-3-5-haiku-20241022',  label: 'Claude 3.5 Haiku' },
            { id: 'claude-3-opus-20240229',     label: 'Claude 3 Opus' },
            { id: 'claude-3-haiku-20240307',    label: 'Claude 3 Haiku' },
        ]
    },
    {
        id: 'gemini',
        label: 'Google Gemini',
        defaultModel: 'gemini-2.0-flash',
        defaultBaseUrl: 'https://generativelanguage.googleapis.com/v1beta',
        defaultKeyEnv: 'GEMINI_API_KEY',
        models: [
            { id: 'gemini-2.0-flash',      label: 'Gemini 2.0 Flash' },
            { id: 'gemini-2.0-flash-lite', label: 'Gemini 2.0 Flash Lite' },
            { id: 'gemini-1.5-pro',        label: 'Gemini 1.5 Pro' },
            { id: 'gemini-1.5-flash',      label: 'Gemini 1.5 Flash' },
            { id: 'gemini-1.0-pro',        label: 'Gemini 1.0 Pro' },
        ]
    },
    {
        id: 'mistral',
        label: 'Mistral',
        defaultModel: 'mistral-small-latest',
        defaultBaseUrl: 'https://api.mistral.ai/v1',
        defaultKeyEnv: 'MISTRAL_API_KEY',
        models: [
            { id: 'mistral-large-latest',  label: 'Mistral Large' },
            { id: 'mistral-medium-latest', label: 'Mistral Medium' },
            { id: 'mistral-small-latest',  label: 'Mistral Small' },
            { id: 'codestral-latest',      label: 'Codestral' },
            { id: 'open-mistral-7b',       label: 'Open Mistral 7B' },
            { id: 'open-mixtral-8x7b',     label: 'Open Mixtral 8x7B' },
        ]
    },
    {
        id: 'groq',
        label: 'Groq',
        defaultModel: 'llama-3.1-8b-instant',
        defaultBaseUrl: 'https://api.groq.com/openai/v1',
        defaultKeyEnv: 'GROQ_API_KEY',
        models: [
            { id: 'llama-3.3-70b-versatile', label: 'Llama 3.3 70B' },
            { id: 'llama-3.1-70b-versatile', label: 'Llama 3.1 70B' },
            { id: 'llama-3.1-8b-instant',    label: 'Llama 3.1 8B Instant' },
            { id: 'mixtral-8x7b-32768',      label: 'Mixtral 8x7B' },
            { id: 'gemma2-9b-it',            label: 'Gemma2 9B' },
        ]
    },
    {
        id: 'openrouter',
        label: 'OpenRouter',
        defaultModel: 'openai/gpt-4o-mini',
        defaultBaseUrl: 'https://openrouter.ai/api/v1',
        defaultKeyEnv: 'OPENROUTER_API_KEY',
        models: [
            { id: 'openai/gpt-4o',                   label: 'OpenAI GPT-4o' },
            { id: 'openai/gpt-4o-mini',              label: 'OpenAI GPT-4o mini' },
            { id: 'anthropic/claude-3-5-sonnet',     label: 'Anthropic Claude 3.5 Sonnet' },
            { id: 'google/gemini-pro-1.5',           label: 'Google Gemini Pro 1.5' },
            { id: 'meta-llama/llama-3-70b-instruct', label: 'Meta Llama 3 70B' },
            { id: 'mistralai/mistral-7b-instruct',   label: 'Mistral 7B Instruct' },
        ]
    },
    {
        id: 'azure',
        label: 'Azure OpenAI',
        defaultModel: 'gpt-4o-mini',
        defaultBaseUrl: 'https://YOUR-RESOURCE.openai.azure.com/openai/deployments/YOUR-DEPLOYMENT',
        defaultKeyEnv: 'AZURE_OPENAI_API_KEY',
        models: [
            { id: 'gpt-4o',       label: 'GPT-4o' },
            { id: 'gpt-4o-mini',  label: 'GPT-4o mini' },
            { id: 'gpt-4-turbo',  label: 'GPT-4 Turbo' },
            { id: 'gpt-35-turbo', label: 'GPT-3.5 Turbo' },
        ]
    },
    {
        id: 'ollama',
        label: 'Ollama',
        defaultModel: 'llama3.2',
        defaultBaseUrl: 'http://localhost:11434',
        defaultKeyEnv: '',
        models: [
            { id: 'llama3.2',    label: 'Llama 3.2' },
            { id: 'llama3.1',    label: 'Llama 3.1' },
            { id: 'mistral',     label: 'Mistral' },
            { id: 'codellama',   label: 'Code Llama' },
            { id: 'phi3',        label: 'Phi-3' },
            { id: 'gemma2',      label: 'Gemma 2' },
            { id: 'deepseek-r1', label: 'DeepSeek R1' },
        ]
    }
];

export function iconAiInsight() {
    return `<svg class="ai-svg ai-svg--insight" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
        <path d="M12 2a4 4 0 0 1 4 4v1a3 3 0 0 1 3 3v1a2 2 0 0 1-2 2h-1.2a1 1 0 0 0-.8.4l-.6.8a1 1 0 0 1-1.6 0l-.6-.8a1 1 0 0 0-.8-.4H7a2 2 0 0 1-2-2v-1a3 3 0 0 1 3-3V6a4 4 0 0 1 4-4z"/>
        <path d="M9 18h6"/>
        <path d="M10 22h4"/>
        <circle cx="9" cy="8" r="1" fill="currentColor" stroke="none"/>
        <circle cx="15" cy="8" r="1" fill="currentColor" stroke="none"/>
    </svg>`;
}

export function logoForProvider(providerId) {
    switch (providerId) {
        case 'openai':     return logoOpenAI();
        case 'anthropic':  return logoAnthropic();
        case 'gemini':     return logoGemini();
        case 'mistral':    return logoMistral();
        case 'groq':       return logoGroq();
        case 'openrouter': return logoOpenRouter();
        case 'azure':      return logoAzure();
        case 'ollama':     return logoOllama();
        default:           return logoDisabled();
    }
}

/* ── Real monochrome logos — all use fill="currentColor" ────────── */

function logoOpenAI() {
    return `<svg class="ai-svg ai-svg--logo" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
        <path fill="currentColor" d="M22.282 9.821a5.985 5.985 0 0 0-.516-4.938 6.046 6.046 0 0 0-6.51-2.9A6.065 6.065 0 0 0 4.981 4.18a5.985 5.985 0 0 0-3.998 2.9 6.046 6.046 0 0 0 .743 7.097 5.98 5.98 0 0 0 .51 4.911 6.051 6.051 0 0 0 6.516 2.9A5.985 5.985 0 0 0 13.26 24a6.056 6.056 0 0 0 5.772-4.206 5.99 5.99 0 0 0 3.997-2.9 6.056 6.056 0 0 0-.747-7.073zM13.26 22.43a4.476 4.476 0 0 1-2.876-1.04l.141-.081 4.779-2.758a.795.795 0 0 0 .392-.681v-6.737l2.02 1.168a.071.071 0 0 1 .038.052v5.583a4.504 4.504 0 0 1-4.494 4.494zm-9.66-4.125a4.47 4.47 0 0 1-.535-3.014l.142.085 4.783 2.759a.771.771 0 0 0 .78 0l5.843-3.369v2.332a.08.08 0 0 1-.033.062L9.74 19.95a4.5 4.5 0 0 1-6.14-1.646zM2.34 7.896a4.485 4.485 0 0 1 2.366-1.973V11.6a.766.766 0 0 0 .388.676l5.815 3.355-2.02 1.168a.076.076 0 0 1-.071 0l-4.83-2.786A4.504 4.504 0 0 1 2.34 7.872zm16.597 3.855-5.833-3.387L15.119 7.2a.076.076 0 0 1 .071 0l4.83 2.791a4.494 4.494 0 0 1-.676 8.105v-5.678a.79.79 0 0 0-.407-.667zm2.01-3.023-.142-.085-4.774-2.782a.776.776 0 0 0-.785 0L9.409 9.229V6.897a.066.066 0 0 1 .028-.061l4.83-2.787a4.5 4.5 0 0 1 6.68 4.66zm-12.64 4.135-2.02-1.164a.08.08 0 0 1-.038-.057V6.843a4.5 4.5 0 0 1 7.375-3.453l-.142.08-4.778 2.758a.795.795 0 0 0-.393.681zm1.49-3.208 2.602-1.5 2.607 1.5v3l-2.597 1.5-2.607-1.5z"/>
    </svg>`;
}

function logoAnthropic() {
    /* Anthropic official icon — triangle with inner cut */
    return `<svg class="ai-svg ai-svg--logo" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
        <path fill="currentColor" d="M13.827 3.52h3.603L24 20h-3.603zm-7.258 0h3.603L16.737 20h-3.603z"/>
    </svg>`;
}

function logoGemini() {
    /* Google Gemini — four-pointed star (official shape) */
    return `<svg class="ai-svg ai-svg--logo" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
        <path fill="currentColor" d="M12 0C12 0 12.3 6.9 8.5 9.5 5.5 11.5 0 12 0 12s5.5.5 8.5 2.5C12.3 17.1 12 24 12 24s-.3-6.9 3.5-9.5C18.5 12.5 24 12 24 12s-5.5-.5-8.5-2.5C12.3 6.9 12 0 12 0z"/>
    </svg>`;
}

function logoMistral() {
    /* Mistral AI — official stacked-blocks pattern */
    return `<svg class="ai-svg ai-svg--logo" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
        <path fill="currentColor" d="M2 2h4v4H2zm6 0h4v4H8zm6 0h4v4h-4zm4 6h-4v4h4zm-4 6h4v4h-4zM8 8h4v4H8zM2 8h4v4H2zm0 6h4v4H2zm6 0h4v4H8z"/>
    </svg>`;
}

function logoGroq() {
    /* Groq — the official G-arc chip icon */
    return `<svg class="ai-svg ai-svg--logo" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
        <path fill="currentColor" d="M12 2a10 10 0 1 0 10 10A10.011 10.011 0 0 0 12 2zm0 18a8 8 0 1 1 8-8 8.009 8.009 0 0 1-8 8zm1-8.5V9h-2v3.5a.5.5 0 0 0 .5.5H15v-2h-2a.5.5 0 0 1 0-1z"/>
        <path fill="currentColor" d="M12 6a6 6 0 0 0-6 6h2a4 4 0 0 1 4-4V6z"/>
    </svg>`;
}

function logoOpenRouter() {
    /* OpenRouter — branching routes icon */
    return `<svg class="ai-svg ai-svg--logo" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
        <path fill="currentColor" d="M16 3h5v5l-1.9-1.9-4.6 4.6-1.4-1.4 4.6-4.6zM7 3H2v5l1.9-1.9 4.6 4.6 1.4-1.4L5.3 4.9zm5 10.4L7.4 18H2v-5l1.9 1.9L9.6 11l1.4 1.4zM17 18h5v-5l-1.9 1.9-5.7-5.7-1.4 1.4 5.7 5.7z"/>
    </svg>`;
}

function logoAzure() {
    /* Microsoft Azure — official "A" wave mark */
    return `<svg class="ai-svg ai-svg--logo" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
        <path fill="currentColor" d="M13.05 3H9l-5.8 9.5L7 19h4.2l-3.1-6.3L13.05 3zm2.6 0-5.4 9.3 3.85 4.4L22 18.8l-6.7-1.05L18.6 10 15.65 3z"/>
    </svg>`;
}

function logoOllama() {
    /* Ollama — stylized llama silhouette */
    return `<svg class="ai-svg ai-svg--logo" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
        <path fill="currentColor" d="M9 3a3 3 0 0 0-3 3c0 .88.38 1.67 1 2.22V9.5a4.5 4.5 0 0 0-3 4.24V21h2v-7.26A2.5 2.5 0 0 1 8.5 11.5h7a2.5 2.5 0 0 1 2.5 2.24V21h2v-7.26a4.5 4.5 0 0 0-3-4.24V8.22A3 3 0 1 0 15 3a3 3 0 0 0-2 .78A3 3 0 0 0 9 3zm0 2a1 1 0 1 1 0 2 1 1 0 0 1 0-2zm6 0a1 1 0 1 1 0 2 1 1 0 0 1 0-2zM9.5 9h5v1.5h-5z"/>
    </svg>`;
}

function logoDisabled() {
    return `<svg class="ai-svg ai-svg--logo" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
        <circle cx="12" cy="12" r="9" opacity=".4"/>
        <path d="M14.83 14.83 9.17 9.17" opacity=".4"/>
    </svg>`;
}

export function getProviderMeta(providerId) {
    return AI_PROVIDERS.find(p => p.id === providerId) || AI_PROVIDERS[0];
}
