/* surge-host — shared utilities */
const App = (() => {
  const TOKEN_KEY = 'surge_host_token';
  const THEME_KEY = 'surge_host_theme';
  const authEnabled = document.body.dataset.authEnabled === 'true';
  const adminUser = document.body.dataset.adminUser || 'admin';

  function getToken() {
    return localStorage.getItem(TOKEN_KEY);
  }

  function setToken(token) {
    localStorage.setItem(TOKEN_KEY, token);
    updateAuthUI();
  }

  function clearToken() {
    localStorage.removeItem(TOKEN_KEY);
    updateAuthUI();
  }

  function theme() {
    return document.documentElement.dataset.theme || 'dark';
  }

  function setTheme(t) {
    document.documentElement.dataset.theme = t;
    localStorage.setItem(THEME_KEY, t);
    updateThemeUI();
  }

  function toggleTheme() {
    setTheme(theme() === 'dark' ? 'light' : 'dark');
  }

  function updateThemeUI() {
    const label = document.getElementById('theme-label');
    if (label) label.textContent = theme() === 'dark' ? 'Light' : 'Dark';
  }

  function bindMobileNav() {
    const toggle = document.getElementById('nav-mobile-toggle');
    const nav = document.getElementById('mobile-nav');
    if (!toggle || !nav) return;
    toggle.addEventListener('click', () => {
      nav.classList.toggle('is-open');
    });
    nav.querySelectorAll('.nav-link').forEach((link) => {
      link.addEventListener('click', () => nav.classList.remove('is-open'));
    });
  }

  function updateAuthUI() {
    const area = document.getElementById('auth-area');
    if (!area) return;
    if (!authEnabled) {
      area.innerHTML = '<span class="auth-badge">DEV</span>';
      return;
    }
    if (getToken()) {
      area.innerHTML = `<span class="auth-badge">${escapeHTML(adminUser)}</span>
        <button type="button" class="btn btn-ghost btn-sm" id="logout-btn">Logout</button>`;
      document.getElementById('logout-btn')?.addEventListener('click', () => {
        clearToken();
        toast('Signed out');
      });
    } else {
      area.innerHTML = '<button type="button" class="btn btn-ghost btn-sm" id="login-btn">Sign In</button>';
      document.getElementById('login-btn')?.addEventListener('click', showLogin);
    }
  }

  function showLogin() {
    document.getElementById('login-modal')?.classList.remove('hidden');
  }

  function hideLogin() {
    document.getElementById('login-modal')?.classList.add('hidden');
  }

  async function login(username, password) {
    const res = await fetch('/api/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, password }),
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || '登录失败');
    setToken(data.token);
    hideLogin();
    return data;
  }

  async function api(path, options = {}) {
    const headers = { ...(options.headers || {}) };
    const token = getToken();
    if (token) headers['Authorization'] = 'Bearer ' + token;

    const res = await fetch(path, { ...options, headers });
    if (res.status === 401 && authEnabled) {
      clearToken();
      showLogin();
      throw new Error('Sign in required');
    }
    return res;
  }

  async function apiJSON(path, options = {}) {
    const res = await api(path, options);
    const data = await res.json().catch(() => ({}));
    if (!res.ok) throw new Error(data.error || res.statusText);
    return data;
  }

  function toast(msg, type = 'info') {
    const el = document.getElementById('toast');
    if (!el) return;
    el.textContent = msg;
    el.className = 'toast toast-' + type;
    el.classList.remove('hidden');
    clearTimeout(el._timer);
    el._timer = setTimeout(() => el.classList.add('hidden'), 3500);
  }

  function formatSize(bytes) {
    if (bytes < 1024) return bytes + ' B';
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
    return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
  }

  function escapeHTML(s) {
    return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
  }

  function copyText(text) {
    return navigator.clipboard.writeText(text);
  }

  function urlInputFromTarget(target) {
    const input = target.closest('.url-input');
    if (input?.value) return input;
    const field = target.closest('.url-field');
    return field?.querySelector('.url-input') || null;
  }

  function bindUrlCopy() {
    document.addEventListener('contextmenu', (e) => {
      const input = urlInputFromTarget(e.target);
      if (!input?.value) return;
      e.preventDefault();
      copyText(input.value)
        .then(() => toast('Raw URL copied', 'success'))
        .catch(() => toast('Copy failed', 'error'));
    });

    document.addEventListener('click', (e) => {
      const btn = e.target.closest('.copy-btn');
      if (!btn) return;
      const text = btn.dataset.copy || urlInputFromTarget(btn)?.value;
      if (!text) return;
      e.preventDefault();
      copyText(text)
        .then(() => {
          const orig = btn.textContent;
          btn.textContent = 'Copied';
          setTimeout(() => { btn.textContent = orig; }, 1500);
        })
        .catch(() => toast('Copy failed', 'error'));
    });
  }

  function detectEditorMode(path = '') {
    const lower = String(path).toLowerCase();
    if (lower.endsWith('.json')) return 'json';
    if (lower.endsWith('.yaml') || lower.endsWith('.yml')) return 'yaml';
    if (lower.endsWith('.list') || lower.endsWith('.conf') || lower.endsWith('.module')) return 'surge';
    return 'text';
  }

  function highlightSurge(code) {
    let s = escapeHTML(code);
    // comments
    s = s.replace(/(#.*)$/gm, '<span class="hl-comment">$1</span>');
    // section headers
    s = s.replace(/(\[[\w-]+\])/g, '<span class="hl-section">$1</span>');
    // rule keywords
    s = s.replace(/\b(DOMAIN(?:-SUFFIX|-KEYWORD|-SET)?|IP-CIDR(?:6)?|GEOIP|RULE-SET|URL-REGEX|PROCESS-NAME|AND|OR|NOT|FINAL|DIRECT|REJECT|PROXY|SCRIPT|HTTP-RESPONSE|SUBNET)\b/g,
      '<span class="hl-keyword">$1</span>');
    // URLs
    s = s.replace(/(https?:\/\/[^\s,]+)/g, '<span class="hl-url">$1</span>');
    return s;
  }

  function highlightYAML(code) {
    let s = escapeHTML(code);
    s = s.replace(/(#.*)$/gm, '<span class="hl-comment">$1</span>');
    s = s.replace(/^(\s*-\s*)?([A-Za-z0-9_.-]+)(\s*:)/gm, (_, prefix = '', key, suffix) => {
      return `${prefix}<span class="hl-keyword">${key}</span>${suffix}`;
    });
    s = s.replace(/(https?:\/\/[^\s'"]+)/g, '<span class="hl-url">$1</span>');
    return s;
  }

  function highlightJSON(code) {
    let s = escapeHTML(code);
    s = s.replace(/("(?:\\.|[^"\\])*")(\s*:)/g, '<span class="hl-keyword">$1</span>$2');
    s = s.replace(/:\s*("(?:\\.|[^"\\])*")/g, ': <span class="hl-url">$1</span>');
    s = s.replace(/\b(true|false|null)\b/g, '<span class="hl-comment">$1</span>');
    return s;
  }

  function highlightConfig(path, code) {
    switch (detectEditorMode(path)) {
      case 'json':
        return highlightJSON(code);
      case 'yaml':
        return highlightYAML(code);
      case 'surge':
        return highlightSurge(code);
      default:
        return escapeHTML(code);
    }
  }

  function bindLoginForm() {
    const form = document.getElementById('login-form');
    if (!form) return;
    form.addEventListener('submit', async (e) => {
      e.preventDefault();
      try {
        await login(
          document.getElementById('login-user').value,
          document.getElementById('login-pass').value
        );
        toast('Signed in', 'success');
        window.dispatchEvent(new Event('surge-auth-ready'));
      } catch (err) {
        toast(err.message, 'error');
      }
    });
    document.querySelector('#login-modal .modal-backdrop')?.addEventListener('click', hideLogin);
  }

  function requireAuth() {
    return new Promise((resolve) => {
      if (!authEnabled || getToken()) {
        resolve();
        return;
      }
      showLogin();
      const handler = () => {
        window.removeEventListener('surge-auth-ready', handler);
        resolve();
      };
      window.addEventListener('surge-auth-ready', handler);
    });
  }

  document.addEventListener('DOMContentLoaded', () => {
    updateThemeUI();
    updateAuthUI();
    bindLoginForm();
    bindUrlCopy();
    bindMobileNav();
    document.getElementById('theme-toggle')?.addEventListener('click', toggleTheme);
  });

  function showValidationIssues(container, validation) {
    if (!container || !validation) return;
    const issues = validation.issues || [];
    if (issues.length === 0) {
      container.classList.add('hidden');
      container.innerHTML = '';
      return;
    }
    container.classList.remove('hidden');
    const html = issues.map(i =>
      `<div class="validation-item validation-${i.level}">
        <span class="validation-line">L${i.line}</span>
        <span>${escapeHTML(i.message)}</span>
      </div>`
    ).join('');
    container.innerHTML = `<div class="validation-header">${validation.valid ? 'Warnings' : 'Validation failed'}</div>${html}`;
  }

  async function validateContent(path, content) {
    return apiJSON('/api/validate', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ path, content }),
    });
  }

  return {
    authEnabled, getToken, api, apiJSON, toast, formatSize,
    detectEditorMode, highlightConfig, escapeHTML, requireAuth, showLogin,
    showValidationIssues, validateContent,
  };
})();
