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
              <input class="url-input" readonly value="${App.escapeHTML(f.raw_url)}" title="Right-click to copy Raw URL" onclick="this.select()">
              <button type="button" class="btn btn-ghost btn-sm copy-btn" data-copy="${App.attrEscape(f.raw_url)}">Copy</button>
            </div>
          </td>
          <td class="actions-cell">
            <a href="/edit/${encodeURI(f.path)}" class="btn btn-ghost btn-sm">Edit</a>
            <button class="btn btn-ghost btn-sm" data-action="history" data-path="${App.escapeHTML(f.path)}">History</button>
            <button class="btn btn-ghost btn-sm" data-action="rename" data-path="${App.escapeHTML(f.path)}">Rename</button>
            <button class="btn btn-ghost btn-sm btn-danger" data-action="delete" data-path="${App.escapeHTML(f.path)}">Delete</button>
          </td>
        </tr>`).join('');

      tbody.querySelectorAll('[data-action]').forEach(btn => {
        btn.addEventListener('click', () => handleAction(btn.dataset.action, btn.dataset.path));
      });
      App.bindCopyButtons(tbody);
    } catch (err) {
      loading.classList.add('hidden');
      App.toast(err.message, 'error');
    }
  }

  async function handleAction(action, path) {
    if (action === 'delete') {
      if (!confirm(`Delete ${path}?`)) return;
      try {
        await App.apiJSON('/api/files/' + encodeURI(path), { method: 'DELETE' });
        App.toast('Deleted', 'success');
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
      App.toast('Renamed', 'success');
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
    list.innerHTML = '<p class="muted">Loading…</p>';
    document.getElementById('history-modal').classList.remove('hidden');

    try {
      const data = await App.apiJSON('/api/git/log/' + encodeURI(path));
      const commits = data.commits || [];
      if (commits.length === 0) {
        list.innerHTML = '<p class="empty">No version history</p>';
        return;
      }
      list.innerHTML = commits.map(c => `
        <div class="history-item">
          <div class="history-meta">
            <code>${App.escapeHTML(c.short_hash)}</code>
            <span>${App.escapeHTML(c.message)}</span>
            <span class="muted">${App.escapeHTML(c.timestamp.replace('T', ' ').slice(0, 16))}</span>
          </div>
          <div class="history-actions">
            <button class="btn btn-ghost btn-sm" data-commit="${App.escapeHTML(c.hash)}" data-path="${App.escapeHTML(path)}" data-action="view">View</button>
            <button class="btn btn-ghost btn-sm" data-commit="${App.escapeHTML(c.hash)}" data-path="${App.escapeHTML(path)}" data-action="restore">Restore</button>
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
            if (!confirm(`Restore ${p} to ${commit.slice(0, 7)}?`)) return;
            try {
              await App.apiJSON('/api/git/restore/' + encodeURI(p), {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ commit }),
              });
              App.toast('Restored', 'success');
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