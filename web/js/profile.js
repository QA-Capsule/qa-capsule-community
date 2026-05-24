/**
 * web/js/profile.js
 * Personal account settings and persisted user preferences
 */
import { fetchWithAuth, parseJwt, parseApiJson } from './api.js';
import { notify } from './ui.js';
import { roleLabel, normalizeRole } from './roles.js';
import { setSelectedCurrency, currencySymbols } from './settings.js';

export const PREF_DEFAULT_RANGE_KEY = 'sre-pref-default-range';
export const PREF_CURRENCY_KEY = 'sre-pref-currency';
export const PREF_ALERT_SOUNDS_KEY = 'sre-alert-sounds';

const VALID_DEFAULT_RANGE_PRESETS = [
    '5m', '15m', '30m', '1h', '6h', '24h', '7d', '30d', 'today', 'yesterday', 'all'
];

export let userPreferences = {
    theme: 'dark',
    default_status_filter: 'all',
    analytics_expanded: false
};

/** "Achraf KHABAR" → "AK" */
export function initialsFromDisplayName(name) {
    const cleaned = String(name ?? '').trim().replace(/\s+/g, ' ');
    if (!cleaned) return '?';

    const parts = cleaned.split(' ').filter(Boolean);
    if (parts.length >= 2) {
        const first = parts[0][0] ?? '';
        const last = parts[parts.length - 1][0] ?? '';
        return (first + last).toUpperCase();
    }

    const token = parts[0];
    if (token.length >= 2) return token.slice(0, 2).toUpperCase();
    return token[0].toUpperCase();
}

export function displayNameFromJwtUsername(username) {
    const u = String(username ?? '').trim();
    if (!u) return '';
    const local = u.includes('@') ? u.split('@')[0] : u;
    return local
        .replace(/[._-]+/g, ' ')
        .replace(/\b\w/g, (c) => c.toUpperCase());
}

export function resolveProfileDisplayName({ fullname, username }) {
    const fn = String(fullname ?? '').trim();
    if (fn) return fn;
    return displayNameFromJwtUsername(username);
}

export function isValidDefaultRangePreset(preset) {
    return VALID_DEFAULT_RANGE_PRESETS.includes(preset);
}

export function applyPreferredDefaultRange(preset) {
    if (!isValidDefaultRangePreset(preset)) return;
    try {
        localStorage.setItem(PREF_DEFAULT_RANGE_KEY, preset);
    } catch (_) { /* quota */ }
}

export function bootstrapDashboardRangeFromPreferences() {
    try {
        if (localStorage.getItem('sre-dashboard-range')) return;
        const preferred = localStorage.getItem(PREF_DEFAULT_RANGE_KEY) || '15m';
        const preset = isValidDefaultRangePreset(preferred) ? preferred : '15m';
        localStorage.setItem('sre-dashboard-range', JSON.stringify({ preset }));
    } catch (_) { /* private mode */ }
}

export function isAlertSoundsEnabled() {
    return localStorage.getItem(PREF_ALERT_SOUNDS_KEY) === '1';
}

export function playCriticalAlertSound() {
    if (!isAlertSoundsEnabled()) return;
    try {
        const Ctx = window.AudioContext || window.webkitAudioContext;
        if (!Ctx) return;
        const ctx = new Ctx();
        const osc = ctx.createOscillator();
        const gain = ctx.createGain();
        osc.connect(gain);
        gain.connect(ctx.destination);
        osc.frequency.value = 880;
        gain.gain.setValueAtTime(0.08, ctx.currentTime);
        gain.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + 0.28);
        osc.start();
        osc.stop(ctx.currentTime + 0.28);
        osc.onended = () => ctx.close();
    } catch (_) { /* autoplay policy */ }
}

export function syncApplicationCurrency(code) {
    const currency = code || 'USD';
    window.selectedCurrency = currency;
    setSelectedCurrency(currency);
    try {
        localStorage.setItem(PREF_CURRENCY_KEY, currency);
        localStorage.setItem('selected-currency', currency);
    } catch (_) { /* quota */ }

    const finopsSel = document.getElementById('finops-currency');
    if (finopsSel && finopsSel.value !== currency) finopsSel.value = currency;

    const prefSel = document.getElementById('pref-currency');
    if (prefSel && prefSel.value !== currency) prefSel.value = currency;
}

