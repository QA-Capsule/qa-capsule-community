/**
 * Self-Healing Hub
 */
import { fetchWithAuth, parseApiJson, parseJwt } from './api.js';
import { notify } from './ui.js';
import { canManageHealing, canDeleteIncidents } from './roles.js';

let currentRole = '';
const selected = new Set(); // incident ids currently checked

/* ── Framework / category metadata ─────────────────────────────────────── */
const FW = {
    robotframework: { label: 'Robot',      color: '#e67e22' },
    playwright:     { label: 'Playwright', color: '#7c5cfc' },
    cypress:        { label: 'Cypress',    color: '#17a974' },
    selenium:       { label: 'Selenium',   color: '#2e86c1' },
    pytest:         { label: 'Pytest',     color: '#0097a7' },
    newman:         { label: 'Newman',     color: '#c0392b' },
    jest:           { label: 'Jest',       color: '#c77b00' },
    junit:          { label: 'JUnit',      color: '#6d4c41' },
};
const FW_DEF = { label: 'Unknown', color: '#7f8c8d' };

const CAT = {
    timeout:        { label: 'Timeout',   color: '#d4a017' },
    locator:        { label: 'Locator',   color: '#c0392b' },
    assertion:      { label: 'Assertion', color: '#e67e22' },
    network:        { label: 'Network',   color: '#2980b9' },
    script_failure: { label: 'Script',    color: '#8e44ad' },
    unknown:        { label: 'Unknown',   color: '#95a5a6' },
};

const fw  = r => FW[(r || '').toLowerCase().replace(/[^a-z]/g, '')] || FW_DEF;
const cat = r => CAT[(r || '').toLowerCase().replace(/[^a-z_]/g, '')] || CAT.unknown;
const esc = s => String(s || '').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');
const ago = s => { if (!s) return ''; const m = Math.floor((Date.now() - new Date(s)) / 60000); return m < 2 ? 'just now' : m < 60 ? `${m}m ago` : Math.floor(m / 60) < 24 ? `${Math.floor(m/60)}h ago` : `${Math.floor(m/1440)}d ago`; };

/** Build search variants for the healed locator (Robot, Playwright, quoted forms). */
function locatorHighlightVariants(healed) {
    if (!healed) return [];
    const v = new Set([healed]);
    v.add(`'${healed}'`);
    v.add(`"${healed}"`);
    if (healed.startsWith('#')) {
        const id = healed.slice(1);
        v.add(`css=${healed}`);
        v.add(`id=${id}`);
    }
    return [...v].sort((a, b) => b.length - a.length);
}

const LOCATOR_TOKEN_RE = /^(#[\w-]+|css=[^\s,]+|id=[^\s,]+|xpath=[^\s,]+|\$\{?\w+\}?|\$[A-Z_]+|[\w-]+=[^\s,]+)$/;

function looksLikeLocatorToken(s) {
    if (!s) return false;
    return LOCATOR_TOKEN_RE.test(s.trim());
}

/** Extract healed selector from AI explanation text (backticks, quotes, plain). */
function extractHealedFromExplanation(explanation) {
    if (!explanation) return '';
    const patterns = [
        /replaced with(?:\s+the correct(?:\s+one|\s+locator|\s+selector)?)?\s+`([^`]+)`/i,
        /replaced with(?:\s+the correct(?:\s+one|\s+locator|\s+selector)?)?\s+'([^']+)'/i,
        /replaced with(?:\s+the correct(?:\s+one|\s+locator|\s+selector)?)?\s+"([^"]+)"/i,
        /replaced with(?:\s+the correct(?:\s+one|\s+locator|\s+selector)?)?\s+(#[\w-]+|css=[^\s,.]+|id=[^\s,.]+)/i,
        /changed to(?:\s+the correct(?:\s+one|\s+locator|\s+selector)?)?\s+`([^`]+)`/i,
        /changed to(?:\s+the correct(?:\s+one|\s+locator|\s+selector)?)?\s+'([^']+)'/i,
        /changed to(?:\s+the correct(?:\s+one|\s+locator|\s+selector)?)?\s+"([^"]+)"/i,
        /changed to(?:\s+the correct(?:\s+one|\s+locator|\s+selector)?)?\s+(#[\w-]+|css=[^\s,.]+|id=[^\s,.]+)/i,
        /correct selector\s+`([^`]+)`/i,
        /(?:fixed|healed)\s+selector:?\s*[`'"]([^`'"]+)[`'"]/i,
        /(?:use|using)\s+[`'"]([^`'"]+)[`'"]\s+instead/i,
        /selector\s+`([^`]+)`/i,
    ];
    for (const re of patterns) {
        const m = explanation.match(re);
        if (m?.[1]?.trim() && looksLikeLocatorToken(m[1].trim())) return m[1].trim();
    }
    for (const m of explanation.matchAll(/`([^`]+)`/g)) {
        const s = m[1].trim();
        if (!s || s.startsWith('${')) continue;
        if (looksLikeLocatorToken(s)) return s;
    }
    return '';
}

