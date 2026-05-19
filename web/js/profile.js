/**
 * web/js/profile.js
 * Personal account settings and persisted user preferences
 */
import { fetchWithAuth, parseJwt } from './api.js';
import { notify } from './ui.js';
import { roleLabel } from './roles.js';

export let userPreferences = {
    theme: 'dark',
    default_status_filter: 'all',
    analytics_expanded: false
};

export function applyTheme(theme) {
    if (theme === 'dark') {
        document.body.setAttribute('data-theme', 'dark');
        localStorage.setItem('sre-theme', 'dark');
    } else {
        document.body.removeAttribute('data-theme');
        localStorage.setItem('sre-theme', 'light');
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

    const analytics = document.getElementById('analytics-view');
    if (analytics && userPreferences.analytics_expanded) {
        analytics.style.display = 'block';
    }
}

export async function loadUserPreferences() {
    try {
        const res = await fetchWithAuth('/api/me');
        if (!res.ok) return;
        const data = await res.json();
        if (data.preferences) applyPreferences(data.preferences);
        return data;
    } catch (e) {
        console.error('Could not load user preferences', e);
    }
}

export function loadProfileView() {
    const payload = parseJwt(localStorage.getItem('sre-jwt'));
    const roleEl = document.getElementById('profile-role');
    const usernameEl = document.getElementById('profile-username');
    if (roleEl && payload.role) roleEl.textContent = roleLabel(payload.role);
    if (usernameEl && payload.username) usernameEl.textContent = payload.username;

    fetchWithAuth('/api/me')
        .then(res => {
            if (!res.ok) throw new Error('Failed to load profile');
            return res.json();
        })
        .then(data => {
            const nameInput = document.getElementById('profile-fullname');
            if (nameInput) nameInput.value = data.fullname || '';

            const prefs = data.preferences || userPreferences;
            const themeSel = document.getElementById('pref-theme');
            const statusSel = document.getElementById('pref-status-filter');
            const analyticsChk = document.getElementById('pref-analytics-expanded');
            if (themeSel) themeSel.value = prefs.theme || 'dark';
            if (statusSel) statusSel.value = prefs.default_status_filter || 'all';
            if (analyticsChk) analyticsChk.checked = !!prefs.analytics_expanded;
        })
        .catch(() => notify('Could not load profile', 'error'));
}

export function saveProfile() {
    const fullname = document.getElementById('profile-fullname')?.value?.trim();
    if (!fullname) return notify('Display name is required', 'error');

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
