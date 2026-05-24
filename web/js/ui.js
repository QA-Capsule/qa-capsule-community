/**
 * web/js/ui.js
 * UI utilities: Notifications, Modals, and Theme management
 */

const THEME_STORAGE_KEY = 'sre-theme';

export function getStoredTheme() {
    const stored = localStorage.getItem(THEME_STORAGE_KEY);
    if (stored === 'dark' || stored === 'light') return stored;
    if (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) {
        return 'dark';
    }
    return 'light';
}

/** Apply light/dark theme to document (html + body) and persist. */
export function applyTheme(theme) {
    const next = theme === 'dark' ? 'dark' : 'light';
    document.documentElement.setAttribute('data-theme', next);
    if (next === 'dark') {
        document.body.setAttribute('data-theme', 'dark');
    } else {
        document.body.setAttribute('data-theme', 'light');
    }
    document.documentElement.style.colorScheme = next;
    localStorage.setItem(THEME_STORAGE_KEY, next);
    return next;
}

export function notify(message, type = 'success') {
    const container = document.getElementById('notification-container');
    if (!container) return alert(message);

    const toast = document.createElement('div');
    const bgColor = type === 'error' ? 'var(--log-fatal, #dc2626)' : 'var(--log-pass, #059669)';

    toast.style.cssText = `background-color: ${bgColor}; color: #fff; padding: 12px 18px; border-radius: var(--radius-sm, 8px); box-shadow: var(--shadow-md, 0 8px 24px rgba(0,0,0,0.2)); font-weight: 600; font-size: 13px; transition: opacity 0.5s ease; margin-top: 10px; z-index: 100001; max-width: min(420px, 90vw); line-height: 1.4;`;
    toast.innerHTML = `<span>${message}</span>`;
    container.appendChild(toast);
    setTimeout(() => { toast.style.opacity = '0'; setTimeout(() => toast.remove(), 500); }, 4000);
}

export function initTheme() {
    applyTheme(getStoredTheme());
}

const SIDEBAR_COLLAPSED_KEY = 'sre-sidebar-collapsed';

export function initSidebar() {
    const sidebar = document.querySelector('.sidebar');
    const btn = document.getElementById('sidebar-collapse-btn');
    if (!sidebar) return;
    const collapsed = localStorage.getItem(SIDEBAR_COLLAPSED_KEY) === '1';
    sidebar.classList.toggle('collapsed', collapsed);
    if (btn) {
        btn.setAttribute('aria-expanded', collapsed ? 'false' : 'true');
        btn.title = collapsed ? 'Expand sidebar' : 'Collapse sidebar';
    }
}

export function toggleSidebar() {
    const sidebar = document.querySelector('.sidebar');
    const btn = document.getElementById('sidebar-collapse-btn');
    if (!sidebar) return;
    const collapsed = !sidebar.classList.contains('collapsed');
    sidebar.classList.toggle('collapsed', collapsed);
    localStorage.setItem(SIDEBAR_COLLAPSED_KEY, collapsed ? '1' : '0');
    if (btn) {
        btn.setAttribute('aria-expanded', collapsed ? 'false' : 'true');
        btn.title = collapsed ? 'Expand sidebar' : 'Collapse sidebar';
    }
}

export function toggleTheme(e) {
    if (e?.stopPropagation) e.stopPropagation();
    const isDark = document.body.getAttribute('data-theme') === 'dark';
    const next = isDark ? 'light' : 'dark';
    applyTheme(next);
    if (typeof window.persistThemeFromToggle === 'function') {
        window.persistThemeFromToggle(next);
    }
    const analyticsView = document.getElementById('analytics-view');
    if (analyticsView && analyticsView.style.display !== 'none' && typeof window.loadAnalytics === 'function') {
        window.loadAnalytics(false);
    }
}

export function showConfirmModal(title, message, type, confirmCallback) {
    const modal = document.getElementById('custom-modal');
    const box = document.getElementById('custom-modal-box');
    const titleEl = document.getElementById('custom-modal-title');

    titleEl.innerText = title;
    document.getElementById('custom-modal-message').innerText = message;
    document.getElementById('custom-modal-input').style.display = 'none';

    if (type === 'danger') {
        box.style.borderColor = 'var(--log-fatal)';
        titleEl.style.color = 'var(--log-fatal)';
    } else {
        box.style.borderColor = 'var(--border-main)';
        titleEl.style.color = 'var(--text-main)';
    }

    modal.style.display = 'flex';
    const confirmBtn = document.getElementById('custom-modal-confirm');
    confirmBtn.onclick = () => {
        closeModal();
        if (confirmCallback) confirmCallback();
    };
}

export function showPromptModal(title, message, defaultValue, confirmCallback) {
    const modal = document.getElementById('custom-modal');
    const box = document.getElementById('custom-modal-box');
    const titleEl = document.getElementById('custom-modal-title');
    const inputEl = document.getElementById('custom-modal-input');

    titleEl.innerText = title;
    titleEl.style.color = 'var(--text-main)';
    box.style.borderColor = 'var(--border-main)';
    document.getElementById('custom-modal-message').innerText = message;
    inputEl.style.display = 'block';
    inputEl.value = defaultValue || '';

    modal.style.display = 'flex';
    const confirmBtn = document.getElementById('custom-modal-confirm');
    confirmBtn.onclick = () => {
        const val = inputEl.value;
        closeModal();
        if (confirmCallback) confirmCallback(val);
    };
}

export function closeModal() {
    const modal = document.getElementById('custom-modal');
    if (modal) modal.style.display = 'none';
}