function renderProfileHeader({ fullname, username, role }) {
    const displayName = resolveProfileDisplayName({ fullname, username });
    const initials = initialsFromDisplayName(displayName);

    const initialsEl = document.getElementById('profile-avatar-initials');
    const nameEl = document.getElementById('profile-display-name');
    const badgeEl = document.getElementById('profile-role-badge');
    const roleEl = document.getElementById('profile-role');

    if (initialsEl) initialsEl.textContent = initials;
    if (nameEl) nameEl.textContent = displayName || username || '—';
    if (roleEl && role) roleEl.textContent = roleLabel(role);
    if (badgeEl && role) {
        const norm = normalizeRole(role);
        badgeEl.textContent = roleLabel(role);
        badgeEl.className = `profile-role-badge role-${norm || 'observer'}`;
    }
}

export function applyTheme(theme) {
    if (theme === 'dark') {
        document.body.setAttribute('data-theme', 'dark');
        localStorage.setItem('sre-theme', 'dark');
    } else {
        document.body.removeAttribute('data-theme');
        localStorage.setItem('sre-theme', 'light');
    }
    const analyticsView = document.getElementById('analytics-view');
    if (analyticsView && analyticsView.style.display !== 'none' && typeof window.loadAnalytics === 'function') {
        window.loadAnalytics(false);
    }
}

export function applyPreferences(prefs) {
    if (!prefs) return;
    userPreferences = { ...userPreferences, ...prefs };
    applyTheme(userPreferences.theme || 'dark');

    if (userPreferences.default_status_filter && window.setStatusFilter) {
        window.statusFilter = userPreferences.default_status_filter;
        window.setStatusFilter(userPreferences.default_status_filter);
    }

    if (typeof window.loadAnalyticsLayoutFromPrefs === 'function') {
        window.loadAnalyticsLayoutFromPrefs();
    }

    const analytics = document.getElementById('analytics-view');
    if (analytics && userPreferences.analytics_expanded) {
        analytics.style.display = 'block';
        if (typeof window.loadAnalytics === 'function') window.loadAnalytics(false);
    }
}

export async function loadUserPreferences() {
    try {
        const savedCur = localStorage.getItem(PREF_CURRENCY_KEY) || localStorage.getItem('selected-currency');
        if (savedCur) syncApplicationCurrency(savedCur);

        const res = await fetchWithAuth('/api/me');
        const { ok, data } = await parseApiJson(res);
        if (!ok || !data) return;
        if (data.preferences) applyPreferences(data.preferences);
        return data;
    } catch {
        /* offline — use local theme only */
    }
}

function hydrateProfileFormFields(prefs) {
    const themeSel = document.getElementById('pref-theme');
    const statusSel = document.getElementById('pref-status-filter');
    const analyticsChk = document.getElementById('pref-analytics-expanded');
    const rangeSel = document.getElementById('pref-default-range');
    const currencySel = document.getElementById('pref-currency');
    const soundsChk = document.getElementById('pref-alert-sounds');

    if (themeSel) themeSel.value = prefs.theme || 'dark';
    if (statusSel) statusSel.value = prefs.default_status_filter || 'all';
    if (analyticsChk) analyticsChk.checked = !!prefs.analytics_expanded;

    const storedRange = localStorage.getItem(PREF_DEFAULT_RANGE_KEY) || '15m';
    if (rangeSel) rangeSel.value = isValidDefaultRangePreset(storedRange) ? storedRange : '15m';

    const storedCurrency = localStorage.getItem(PREF_CURRENCY_KEY) || window.selectedCurrency || 'USD';
    if (currencySel) currencySel.value = storedCurrency;
    syncApplicationCurrency(storedCurrency);

    if (soundsChk) soundsChk.checked = isAlertSoundsEnabled();
}

