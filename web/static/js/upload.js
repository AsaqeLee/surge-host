document.addEventListener('DOMContentLoaded', async () => {
  const dropzone = document.getElementById('dropzone');
  const fileInput = document.getElementById('file-input');
  const pickBtn = document.getElementById('pick-btn');
  const uploadForm = document.getElementById('upload-form');
  const uploadBtn = document.getElementById('upload-btn');
  const uploadPath = document.getElementById('upload-path');
  const preview = document.getElementById('upload-preview');
  const previewName = document.getElementById('preview-name');
  const previewSize = document.getElementById('preview-size');
  const resultBox = document.getElementById('upload-result');

  let selectedFile = null;

  await App.requireAuth();

  function setFile(file) {
    selectedFile = file;
    previewName.textContent = file.name;
    previewSize.textContent = App.formatSize(file.size);
    preview.classList.remove('hidden');
    uploadBtn.disabled = false;
    if (!uploadPath.value) uploadPath.placeholder = file.name;
  }

  pickBtn.addEventListener('click', () => fileInput.click());
  dropzone.addEventListener('click', (e) => {
    if (e.target === pickBtn) return;
    fileInput.click();
  });

  fileInput.addEventListener('change', () => {
    if (fileInput.files[0]) setFile(fileInput.files[0]);
  });

  ['dragenter', 'dragover'].forEach(evt => {
    dropzone.addEventListener(evt, (e) => {
      e.preventDefault();
      dropzone.classList.add('dragover');
    });
  });
  ['dragleave', 'drop'].forEach(evt => {
    dropzone.addEventListener(evt, (e) => {
      e.preventDefault();
      dropzone.classList.remove('dragover');
    });
  });
  dropzone.addEventListener('drop', (e) => {
    const file = e.dataTransfer.files[0];
    if (file) setFile(file);
  });

  uploadForm.addEventListener('submit', async (e) => {
    e.preventDefault();
    if (!selectedFile) return;

    uploadBtn.disabled = true;
    uploadBtn.textContent = 'Uploading…';

    const fd = new FormData();
    fd.append('file', selectedFile);
    const path = uploadPath.value.trim() || selectedFile.name;
    fd.append('path', path);

    try {
      const res = await App.api('/api/files', { method: 'POST', body: fd });
      const data = await res.json();
      if (res.status === 422 && data.validation) {
        resultBox.classList.remove('hidden');
        App.showValidationIssues(resultBox, data.validation);
        throw new Error('Validation failed');
      }
      if (!res.ok) throw new Error(data.error || 'Upload failed');

      resultBox.classList.remove('hidden');
      resultBox.innerHTML = `
        <p class="result-success">Upload complete</p>
        <div class="url-field">
          <input class="url-input" readonly value="${App.escapeHTML(data.raw_url)}" title="Right-click to copy Raw URL" onclick="this.select()">
          <button type="button" class="btn btn-ghost btn-sm copy-btn" id="result-copy">Copy</button>
        </div>
        <div class="result-actions">
          <a href="/edit/${encodeURI(data.path)}" class="btn btn-secondary btn-sm">Edit</a>
          <a href="/files" class="btn btn-primary btn-sm">Manage</a>
        </div>`;
      App.toast('Upload complete', 'success');
    } catch (err) {
      App.toast(err.message, 'error');
    } finally {
      uploadBtn.disabled = false;
      uploadBtn.textContent = 'Upload';
    }
  });
});