/** Extract broken selector from AI explanation when API omits original_locator. */
function extractOriginalFromExplanation(explanation) {
    if (!explanation) return '';
    const patterns = [
        /which was\s+(\$\{?\w+\}?|\$[A-Z_]+|#[\w-]+|css=[^\s,.]+|id=[^\s,.]+)/i,
        /incorrect locator(?:\s+for[^,.]+)?\s*,?\s*which was\s+(\$\{?\w+\}?|\$[A-Z_]+|#[\w-]+|css=[^\s,.]+|id=[^\s,.]+)/i,
        /broken locator[^.]*?\bwas\s+(\$\{?\w+\}?|\$[A-Z_]+|#[\w-]+|css=[^\s,.]+|id=[^\s,.]+)/i,
        /(?:from|replaced)\s+`([^`]+)`\s+(?:to|with)/i,
    ];
    for (const re of patterns) {
        const m = explanation.match(re);
        if (m?.[1]?.trim() && looksLikeLocatorToken(m[1].trim())) return m[1].trim();
    }
    return '';
}

function extractLocatorFromChangeLine(line) {
    const trimmed = String(line || '').trim();
    if (!trimmed) return '';
    const action = trimmed.match(/^\s*(?:Click|Fill Text|Get Element|Tap|Type Text|Wait For Elements State)\s+(\S+)/i);
    if (action?.[1] && looksLikeLocatorToken(action[1])) return action[1];
    const tokens = trimmed.split(/\s+/);
    const last = tokens[tokens.length - 1];
    return looksLikeLocatorToken(last) ? last : trimmed;
}

const LOCATOR_ACTION_RE = /^\s*(Click|Get Element|Tap|Wait For Elements State)\s+(\S+)/i;
const FILL_ACTION_RE = /^\s*(Fill Text|Type Text)\s+(\S+)/i;

function robotVarName(token) {
    const m = String(token || '').trim().match(/^\$[\{]?([A-Za-z_]\w*)\}?$/);
    return m ? m[1] : '';
}

/** All spellings of a Robot variable reference (${X}, $X, $IX typo). */
function originalLocatorVariants(original) {
    if (!original) return [];
    const out = new Set([original]);
    const name = robotVarName(original);
    if (name) {
        out.add(`\${${name}}`);
        out.add(`$${name}`);
        out.add(`$I${name}`);
    } else if (/^\$I([A-Z_]+)$/.test(original)) {
        const n = original.slice(2);
        out.add(`\${${n}}`);
        out.add(`$${n}`);
    }
    return [...out];
}

function lineIncludesAny(line, terms) {
    return terms.some(t => t && line.includes(t));
}

function normalizeChangeLine(line) {
    return String(line || '').replace(/\s+/g, ' ').trim();
}

function linesEquivalent(a, b) {
    return normalizeChangeLine(a) === normalizeChangeLine(b);
}

function locatorTokensOnActionLine(line) {
    const m = line.match(LOCATOR_ACTION_RE) || line.match(FILL_ACTION_RE);
    return m?.[2]?.trim() || '';
}

function isFormFieldLocator(tok) {
    return /^id=(username|password|email|user|pass|login)$/i.test(tok || '');
}

function diffActionLocators(before, after) {
    const oldTok = locatorTokensOnActionLine(before);
    const newTok = locatorTokensOnActionLine(after);
    if (oldTok && newTok && oldTok !== newTok) return { oldTok, newTok };
    return { oldTok: oldTok || '', newTok: newTok || '' };
}

function findClickLocatorInCode(code) {
    for (const line of (code || '').split('\n')) {
        const m = line.match(LOCATOR_ACTION_RE);
        if (m && /^\$\{?[A-Za-z_]\w*\}?$/.test(m[2].trim())) return m[2].trim();
    }
    return '';
}

function shouldReplaceRobotVar(varTok, original, explanation) {
    const name = robotVarName(varTok);
    if (!name) return false;
    const expl = explanation || '';
    if (original && originalLocatorVariants(original).some(v => v === varTok || robotVarName(v) === name)) return true;
    if (expl.includes(varTok) || expl.includes(`\${${name}}`) || expl.includes(`$${name}`)) return true;
    if (new RegExp(`\\b${name}\\b`, 'i').test(expl)) return true;
    if (/broken|incorrect|outdated|wrong/i.test(expl) && /submit|click|button|locator|selector/i.test(expl)) {
        return /submit|broken|btn|click/i.test(name);
    }
    return false;
}

function findOriginalInCode(code, fallback, explanation) {
    if (fallback) {
        for (const v of originalLocatorVariants(fallback)) {
            if (code.includes(v)) return v;
        }
    }
    const clickVar = findClickLocatorInCode(code);
    if (clickVar && shouldReplaceRobotVar(clickVar, fallback, explanation)) return clickVar;
    return fallback || extractOriginalFromExplanation(explanation);
}

function applyFixPreview(code, original, healed, explanation) {
    if (!code || !healed || original === healed) return code || '';

    return code.split('\n').map(line => {
        const m = line.match(LOCATOR_ACTION_RE);
        if (!m) return line;

        const tok = m[2].trim();
        if (line.includes(healed)) return line;

        for (const t of originalLocatorVariants(original).sort((a, b) => b.length - a.length)) {
            if (t && line.includes(t)) return line.split(t).join(healed);
        }

        if (/^\$\{?[A-Za-z_]\w*\}?$/.test(tok) && shouldReplaceRobotVar(tok, original, explanation)) {
            return line.split(tok).join(healed);
        }
        return line;
    }).join('\n');
}

function highlightFixToken(line, token) {
    if (!token || !line.includes(token)) return esc(line);
    return line.split(token).map(esc).join(fixHighlightSpan(token, 'fix'));
}

function getHighlightTokenForLine(lineIndex, line, srcLine, healed, data) {
    const lineNo = lineIndex + 1;
    for (const h of (Array.isArray(data?.fix_highlights) ? data.fix_highlights : [])) {
        if (Number(h.line) === lineNo) return (h.token || healed || '').trim();
    }
    const action = line.match(LOCATOR_ACTION_RE);
    if (!action || !healed) return '';
    if (line.includes(healed)) return healed;
    if (srcLine !== undefined && srcLine !== line) {
        return (healed && line.includes(healed)) ? healed : action[2].trim();
    }
    return '';
}

function renderHighlightedLines(displayCode, sourceCode, healed, data) {
    const srcLines = (sourceCode || '').split('\n');
    return displayCode.split('\n').map((line, i) => {
        const token = getHighlightTokenForLine(i, line, srcLines[i], healed, data);
        const rowCls = token ? 'hc-code-line hc-code-line--fix' : 'hc-code-line';
        const content = token ? highlightFixToken(line, token) : esc(line);
        return `<div class="${rowCls}"><span class="hc-code-line__n">${i + 1}</span><span class="hc-code-line__t">${content}</span></div>`;
    }).join('');
}

function renderFixCodeBlock(code, data, id) {
    if (!code) return '';
    const expl = data?.explanation || '';
    const locs = resolveFixLocators(data);
    const sourceCode = (data?.source_code || '').trim() || code;
    let displayCode = code;
    if (locs.healed) {
        if (sourceCode === code || !code.includes(locs.healed)) {
            displayCode = applyFixPreview(sourceCode, locs.original, locs.healed, expl) || code;
        }
    }
    const changes = renderChangeLines(data.change_lines, locs.original, locs.healed);
    const lines = renderHighlightedLines(displayCode, sourceCode, locs.healed, data);
    return `<div class="hc-panel__code">
        <div class="hc-panel__code-head"><strong>Code fix</strong><button class="hc-panel__copy" onclick="window._copyFix(${id})">${ICON.copy} Copy</button></div>
        ${changes}
        <div class="hc-code-lines" role="presentation">${lines}</div>
    </div>`;
}

function inferHealedLocatorFromCode(code, original) {
    if (!code || !original) return '';
    const origTerms = originalLocatorVariants(original);
    if (lineIncludesAny(code, origTerms)) return '';

    const counts = new Map();
    for (const line of code.split('\n')) {
        const m = line.match(LOCATOR_ACTION_RE);
        if (!m) continue;
        const tok = m[2].trim();
        if (!looksLikeLocatorToken(tok)) continue;
        if (lineIncludesAny(tok, origTerms)) continue;
        if (/^\$\{?\w+\}?$/.test(tok)) continue;
        if (isFormFieldLocator(tok)) continue;
        counts.set(tok, (counts.get(tok) || 0) + 1);
    }
    let best = '';
    let bestScore = -1;
    for (const [tok, n] of counts) {
        let score = n * 10;
        if (tok.startsWith('#')) score += 5;
        if (tok.startsWith('css=')) score += 4;
        if (score > bestScore || (score === bestScore && tok.length > best.length)) {
            best = tok;
            bestScore = score;
        }
    }
    return best;
}

function buildHealedSearchTerms(healed) {
    if (!healed) return [];
    const terms = [healed];
    if (healed.startsWith('#')) {
        terms.push(`css=${healed}`, `id=${healed.slice(1)}`);
    }
    for (const v of [...terms]) {
        terms.push(`Click    ${v}`, `Click\t${v}`, `Click ${v}`, `Get Element    ${v}`, `Get Element ${v}`);
    }
    return [...new Set(terms)];
}

function isDarkTheme() {
    return document.documentElement.getAttribute('data-theme') === 'dark'
        || document.body.getAttribute('data-theme') === 'dark';
}

function fixHighlightSpan(term, kind) {
    const dark = isDarkTheme();
    const cls = kind === 'old' ? 'hc-tok hc-tok--old hc-code-fix-old' : 'hc-tok hc-tok--fix hc-code-fix-hl';
    const style = kind === 'old'
        ? (dark ? 'background:rgba(251,146,60,0.25);color:#fdba74;text-decoration:line-through;padding:1px 4px;border-radius:3px'
            : 'background:#ffedd5;color:#c2410c;text-decoration:line-through;padding:1px 4px;border-radius:3px')
        : (dark ? 'background:rgba(34,197,94,0.45);color:#dcfce7;font-weight:800;padding:1px 5px;border-radius:3px'
            : 'background:#fecaca;color:#991b1b;font-weight:800;padding:1px 5px;border-radius:3px');
    return `<span class="${cls}" style="${style}">${esc(term)}</span>`;
}

function codeIncludesAnyVariant(code, locator) {
    if (!code || !locator) return false;
    return locatorHighlightVariants(locator).some(v => code.includes(v));
}

function resolveFixLocators(data) {
    let original = (data?.original_locator || '').trim();
    let healed = (data?.healed_locator || '').trim();
    const expl = data?.explanation || '';
    const code = data?.code || '';
    if (!healed) healed = extractHealedFromExplanation(expl);
    if (!original) original = extractOriginalFromExplanation(expl);
    const changeRows = Array.isArray(data?.change_lines) ? data.change_lines : [];
    if (!healed) {
        for (const row of changeRows) {
            const after = extractLocatorFromChangeLine(row?.after);
            if (after) { healed = after; break; }
        }
    }
    if (!original) {
        for (const row of changeRows) {
            const before = extractLocatorFromChangeLine(row?.before);
            if (before) { original = before; break; }
        }
    }
    if (code) original = findOriginalInCode(code, original, expl);
    if (!healed && code && original) healed = inferHealedLocatorFromCode(code, original);
    if (!healed) healed = extractHealedFromExplanation(expl);
    return { original, healed };
}

function highlightLineTerms(line, planEntry) {
    if (!planEntry?.terms?.length) return esc(line);
    let out = esc(line);
    const sorted = [...planEntry.terms].sort((a, b) => b.length - a.length);
    for (const term of sorted) {
        if (!line.includes(term)) continue;
        const span = fixHighlightSpan(term, planEntry.kind === 'old' ? 'old' : 'fix');
        out = line.split(term).map(esc).join(span);
        return out;
    }
    return out;
}

function renderChangeLines(changeLines, original, healed) {
    let rows = Array.isArray(changeLines) ? changeLines.filter(r => r?.before && r?.after) : [];
    if (!rows.length && original && healed && original !== healed) {
        rows = [{ before: original, after: healed }];
    }
    if (!rows.length) return '';
    return `<div class="hc-fix-changes">
        <div class="hc-fix-changes__head">What changes</div>
        ${rows.map(r => `
        <div class="hc-fix-change">
            <div class="hc-fix-change__row hc-fix-change__row--old">
                <span class="hc-fix-change__tag">Before</span>
                <code>${esc(r.before)}</code>
            </div>
            <div class="hc-fix-change__row hc-fix-change__row--new">
                <span class="hc-fix-change__tag">After</span>
                <code>${esc(r.after)}</code>
            </div>
        </div>`).join('')}
    </div>`;
}

/* ── Icons SVG ────────────────────────────────────────────────────────── */
const ICON = {
    fix:     `<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round"><path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z"/></svg>`,
    context: `<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg>`,
    copy:    `<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>`,
    trash:   `<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><polyline points="3 6 5 6 21 6"/><path d="M19 6l-1 14H6L5 6"/><path d="M9 6V4h6v2"/></svg>`,
    refresh: `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round"><polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 1 1-.08-1.96"/></svg>`,
    arrow:   `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><line x1="5" y1="12" x2="19" y2="12"/><polyline points="12 5 19 12 12 19"/></svg>`,
    close:   `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>`,
    check:   `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round"><polyline points="20 6 9 17 4 12"/></svg>`,
    star:    `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M12 2l2.4 7.4H22l-6.2 4.5 2.4 7.4L12 17l-6.2 4.3 2.4-7.4L2 9.4h7.6z"/></svg>`,
};

/* ── Init ─────────────────────────────────────────────────────────────── */
window.loadHealingView = loadHealingView;
export function loadHealingView() {
    if (!document.getElementById('healing-insights-body')) return;
    currentRole = parseJwt(localStorage.getItem('sre-jwt'))?.role || '';
    const sel = document.getElementById('healing-project-filter');
    loadHealingProjects(sel);
    loadHealingInsights();
    loadLocatorInterventions();
    _loadAIChip();
    if (sel && !sel.dataset.bound) {
        sel.dataset.bound = '1';
        sel.addEventListener('change', () => { loadHealingInsights(); loadLocatorInterventions(); });
    }
}

async function _loadAIChip() {
    const chip = document.getElementById('healing-ai-chip');
    if (!chip) return;
    try {
        const { ok, data } = await parseApiJson(await fetchWithAuth('/api/ai/status'));
        if (ok && data?.enabled && data.provider !== 'disabled') {
            chip.innerHTML = `<span class="hc-dot hc-dot--live"></span>${esc(data.provider.toUpperCase())} · ${esc(data.model || '—')}`;
            chip.className = 'hc-chip hc-chip--on';
        } else {
            chip.innerHTML = `<span class="hc-dot"></span>AI off`;
            chip.className = 'hc-chip hc-chip--off';
        }
    } catch { if (chip) { chip.textContent = 'AI —'; chip.className = 'hc-chip hc-chip--off'; } }
}

async function loadHealingProjects(sel) {
    if (!sel) return;
    const { ok, data } = await parseApiJson(await fetchWithAuth('/api/my-projects'));
    sel.innerHTML = '<option value="">All gateways</option>';
    if (ok) (Array.isArray(data) ? data : []).forEach(p =>
        sel.innerHTML += `<option value="${esc(p.name)}">${esc(p.name)}</option>`
    );
}

/* ── Load failures ────────────────────────────────────────────────────── */
export async function loadHealingInsights() {
    const body = document.getElementById('healing-insights-body');
    if (!body) return;
    const project = document.getElementById('healing-project-filter')?.value || '';
    const q = project ? `?project=${encodeURIComponent(project)}&limit=100` : '?limit=100';

    body.innerHTML = `<div class="hc-loading">Loading…</div>`;

    const { ok, data } = await parseApiJson(await fetchWithAuth(`/api/healing/insights${q}`));
    if (!ok) { body.innerHTML = `<div class="hc-empty">Failed to load failures.</div>`; updateStats([]); return; }
    const rows = Array.isArray(data) ? data : [];
    updateStats(rows);
    selected.clear();
    _updateBulkBar();

    if (!rows.length) {
        body.innerHTML = `<div class="hc-empty">No open failures — all tests are passing.</div>`;
        return;
    }
    body.innerHTML = rows.map(r => renderCard(r)).join('');
}
window.loadHealingInsights = loadHealingInsights;

function updateStats(rows) {
    _count('healing-stat-total',         rows.length);
    _count('healing-stat-categorized',   rows.filter(r => r.error_category && r.error_category !== 'unknown').length);
    _count('healing-stat-mcp-ready',     rows.length);
    _count('healing-stat-interventions', rows.filter(r => r.mcp_healed).length);
}

function _count(id, n) {
    const el = document.getElementById(id);
    if (!el) return;
    const from = parseInt(el.textContent) || 0;
    if (from === n) { el.textContent = n; return; }
    const t0 = performance.now();
    const tick = now => { const p = Math.min((now - t0) / 500, 1); el.textContent = Math.round(from + (n - from) * p); if (p < 1) requestAnimationFrame(tick); };
    requestAnimationFrame(tick);
}

/* ── Render one card ──────────────────────────────────────────────────── */
function renderCard(r) {
    const id     = Number(r.incident_id) || 0;
    const f      = fw(r.framework || r.project_name);
    const c      = cat(r.error_category);
    const healed = !!r.mcp_healed;
    const canMgr = canManageHealing(currentRole);
    const canDel = canDeleteIncidents(currentRole);

    // inline tinted badge style using the framework/category color
    const fwStyle  = `color:${f.color};border-color:${f.color}30;background:${f.color}18`;
    const catStyle = `color:${c.color};border-color:${c.color}30;background:${c.color}18`;

    return `
<article class="hc-card${healed ? ' hc-card--healed' : ''}" id="hc-card-${id}" data-id="${id}">
    <div class="hc-card__top">
        <!-- Checkbox -->
        <label class="hc-cb-cell" title="Select failure">
            <input type="checkbox" class="hc-cb" value="${id}" onchange="window._onCardCheck(${id},this.checked)">
        </label>

        <!-- Color stripe -->
        <div class="hc-stripe" style="background:${f.color}"></div>

        <!-- Main body -->
        <div class="hc-body">
            <!-- Meta row -->
            <div class="hc-row-meta">
                <span class="hc-badge" style="${fwStyle}">${esc(f.label)}</span>
                <span class="hc-badge" style="${catStyle}">${esc(c.label)}</span>
                ${healed ? `<span class="hc-badge hc-badge--healed">${ICON.check}&nbsp;Healed</span>` : ''}
                <span class="hc-id">#${id}</span>
            </div>

            <!-- Test name -->
            <h3 class="hc-test-name">${esc(r.test_name || 'Unknown test')}</h3>

            <!-- Error description -->
            <p class="hc-test-desc">${esc(r.summary || r.error_message || 'No description available.')}</p>

            <!-- Action bar -->
            <div class="hc-card__actions">
                ${canMgr ? `
                <button class="hc-btn-fix" title="AI scans live page and proposes the corrected selector" onclick="window.openProposeFix(${id})">
                    ${ICON.fix} Propose Fix
                </button>
                <div class="hc-icon-group">
                    <button class="hc-icon-btn" title="View context" onclick="window.openHealingContext(${id})">${ICON.context}</button>
                    <button class="hc-icon-btn" title="Copy MCP prompt" onclick="window.copyMCPPrompt(${id})">${ICON.copy}</button>
                    ${canDel ? `<button class="hc-icon-btn hc-icon-btn--danger" title="Delete failure" onclick="window.deleteHealingFailure(${id})">${ICON.trash}</button>` : ''}
                </div>` : (canDel ? `
                <div class="hc-icon-group">
                    <button class="hc-icon-btn hc-icon-btn--danger" title="Delete failure" onclick="window.deleteHealingFailure(${id})">${ICON.trash}</button>
                </div>` : '')}
            </div>
        </div>
    </div>

    <!-- Fix proposal panel (injected by openProposeFix) -->
    <div id="fix-proposal-${id}" class="hc-panel" style="display:none"></div>
</article>`;
}

/* ── Checkbox / bulk select ───────────────────────────────────────────── */
window._onCardCheck = (id, checked) => {
    if (checked) selected.add(id); else selected.delete(id);
    _updateBulkBar();
    _syncSelectAll();
};

window._selectAll = checked => {
    document.querySelectorAll('.hc-cb').forEach(cb => {
        cb.checked = checked;
        const id = parseInt(cb.value);
        if (checked) selected.add(id); else selected.delete(id);
    });
    _updateBulkBar();
};

function _updateBulkBar() {
    const bar = document.getElementById('hc-bulk-bar');
    const cnt = document.getElementById('hc-bulk-count');
    if (!bar) return;
    if (selected.size > 0) {
        bar.style.display = 'flex';
        if (cnt) cnt.textContent = `${selected.size} selected`;
    } else {
        bar.style.display = 'none';
    }
}

function _syncSelectAll() {
    const allCbs = document.querySelectorAll('.hc-cb');
    const sa = document.getElementById('hc-select-all');
    if (!sa || !allCbs.length) return;
    const allChecked = [...allCbs].every(cb => cb.checked);
    const someChecked = [...allCbs].some(cb => cb.checked);
    sa.checked = allChecked;
    sa.indeterminate = someChecked && !allChecked;
}

window.bulkDelete = async () => {
    if (!selected.size) return;
    const ids = [...selected].join(',');
    window.showConfirmModal('Delete failures?', `Delete ${selected.size} selected failure(s)?`, 'danger', async () => {
        const { ok } = await parseApiJson(await fetchWithAuth(`/api/incidents?ids=${ids}`, { method: 'DELETE' }));
        if (!ok) { notify('Failed to delete.', 'error'); return; }
        selected.forEach(id => document.getElementById(`hc-card-${id}`)?.remove());
        selected.clear();
        _updateBulkBar();
        notify(`${ids.split(',').length} failure(s) deleted.`, 'success');
        loadHealingInsights();
    });
};

/* ── Delete single ────────────────────────────────────────────────────── */
export function deleteHealingFailure(id) {
    if (!canDeleteIncidents(currentRole)) { notify('Manager or Lead role required.', 'error'); return; }
    window.showConfirmModal('Delete failure?', `Delete INC #${id}?`, 'danger', async () => {
        const { ok } = await parseApiJson(await fetchWithAuth(`/api/incidents?ids=${id}`, { method: 'DELETE' }));
        if (!ok) { notify('Failed to delete.', 'error'); return; }
        document.getElementById(`hc-card-${id}`)?.remove();
        selected.delete(id); _updateBulkBar();
        notify(`INC #${id} deleted.`, 'success');
        loadHealingInsights();
    });
}
window.deleteHealingFailure = deleteHealingFailure;

/* ── Context / copy prompt ────────────────────────────────────────────── */
export async function openHealingContext(id) {
    const { ok, data } = await parseApiJson(await fetchWithAuth(`/api/incidents/${id}/healing/context`));
    if (!ok || !data) { notify('Failed to load context.', 'error'); return; }
    window.__lastHealingContext = data;
    const info = [data.error_category && `Category: ${data.error_category}`, data.selector_hint && `Selector: ${data.selector_hint}`].filter(Boolean).join(' · ');
    notify(info || 'Context loaded.', 'info');
}
window.openHealingContext = openHealingContext;

export async function copyMCPPrompt(id) {
    let ctx = window.__lastHealingContext;
    if (!ctx || ctx.incident_id !== id) {
        const { ok, data } = await parseApiJson(await fetchWithAuth(`/api/incidents/${id}/healing/context`));
        if (!ok || !data) { notify('Failed to build prompt.', 'error'); return; }
        ctx = data;
    }
    const text = ctx.mcp_prompt || `Use get_incident_context with incident_id=${id} and propose a minimal fix.`;
    try { await navigator.clipboard.writeText(text); notify('Prompt copied.', 'success'); } catch { notify(text, 'info'); }
}
window.copyMCPPrompt = copyMCPPrompt;

const HEAL_METHOD_LABEL = {
    html_scan: 'HTML scan (page DOM parsed)',
    ai_dom:    'AI + captured/live DOM',
    ai:        'AI analysis',
    rules:     'Rules (no DOM)',
};

function resolveDisplayCode(data) {
    const locs = resolveFixLocators(data);
    const sourceCode = (data?.source_code || '').trim() || (data?.code || '');
    let displayCode = data?.code || '';
    if (locs.healed && displayCode) {
        if (sourceCode === displayCode || !displayCode.includes(locs.healed)) {
            displayCode = applyFixPreview(sourceCode, locs.original, locs.healed, data?.explanation || '') || displayCode;
        }
    }
    return displayCode;
}

/* ── Propose fix ──────────────────────────────────────────────────────── */
export async function openProposeFix(id) {
    const panel = document.getElementById(`fix-proposal-${id}`);
    if (!panel) return;

    // Toggle
    if (panel.style.display !== 'none') { panel.style.display = 'none'; return; }
    panel.style.display = 'block';
    panel.innerHTML = `<div class="hc-panel__loading"><span class="hc-panel__loading-icon" aria-hidden="true">${ICON.refresh}</span><span>Fetching live page DOM and querying AI…</span></div>`;

    try {
        const { ok, data } = await parseApiJson(await fetchWithAuth(`/api/incidents/${id}/healing/propose`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ file_content: '' }),
        }));

        if (!ok || !data) {
            panel.innerHTML = `<div class="hc-panel__error">Failed to get a fix proposal. Verify AI configuration.</div>`;
            return;
        }

        const pct   = data.confidence ? Math.round(data.confidence * 100) : 0;
        const conf  = pct >= 70 ? 'high' : pct >= 40 ? 'mid' : 'low';
        const domOk = data.dom_used;
        const methodLbl = HEAL_METHOD_LABEL[data.heal_method] || (domOk ? 'Live DOM available' : 'No DOM snapshot');

        // Locator diff row
        let locRow = '';
        if (data.original_locator || data.healed_locator) {
            locRow = `
            <div class="hc-locrow">
                <div class="hc-locrow__col">
                    <span class="hc-loclbl">Broken selector</span>
                    <code class="hc-loc hc-loc--bad">${esc(data.original_locator || '—')}</code>
                </div>
                <span class="hc-arrow">${ICON.arrow}</span>
                <div class="hc-locrow__col">
                    <span class="hc-loclbl">Fixed selector</span>
                    <code class="hc-loc hc-loc--ok">${esc(data.healed_locator || '—')}</code>
                </div>
                <span class="hc-conf hc-conf--${conf}">${pct}% · ${esc(methodLbl)}</span>
            </div>`;
        }

        const domNote = !domOk
            ? `<p class="hc-panel__dom-note">No page HTML was captured for this run. Attach <code>dom_capture_listener.py</code> to Robot tests for precise DOM-based healing via MCP.</p>`
            : '';

        // Explanation
        const expl = data.explanation
            ? `<div class="hc-panel__expl"><strong>Explanation</strong><p>${esc(data.explanation)}</p></div>`
            : '';

        const code = data.code ? renderFixCodeBlock(data.code, data, id) : '';
        window[`__fixCode_${id}`] = resolveDisplayCode(data);

        panel.innerHTML = `
        <div class="hc-panel__inner">
            <div class="hc-panel__head">
                <span class="hc-panel__title">${ICON.star} AI Fix Proposal</span>
                <button class="hc-panel__close" onclick="document.getElementById('fix-proposal-${id}').style.display='none'">${ICON.close}</button>
            </div>
            ${locRow}
            ${domNote}
            ${expl}
            ${code}
        </div>`;
    } catch (e) {
        panel.innerHTML = `<div class="hc-panel__error">${esc(String(e))}</div>`;
    }
}
window.openProposeFix = openProposeFix;
window._copyFix = id => {
    const c = window[`__fixCode_${id}`] || '';
    navigator.clipboard.writeText(c).then(() => notify('Copied.', 'success'), () => notify(c, 'info'));
};

