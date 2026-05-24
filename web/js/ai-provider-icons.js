/**
 * Inline SVG icons for AI provider UI (no emoji, no external assets).
 */

export const AI_PROVIDERS = [
    {
        id: 'disabled',
        label: 'Disabled',
        defaultModel: '',
        defaultBaseUrl: '',
        defaultKeyEnv: 'OPENAI_API_KEY'
    },
    {
        id: 'openai',
        label: 'OpenAI',
        defaultModel: 'gpt-4o-mini',
        defaultBaseUrl: 'https://api.openai.com/v1',
        defaultKeyEnv: 'OPENAI_API_KEY'
    },
    {
        id: 'ollama',
        label: 'Ollama',
        defaultModel: 'llama3.2',
        defaultBaseUrl: 'http://localhost:11434',
        defaultKeyEnv: 'OPENAI_API_KEY'
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
        case 'openai':
            return logoOpenAI();
        case 'ollama':
            return logoOllama();
        default:
            return logoDisabled();
    }
}

function logoOpenAI() {
    return `<svg class="ai-svg ai-svg--logo ai-svg--openai" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
        <path fill="currentColor" d="M22.282 9.821a5.985 5.985 0 0 0-.516-4.938 6.046 6.046 0 0 0-6.51-2.9A6.065 6.065 0 0 0 4.98 4.18a5.985 5.985 0 0 0-3.998 2.9 6.046 6.046 0 0 0 .743 7.097 5.98 5.98 0 0 0 .51 4.911 6.051 6.051 0 0 0 6.516 2.9A5.985 5.985 0 0 0 13.26 24a6.056 6.056 0 0 0 5.772-4.206 5.99 5.99 0 0 0 3.997-2.9 6.056 6.056 0 0 0-.747-7.073zM13.26 22.43a4.476 4.476 0 0 1-2.876-1.04l.141-.081 4.779-2.758a.795.795 0 0 0 .392-.681v-6.737l2.02 1.168a.071.071 0 0 1 .038.052v5.583a4.504 4.504 0 0 1-4.494 4.494zm-9.66-4.125a4.47 4.47 0 0 1-.535-3.014l.142.085 4.783 2.759a.771.771 0 0 0 .78 0l5.843-3.369v2.332a.08.08 0 0 1-.033.062L9.74 19.95a4.5 4.5 0 0 1-6.14-1.646zM2.34 7.896a4.485 4.485 0 0 1 2.366-1.973V11.6a.766.766 0 0 0 .388.676l5.815 3.355-2.02 1.168a.076.076 0 0 1-.071 0l-4.83-2.786A4.504 4.504 0 0 1 2.34 7.872zm16.597 3.855-5.833-3.387L15.119 7.2a.076.076 0 0 1 .071 0l4.83 2.791a4.494 4.494 0 0 1-.676 8.105v-5.678a.79.79 0 0 0-.407-.667zm2.01-3.023-.142-.085-4.774-2.782a.776.776 0 0 0-.785 0L9.409 9.229V6.897a.066.066 0 0 1 .028-.061l4.83-2.787a4.5 4.5 0 0 1 6.68 4.66zm-12.64 4.135-2.02-1.164a.08.08 0 0 1-.038-.057V6.843a4.5 4.5 0 0 1 7.375-3.453l-.142.08-4.778 2.758a.795.795 0 0 0-.393.681zm1.49-3.208 2.602-1.5 2.607 1.5v3l-2.597 1.5-2.607-1.5z"/>
    </svg>`;
}

function logoOllama() {
    return `<svg class="ai-svg ai-svg--logo ai-svg--ollama" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
        <rect width="24" height="24" rx="6" fill="#1a1a1a"/>
        <path fill="#fff" d="M7.5 8.5c0-1.5 1.2-2.8 2.8-2.8.8 0 1.5.3 2 .9.5-.6 1.2-.9 2-.9 1.6 0 2.8 1.3 2.8 2.8 0 2.2-2.5 4.5-4.8 6.2-2.3-1.7-4.8-4-4.8-6.2z"/>
        <ellipse cx="10" cy="8.2" rx=".65" ry=".9" fill="#1a1a1a"/>
        <ellipse cx="14" cy="8.2" rx=".65" ry=".9" fill="#1a1a1a"/>
        <path fill="#fff" d="M11.2 10.2h1.6v.35a.8.8 0 0 1-.8.8h0a.8.8 0 0 1-.8-.8v-.35z"/>
    </svg>`;
}

function logoDisabled() {
    return `<svg class="ai-svg ai-svg--logo ai-svg--disabled" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">
        <circle cx="12" cy="12" r="9"/>
        <path d="M8 8l8 8"/>
    </svg>`;
}

export function getProviderMeta(providerId) {
    return AI_PROVIDERS.find(p => p.id === providerId) || AI_PROVIDERS[0];
}
