/**
 * Shared intelligent search / autocomplete for text fields.
 */
let activeAutocomplete = null;

function escapeHtml(s) {
    return String(s || '')
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;');
}

function highlightMatch(text, query) {
    if (!query) return escapeHtml(text);
    const lower = text.toLowerCase();
    const q = query.toLowerCase();
    const i = lower.indexOf(q);
    if (i < 0) return escapeHtml(text);
    return escapeHtml(text.slice(0, i)) +
        '<mark class="ac-highlight">' + escapeHtml(text.slice(i, i + q.length)) + '</mark>' +
        escapeHtml(text.slice(i + q.length));
}

/**
 * @param {Object} opts
 * @param {HTMLInputElement} opts.input
 * @param {HTMLElement} opts.list
 * @param {() => Array<{label:string, sublabel?:string, value?:any}>} opts.getSuggestions
 * @param {(item) => void} opts.onSelect
 * @param {number} [opts.minChars]
 * @param {number} [opts.maxItems]
 */
export function setupAutocomplete({ input, list, getSuggestions, onSelect, minChars = 0, maxItems = 12 }) {
    if (!input || !list) return;

    let highlightIndex = -1;

    const close = () => {
        list.style.display = 'none';
        list.innerHTML = '';
        highlightIndex = -1;
        if (activeAutocomplete === close) activeAutocomplete = null;
    };

    const render = () => {
        const query = input.value.trim();
        if (query.length < minChars) {
            close();
            return;
        }

        const items = (getSuggestions(query) || []).slice(0, maxItems);
        if (items.length === 0) {
            list.innerHTML = '<div class="autocomplete-empty">No matches</div>';
            list.style.display = 'block';
            return;
        }

        list.innerHTML = items.map((item, i) => `
            <div class="autocomplete-item${i === highlightIndex ? ' active' : ''}" data-index="${i}">
                <span class="autocomplete-item-label">${highlightMatch(item.label, query)}</span>
                ${item.sublabel ? `<span class="autocomplete-item-sub">${highlightMatch(item.sublabel, query)}</span>` : ''}
            </div>
        `).join('');

        list.querySelectorAll('.autocomplete-item').forEach(el => {
            el.addEventListener('mousedown', e => {
                e.preventDefault();
                const idx = parseInt(el.dataset.index, 10);
                const selected = items[idx];
                if (selected) {
                    onSelect(selected);
                    close();
                }
            });
        });

        list.style.display = 'block';
        activeAutocomplete = close;
    };

    input.addEventListener('input', () => {
        highlightIndex = -1;
        render();
    });

    input.addEventListener('focus', () => {
        if (input.value.trim().length >= minChars) render();
    });

    input.addEventListener('keydown', e => {
        const items = list.querySelectorAll('.autocomplete-item');
        if (!items.length || list.style.display === 'none') return;

        if (e.key === 'ArrowDown') {
            e.preventDefault();
            highlightIndex = Math.min(highlightIndex + 1, items.length - 1);
            items.forEach((el, i) => el.classList.toggle('active', i === highlightIndex));
        } else if (e.key === 'ArrowUp') {
            e.preventDefault();
            highlightIndex = Math.max(highlightIndex - 1, 0);
            items.forEach((el, i) => el.classList.toggle('active', i === highlightIndex));
        } else if (e.key === 'Enter' && highlightIndex >= 0) {
            e.preventDefault();
            items[highlightIndex].dispatchEvent(new MouseEvent('mousedown', { bubbles: true }));
        } else if (e.key === 'Escape') {
            close();
        }
    });

    document.addEventListener('click', e => {
        if (!input.contains(e.target) && !list.contains(e.target)) close();
    });

    return { close, refresh: render };
}