/* ── Locator interventions ────────────────────────────────────────────── */
export async function loadLocatorInterventions() {
    const el = document.getElementById('healing-interventions-body');
    if (!el) return;
    const project = document.getElementById('healing-project-filter')?.value || '';
    const q = project ? `?project=${encodeURIComponent(project)}&limit=50` : '?limit=50';
    el.innerHTML = `<div class="hc-loading">Loading…</div>`;

    const { ok, data } = await parseApiJson(await fetchWithAuth(`/api/healing/locator-interventions${q}`));
    if (!ok) { el.innerHTML = `<div class="hc-empty">Failed to load.</div>`; return; }
    const rows = Array.isArray(data) ? data : [];
    _count('healing-stat-interventions', rows.length);

    if (!rows.length) { el.innerHTML = `<div class="hc-empty">No locator interventions yet. They appear here after the CI Healing Gate runs.</div>`; return; }
    el.innerHTML = rows.map(h => renderItv(h)).join('');
}
window.loadLocatorInterventions = loadLocatorInterventions;

function renderItv(h) {
    const f   = fw(h.framework);
    const pct = h.confidence ? Math.round(h.confidence * 100) : 0;
    const conf = pct >= 70 ? 'high' : pct >= 40 ? 'mid' : 'low';

    return `
<article class="hc-itv">
    <div class="hc-itv__head">
        <span class="hc-badge" style="color:${f.color};border-color:${f.color}20;background:${f.color}14">${esc(f.label)}</span>
        <span class="hc-itv__name">${esc(h.test_name || `INC #${h.incident_id}`)}</span>
        <span class="hc-conf hc-conf--${conf}">${pct}% confidence</span>
        <span class="hc-ago">${ago(h.created_at)}</span>
    </div>
    <div class="hc-locrow">
        <div class="hc-locrow__col">
            <span class="hc-loclbl">Broken</span>
            <code class="hc-loc hc-loc--bad">${esc(h.original_locator || '—')}</code>
        </div>
        <span class="hc-arrow">${ICON.arrow}</span>
        <div class="hc-locrow__col">
            <span class="hc-loclbl">Healed</span>
            <code class="hc-loc hc-loc--ok">${esc(h.healed_locator || 'pending')}</code>
        </div>
    </div>
    ${h.explanation ? `<p class="hc-itv__note">${esc(h.explanation)}</p>` : ''}
</article>`;
}
