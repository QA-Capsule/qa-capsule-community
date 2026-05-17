/**
 * web/js/ui.js
 * UI utilities: Notifications, Modals, and Theme management
 */

export function notify(message, type = 'success') {
    const container = document.getElementById('notification-container');
    if (!container) return alert(message);

    const toast = document.createElement('div');
    const bgColor = type === 'error' ? '#ff4444' : '#00C851';

    toast.style.cssText = `background-color: ${bgColor}; color: white; padding: 15px 20px; border-radius: 4px; box-shadow: 0 4px 6px rgba(0,0,0,0.3); font-weight: bold; font-size: 14px; transition: opacity 0.5s ease; margin-top: 10px; z-index: 99999;`;
    toast.innerHTML = `<span>${message}</span>`;
    container.appendChild(toast);
    setTimeout(() => { toast.style.opacity = '0'; setTimeout(() => toast.remove(), 500); }, 4000);
}

export function initTheme() { 
    if (localStorage.getItem('sre-theme') === 'dark') document.body.setAttribute('data-theme', 'dark'); 
}

export function toggleTheme() {
    const isDark = document.body.hasAttribute('data-theme');
    isDark ? document.body.removeAttribute('data-theme') : document.body.setAttribute('data-theme', 'dark');
    localStorage.setItem('sre-theme', isDark ? 'light' : 'dark');
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
    } else if (type === 'warning') {
        box.style.borderColor = 'var(--log-warn)';
        titleEl.style.color = 'var(--log-warn)';
    } else {
        box.style.borderColor = 'var(--text-main)';
        titleEl.style.color = 'var(--text-main)';
    }

    const confirmBtn = document.getElementById('custom-modal-confirm');
    confirmBtn.onclick = function () {
        closeModal();
        confirmCallback();
    };
    modal.style.display = 'flex';
}

export function showPromptModal(title, message, placeholder, confirmCallback, isPassword = false) {
    const modal = document.getElementById('custom-modal');
    const box = document.getElementById('custom-modal-box');
    const titleEl = document.getElementById('custom-modal-title');
    const inputEl = document.getElementById('custom-modal-input');

    titleEl.innerText = title;
    document.getElementById('custom-modal-message').innerText = message;

    inputEl.style.display = 'block';
    inputEl.type = isPassword ? 'password' : 'text';
    inputEl.placeholder = placeholder;
    inputEl.value = '';

    box.style.borderColor = 'var(--text-main)';
    titleEl.style.color = 'var(--text-main)';

    const confirmBtn = document.getElementById('custom-modal-confirm');
    confirmBtn.onclick = function () {
        const val = inputEl.value;
        if (val) {
            closeModal();
            confirmCallback(val);
        }
    };
    modal.style.display = 'flex';
    inputEl.focus();
}

export function closeModal() {
    document.getElementById('custom-modal').style.display = 'none';
}

// Initialize theme immediately upon module load
initTheme();