export function loadProfileView() {
    const payload = parseJwt(localStorage.getItem('sre-jwt'));
    const usernameEl = document.getElementById('profile-username');
    if (usernameEl && payload.username) usernameEl.textContent = payload.username;
    renderProfileHeader({
        fullname: '',
        username: payload.username,
        role: payload.role
    });

    hydrateProfileFormFields(userPreferences);

    fetchWithAuth('/api/me')
        .then(res => parseApiJson(res))
        .then(({ ok, data }) => {
            if (!ok || !data) throw new Error('Failed to load profile');
            const nameInput = document.getElementById('profile-fullname');
            if (nameInput) nameInput.value = data.fullname || '';

            renderProfileHeader({
                fullname: data.fullname,
                username: data.username ?? payload.username,
                role: data.role ?? payload.role
            });

            const prefs = data.preferences || userPreferences;
            hydrateProfileFormFields(prefs);
        })
        .catch(() => notify('Could not load profile', 'error'));
}

export function onProfileCurrencyChange() {
    const code = document.getElementById('pref-currency')?.value || 'USD';
    syncApplicationCurrency(code);
    if (document.getElementById('view-finops')?.classList.contains('active') && window.refreshFinOpsKPIs) {
        window.refreshFinOpsKPIs();
    }
    if (typeof window.updateCurrencyDisplay === 'function') {
        window.updateCurrencyDisplay();
    }
}

export function saveProfile() {
    const fullname = document.getElementById('profile-fullname')?.value?.trim();
    if (!fullname) return notify('Display name is required', 'error');

    const defaultRange = document.getElementById('pref-default-range')?.value || '15m';
    applyPreferredDefaultRange(defaultRange);

    const currency = document.getElementById('pref-currency')?.value || 'USD';
    syncApplicationCurrency(currency);

    const alertSounds = !!document.getElementById('pref-alert-sounds')?.checked;
    try {
        localStorage.setItem(PREF_ALERT_SOUNDS_KEY, alertSounds ? '1' : '0');
    } catch (_) { /* quota */ }

    const preferences = {
        theme: document.getElementById('pref-theme')?.value || 'dark',
        default_status_filter: document.getElementById('pref-status-filter')?.value || 'all',
        analytics_expanded: !!document.getElementById('pref-analytics-expanded')?.checked
    };

    Promise.all([
        fetchWithAuth('/api/me', { method: 'PUT', body: JSON.stringify({ fullname }) }),
        fetchWithAuth('/api/me/preferences', { method: 'PUT', body: JSON.stringify(preferences) })
    ])
        .then(([profileRes, prefsRes]) => {
            if (!profileRes.ok || !prefsRes.ok) throw new Error('Save failed');
            return prefsRes.json();
        })
        .then(prefs => {
            applyPreferences(prefs);
            renderProfileHeader({
                fullname,
                username: parseJwt(localStorage.getItem('sre-jwt')).username,
                role: parseJwt(localStorage.getItem('sre-jwt')).role
            });
            notify('Profile saved', 'success');
        })
        .catch(() => notify('Failed to save profile', 'error'));
}

export function changeOwnPassword() {
    const current = document.getElementById('profile-current-password')?.value;
    const next = document.getElementById('profile-new-password')?.value;
    const confirm = document.getElementById('profile-confirm-password')?.value;

    if (!current || !next) return notify('Fill in all password fields', 'error');
    if (next !== confirm) return notify('New passwords do not match', 'error');
    if (next.length < 8) return notify('Password must be at least 8 characters', 'error');

    fetchWithAuth('/api/me/password', {
        method: 'PUT',
        body: JSON.stringify({ current_password: current, new_password: next })
    })
        .then(res => {
            if (res.status === 403) throw new Error('wrong');
            if (!res.ok) throw new Error('failed');
            document.getElementById('profile-current-password').value = '';
            document.getElementById('profile-new-password').value = '';
            document.getElementById('profile-confirm-password').value = '';
            notify('Password updated', 'success');
        })
        .catch(err => {
            if (err.message === 'wrong') notify('Current password is incorrect', 'error');
            else notify('Failed to update password', 'error');
        });
}

export function persistThemeFromToggle(theme) {
    const preferences = {
        theme: theme === 'dark' ? 'dark' : 'light',
        default_status_filter: userPreferences.default_status_filter || 'all',
        analytics_expanded: userPreferences.analytics_expanded || false
    };
    fetchWithAuth('/api/me/preferences', { method: 'PUT', body: JSON.stringify(preferences) })
        .then(res => res.ok ? res.json() : null)
        .then(prefs => { if (prefs) userPreferences = prefs; })
        .catch(() => {});
}

export { currencySymbols };
