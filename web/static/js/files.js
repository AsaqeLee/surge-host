document.addEventListener('DOMContentLoaded', async () => {
  const tbody = document.getElementById('files-tbody');
  const loading = document.getElementById('files-loading');
  const empty = document.getElementById('files-empty');
  const tableWrap = document.getElementById('files-table-wrap');

  await App.requireAuth();
  loadFiles();

  async function loadFiles() {
    loading.classList.remove('hidden');
    empty.classList.add('hidden');
    tableWrap.classList.add('hidden');
    try {
      const data = await App.apiJSON('/api/files');
      const files = data.files || [];
      loading.classList.add('hidden');
      if (files.length === 0) {
        empty.classList.remove('hidden');
        return;
      }
      tableWrap.classList.remove('hidden');
      tbody.innerHTML = files.map(f => `
        <tr data-path="${App.escapeHTML(f.path)}">
          <td><code>${App.escapeHTML(f.path)}</code></td>
          <td>${App.formatSize(f.size)}</td>
          <td class="muted">${f.updated_at.replace('T', ' ').slice(0, 16)}</td>
          <td>
            <div class="url-field">
              <input class="url-input" readonly value="${App.escapeHTML(f.raw_url)}" onclick="this.select()">
              <button type="button" class="btn btn-ghost btn-sm copy-btn" data-copy="${App.escapeHTML(f.raw_url)}">复制</button>
            </div>
          </td>
          <td class="actions-cell">
            <a href="/edit/${encodeURI(f.path)}" class="btn btn-ghost btn-sm">编辑</a>
            <button class="btn btn-ghost btn-sm" data-action="history" data-path="${App.escapeHTML(f.path)}">历史</button>
            <button class="btn btn-ghost btn-sm" data-action="rename" data-path="${App.escapeHTML(f.path)}">重命名</button>
            <button class="btn btn-ghost btn-sm btn-danger" data-action="delete" data-path="${App.escapeHTML(f.path)}">删除</button>
          </td>
        </tr>`).join('');

      tbody.querySelectorAll('.copy-btn').forEach(btn => {
        btn.addEventListener('click', () => {
          navigator.clipboard.writeText(btn.dataset.copy).then(() => {
            btn.textContent = '已复制';
            setTimeout(() => { btn.textContent = '复制'; }, 1500);
          });
        });
      });

      tbody.querySelectorAll('[data-action]').forEach(btn => {
        btn.addEventListener('click', () => handleAction(btn.dataset.action, btn.dataset.path));
      });
    } catch (err) {
      loading.classList.add('hidden');
      App.toast(err.message, 'error');
    }
  }

  async function handleAction(action, path) {
    if (action === 'delete') {
      if (!confirm(`确定删除 ${path}？`)) return;
      try {
        await App.apiJSON('/api/files/' + encodeURI(path), { method: 'DELETE' });
        App.toast('已删除', 'success');
        loadFiles();
      } catch (err) {
        App.toast(err.message, 'error');
      }
    } else if (action === 'rename') {
      document.getElementById('rename-old').value = path;
      document.getElementById('rename-new').value = path;
      document.getElementById('rename-modal').classList.remove('hidden');
    } else if (action === 'history') {
      showHistory(path);
    }
  }

  document.getElementById('rename-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    const oldPath = document.getElementById('rename-old').value;
    const newPath = document.getElementById('rename-new').value.trim();
    try {
      await App.apiJSON('/api/files/' + encodeURI(oldPath), {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ new_path: newPath }),
      });
      document.getElementById('rename-modal').classList.add('hidden');
      App.toast('重命名成功', 'success');
      loadFiles();
    } catch (err) {
      App.toast(err.message, 'error');
    }
  });

  document.getElementById('rename-cancel').addEventListener('click', () => {
    document.getElementById('rename-modal').classList.add('hidden');
  });
  document.querySelector('#rename-modal .modal-backdrop')?.addEventListener('click', () => {
    document.getElementById('rename-modal').classList.add('hidden');
  });

  async function showHistory(path) {
    document.getElementById('history-path').textContent = path;
    const list = document.getElementById('history-list');
    list.innerHTML = '<p class="muted">加载中…</p>';
    document.getElementById('history-modal').classList.remove('hidden');

    try {
      const data = await App.apiJSON('/api/git/log/' + encodeURI(path));
      const commits = data.commits || [];
      if (commits.length === 0) {
        list.innerHTML = '<p class="empty">暂无版本记录</p>';
        return;
      }
      list.innerHTML = commits.map(c => `
        <div class="history-item">
          <div class="history-meta">
            <code>${c.short_hash}</code>
            <span>${c.message}</span>
            <span class="muted">${c.timestamp.replace('T', ' ').slice(0, 16)}</span>
          </div>
          <div class="history-actions">
            <button class="btn btn-ghost btn-sm" data-commit="${c.hash}" data-path="${App.escapeHTML(path)}" data-action="view">查看</button>
            <button class="btn btn-ghost btn-sm" data-commit="${c.hash}" data-path="${App.escapeHTML(path)}" data-action="restore">回滚</button>
          </div>
        </div>`).join('');

      list.querySelectorAll('[data-action]').forEach(btn => {
        btn.addEventListener('click', async () => {
          const { commit, path: p, action } = btn.dataset;
          if (action === 'view') {
            const res = await App.api('/api/git/show/' + encodeURI(p) + '?commit=' + commit);
            const text = await res.text();
            alert(text.slice(0, 2000) + (text.length > 2000 ? '\n…' : ''));
          } else if (action === 'restore') {
            if (!confirm(`回滚 ${p} 到 ${commit.slice(0, 7)}？`)) return;
            try {
              await App.apiJSON('/api/git/restore/' + encodeURI(p), {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ commit }),
              });
              App.toast('回滚成功', 'success');
              document.getElementById('history-modal').classList.add('hidden');
              loadFiles();
            } catch (err) {
              App.toast(err.message, 'error');
            }
          }
        });
      });
    } catch (err) {
      list.innerHTML = `<p class="empty">${App.escapeHTML(err.message)}</p>`;
    }
  }

  document.getElementById('history-close').addEventListener('click', () => {
    document.getElementById('history-modal').classList.add('hidden');
  });
  document.querySelector('#history-modal .modal-backdrop')?.addEventListener('click', () => {
    document.getElementById('history-modal').classList.add('hidden');
  });
});