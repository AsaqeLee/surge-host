/* surge-host — shared utilities */
const App = (() => {
  const TOKEN_KEY = 'surge_host_token';
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

  function updateAuthUI() {
    const area = document.getElementById('auth-area');
    if (!area) return;
    if (!authEnabled) {
      area.innerHTML = '<span class="auth-badge">开发模式</span>';
      return;
    }
    if (getToken()) {
      area.innerHTML = `<span class="auth-badge">${escapeHTML(adminUser)}</span>
        <button type="button" class="btn btn-ghost btn-sm" id="logout-btn">退出</button>`;
      document.getElementById('logout-btn')?.addEventListener('click', () => {
        clearToken();
        toast('已退出登录');
      });
    } else {
      area.innerHTML = '<button type="button" class="btn btn-ghost btn-sm" id="login-btn">登录</button>';
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
      throw new Error('请先登录');
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
        toast('登录成功', 'success');
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
    updateAuthUI();
    bindLoginForm();
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
    container.innerHTML = `<div class="validation-header">${validation.valid ? '警告' : '校验未通过'}</div>${html}`;
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
    highlightSurge, escapeHTML, requireAuth, showLogin,
    showValidationIssues, validateContent,
  };
})();