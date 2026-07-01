document.addEventListener('DOMContentLoaded', async () => {
  const path = window.EDITOR_PATH;
  const editor = document.getElementById('editor');
  const highlight = document.getElementById('highlight-code');
  const status = document.getElementById('editor-status');
  const saveBtn = document.getElementById('save-btn');
  const validateBtn = document.getElementById('validate-btn');
  const validationPanel = document.getElementById('validation-panel');
  const rawUrlInput = document.getElementById('editor-raw-url');
  const gutter = document.querySelector('.editor-gutter');

  let dirty = false;
  let original = '';

  await App.requireAuth();
  if (gutter) {
    gutter.textContent = App.detectEditorMode(path);
  }
  await loadFile();

  function syncHighlight() {
    highlight.innerHTML = App.highlightConfig(path, editor.value) + '\n';
  }

  function syncScroll() {
    highlight.parentElement.scrollTop = editor.scrollTop;
    highlight.parentElement.scrollLeft = editor.scrollLeft;
  }

  editor.addEventListener('input', () => {
    dirty = editor.value !== original;
    syncHighlight();
    status.textContent = dirty ? 'Unsaved changes' : 'Saved';
  });
  editor.addEventListener('scroll', syncScroll);

  async function loadFile() {
    try {
      const meta = await App.apiJSON('/api/files/' + encodeURI(path) + '?meta=1&content=1');
      editor.value = meta.content;
      original = meta.content;
      rawUrlInput.value = meta.file.raw_url;
      syncHighlight();
      status.textContent = 'Loaded';
    } catch (err) {
      status.textContent = 'Load failed: ' + err.message;
      App.toast(err.message, 'error');
    }
  }

  validateBtn.addEventListener('click', async () => {
    try {
      const result = await App.validateContent(path, editor.value);
      App.showValidationIssues(validationPanel, result);
      if (result.valid) {
        App.toast('Validation passed', 'success');
        status.textContent = 'Validation passed';
      } else {
        App.toast(result.issues.length + ' issue(s) found', 'error');
        status.textContent = 'Validation failed';
      }
    } catch (err) {
      App.toast(err.message, 'error');
    }
  });

  saveBtn.addEventListener('click', async () => {
    saveBtn.disabled = true;
    saveBtn.textContent = 'Saving…';
    try {
      const res = await App.api('/api/files/' + encodeURI(path), {
        method: 'PUT',
        headers: { 'Content-Type': 'text/plain' },
        body: editor.value,
      });
      const data = await res.json().catch(() => ({}));
      if (res.status === 422 && data.validation) {
        App.showValidationIssues(validationPanel, data.validation);
        throw new Error('Validation failed — fix issues and retry');
      }
      if (!res.ok) {
        throw new Error(data.error || 'Save failed');
      }
      validationPanel.classList.add('hidden');
      original = editor.value;
      dirty = false;
      status.textContent = 'Saved';
      App.toast('Saved', 'success');
    } catch (err) {
      App.toast(err.message, 'error');
      status.textContent = 'Save failed';
    } finally {
      saveBtn.disabled = false;
      saveBtn.textContent = 'Save';
    }
  });

  window.addEventListener('beforeunload', (e) => {
    if (dirty) {
      e.preventDefault();
      e.returnValue = '';
    }
  });

  // History
  document.getElementById('history-btn').addEventListener('click', () => {
    document.getElementById('history-modal').classList.remove('hidden');
    loadHistory();
  });
  document.getElementById('history-close').addEventListener('click', () => {
    document.getElementById('history-modal').classList.add('hidden');
  });
  document.querySelector('#history-modal .modal-backdrop')?.addEventListener('click', () => {
    document.getElementById('history-modal').classList.add('hidden');
  });

  async function loadHistory() {
    const list = document.getElementById('history-list');
    list.innerHTML = '<p class="muted">Loading…</p>';
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
            <button class="btn btn-ghost btn-sm" data-commit="${App.escapeHTML(c.hash)}" data-action="preview">Preview</button>
            <button class="btn btn-ghost btn-sm" data-commit="${App.escapeHTML(c.hash)}" data-action="restore">Restore</button>
          </div>
        </div>`).join('');

      list.querySelectorAll('[data-action]').forEach(btn => {
        btn.addEventListener('click', async () => {
          const commit = btn.dataset.commit;
          if (btn.dataset.action === 'preview') {
            const res = await App.api('/api/git/show/' + encodeURI(path) + '?commit=' + commit);
            editor.value = await res.text();
            syncHighlight();
            dirty = true;
            status.textContent = 'Preview ' + commit.slice(0, 7) + ' (unsaved)';
            document.getElementById('history-modal').classList.add('hidden');
          } else {
            if (!confirm('Restore to ' + commit.slice(0, 7) + '?')) return;
            try {
              await App.apiJSON('/api/git/restore/' + encodeURI(path), {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ commit }),
              });
              document.getElementById('history-modal').classList.add('hidden');
              await loadFile();
              App.toast('Restored', 'success');
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

  // Ctrl+S
  document.addEventListener('keydown', (e) => {
    if ((e.ctrlKey || e.metaKey) && e.key === 's') {
      e.preventDefault();
      saveBtn.click();
    }
  });
});
