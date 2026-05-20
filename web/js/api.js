/**
 * web/js/api.js
 * Core networking and authentication handlers
 */

let lastOfflineWarnAt = 0;
let apiReachable = true;
const FETCH_TIMEOUT_MS = 45000;

export function parseJwt(token) {
    try {
        return JSON.parse(atob(token.split('.')[1]));
    } catch (e) {
        return {};
    }
}

export function showLoginScreen() {
    apiReachable = true;
    window.__apiOffline = false;
    const banner = document.getElementById('api-offline-banner');
    if (banner) banner.style.display = 'none';

    const loginScreen = document.getElementById('login-screen');
    const appContainer = document.getElementById('app-container');
    const pwdScreen = document.getElementById('force-password-screen');
    const errEl = document.getElementById('login-error');

    if (pwdScreen) pwdScreen.style.display = 'none';
    if (appContainer) appContainer.style.display = 'none';
    if (loginScreen) loginScreen.style.display = 'flex';
    if (errEl) errEl.style.display = 'none';
}

export function performLogout() {
    localStorage.removeItem('sre-jwt');
    if (window.__incidentPollTimer) {
        clearInterval(window.__incidentPollTimer);
        window.__incidentPollTimer = null;
    }
    showLoginScreen();
}

export function isApiOffline() {
    return !!window.__apiOffline;
}

function markApiOffline() {
    if (!apiReachable) return;
    apiReachable = false;
    window.__apiOffline = true;
    const banner = document.getElementById('api-offline-banner');
    if (banner) banner.style.display = 'block';
    const now = Date.now();
    if (now - lastOfflineWarnAt > 8000) {
        lastOfflineWarnAt = now;
        console.warn('QA Capsule server unreachable. Start the backend or check the URL/port.');
    }
}

function markApiOnline() {
    if (apiReachable) return;
    apiReachable = true;
    window.__apiOffline = false;
    const banner = document.getElementById('api-offline-banner');
    if (banner) banner.style.display = 'none';
}

export function resetApiOfflineBanner() {
    markApiOnline();
}

export function describeApiFailure(status, offline) {
    if (offline) return 'Server unreachable. Start the backend (go run ./cmd/qacapsule) and open http://localhost:9000.';
    if (status === 504) return 'Server is busy (request timed out). Wait a few seconds and try again.';
    if (status === 401) return 'Session expired. Please sign in again.';
    if (status === 403) return 'Access denied for this action.';
    if (status === 404) return 'API endpoint not found.';
    if (status >= 500) return 'Server error. Check backend logs.';
    if (status) return `Request failed (HTTP ${status}).`;
    return 'Request failed.';
}

/** Normalize API payloads that may be a raw array or wrapped. */
export function asArray(value) {
    if (Array.isArray(value)) return value;
    return [];
}

function isSyntheticNetworkResponse(res, data) {
    return res.status === 503 && data && typeof data === 'object' && data.error === 'network_unavailable';
}

function isSyntheticTimeoutResponse(res, data) {
    return res.status === 504 && data && typeof data === 'object' && data.error === 'request_timeout';
}

/**
 * Parse JSON from a fetch Response.
 * Only marks the server offline for real network failures, not HTTP 4xx/5xx API errors.
 */
export async function parseApiJson(res) {
    let text = '';
    try {
        text = await res.text();
    } catch {
        markApiOffline();
        return { ok: false, offline: true, data: null, status: 0 };
    }

    let data = null;
    if (text) {
        try {
            data = JSON.parse(text);
        } catch {
            data = null;
        }
    }

    if (isSyntheticNetworkResponse(res, data)) {
        markApiOffline();
        return { ok: false, offline: true, data: null, status: res.status };
    }

    if (isSyntheticTimeoutResponse(res, data)) {
        return { ok: false, offline: false, data: null, status: 504 };
    }

    // Any HTTP response from the backend means the server is reachable.
    markApiOnline();

    if (!res.ok) {
        return { ok: false, offline: false, data, status: res.status };
    }
    return { ok: true, offline: false, data, status: res.status };
}

/** Lightweight health check (no auth). Clears a stale offline banner after refresh. */
export async function pingApiServer() {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), 5000);
    try {
        const res = await fetch('/api/sso/status', { method: 'GET', signal: controller.signal });
        clearTimeout(timeoutId);
        if (res.ok) markApiOnline();
        else markApiOffline();
        return res.ok;
    } catch {
        clearTimeout(timeoutId);
        markApiOffline();
        return false;
    }
}

export function fetchWithAuth(url, opts = {}) {
    const token = localStorage.getItem('sre-jwt');

    if (!token) {
        showLoginScreen();
        return Promise.reject('No authentication token found.');
    }

    const headers = { ...opts.headers, 'Authorization': `Bearer ${token}` };
    const method = (opts.method || 'GET').toUpperCase();
    if (method !== 'GET' && method !== 'HEAD') {
        headers['Content-Type'] = headers['Content-Type'] || 'application/json';
    }

    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), FETCH_TIMEOUT_MS);

    return fetch(url, { ...opts, headers, signal: controller.signal })
        .then(res => {
            clearTimeout(timeoutId);
            if (res.status === 401) performLogout();
            else markApiOnline();
            return res;
        })
        .catch(async () => {
            clearTimeout(timeoutId);
            const up = await pingApiServer();
            const error = up ? 'request_timeout' : 'network_unavailable';
            const status = up ? 504 : 503;
            return new Response(JSON.stringify({ error }), {
                status,
                statusText: status === 504 ? 'Gateway Timeout' : 'Network Error',
                headers: { 'Content-Type': 'application/json' }
            });
        });
}
