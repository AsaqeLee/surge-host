document.addEventListener('DOMContentLoaded', async () => {
  const path = window.EDITOR_PATH;
  const editor = document.getElementById('editor');
  const highlight = document.getElementById('highlight-code');
  const status = document.getElementById('editor-status');
  const saveBtn = document.getElementById('save-btn');
  const validateBtn = document.getElementById('validate-btn');
  const validationPanel = document.getElementById('validation-panel');
  const rawUrlInput = document.getElementById('editor-raw-url');

  let dirty = false;
  let original = '';

  await App.requireAuth();
  await loadFile();

  function syncHighlight() {
    highlight.innerHTML = App.highlightSurge(editor.value) + '\n';
  }

  function syncScroll() {
    highlight.parentElement.scrollTop = editor.scrollTop;
    highlight.parentElement.scrollLeft = editor.scrollLeft;
  }

  editor.addEventListener('input', () => {
    dirty = editor.value !== original;
    syncHighlight();
    status.textContent = dirty ? '未保存的更改' : '已保存';
  });
  editor.addEventListener('scroll', syncScroll);

  async function loadFile() {
    try {
      const meta = await App.apiJSON('/api/files/' + encodeURI(path) + '?meta=1&content=1');
      editor.value = meta.content;
      original = meta.content;
      rawUrlInput.value = meta.file.raw_url;
      syncHighlight();
      status.textContent = '已加载';
    } catch (err) {
      status.textContent = '加载失败：' + err.message;
      App.toast(err.message, 'error');
    }
  }

  validateBtn.addEventListener('click', async () => {
    try {
      const result = await App.validateContent(path, editor.value);
      App.showValidationIssues(validationPanel, result);
      if (result.valid) {
        App.toast('校验通过', 'success');
        status.textContent = '校验通过';
      } else {
        App.toast('发现 ' + result.issues.length + ' 个问题', 'error');
        status.textContent = '校验未通过';
      }
    } catch (err) {
      App.toast(err.message, 'error');
    }
  });

  saveBtn.addEventListener('click', async () => {
    saveBtn.disabled = true;
    saveBtn.textContent = '保存中…';
    try {
      const res = await App.api('/api/files/' + encodeURI(path), {
        method: 'PUT',
        headers: { 'Content-Type': 'text/plain' },
        body: editor.value,
      });
      const data = await res.json().catch(() => ({}));
      if (res.status === 422 && data.validation) {
        App.showValidationIssues(validationPanel, data.validation);
        throw new Error('语法校验未通过，请修正后重试');
      }
      if (!res.ok) {
        throw new Error(data.error || '保存失败');
      }
      validationPanel.classList.add('hidden');
      original = editor.value;
      dirty = false;
      status.textContent = '已保存';
      App.toast('保存成功', 'success');
    } catch (err) {
      App.toast(err.message, 'error');
      status.textContent = '保存失败';
    } finally {
      saveBtn.disabled = false;
      saveBtn.textContent = '保存';
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
    list.innerHTML = '<p class="muted">加载中…</p>';
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
            <button class="btn btn-ghost btn-sm" data-commit="${c.hash}" data-action="preview">预览</button>
            <button class="btn btn-ghost btn-sm" data-commit="${c.hash}" data-action="restore">回滚</button>
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
            status.textContent = '预览版本 ' + commit.slice(0, 7) + '（未保存）';
            document.getElementById('history-modal').classList.add('hidden');
          } else {
            if (!confirm('回滚到 ' + commit.slice(0, 7) + '？')) return;
            try {
              await App.apiJSON('/api/git/restore/' + encodeURI(path), {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ commit }),
              });
              document.getElementById('history-modal').classList.add('hidden');
              await loadFile();
              App.toast('回滚成功', 'success');
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