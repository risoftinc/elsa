(function () {
  const ENV_COLUMNS_KEY = 'elsa-env-columns';
  const PAGE_SIZE_KEY = 'elsa-env-page-size';
  const PAGE_SIZES = [10, 25, 50, 100];
  const MAX_ENV_COLUMNS = 5;

  const state = {
    environments: [],
    variables: [],
    templates: [],
    selectedEnvIds: [],
    varSearch: '',
    varPage: 1,
    varPageSize: 10,
    envPanelOpen: false,
    editingTplId: null,
    exportMode: null, // 'template' | 'dotenv'
    exportTemplateId: null,
    savingRows: new Set(),
  };

  const $ = (sel) => document.querySelector(sel);
  const $$ = (sel) => document.querySelectorAll(sel);

  async function api(path, opts = {}) {
    const res = await fetch(path, {
      headers: { 'Content-Type': 'application/json' },
      ...opts,
    });
    const text = await res.text();
    let data = null;
    if (text) {
      try {
        data = JSON.parse(text);
      } catch {
        data = { error: text };
      }
    }
    if (!res.ok) {
      throw new Error(data?.error || res.statusText);
    }
    return data;
  }

  function toast(msg, type = 'success') {
    const el = $('#toast');
    el.textContent = msg;
    el.className = 'toast ' + type;
    el.hidden = false;
    setTimeout(() => { el.hidden = true; }, 2800);
  }

  function showPage(name) {
    $$('.page').forEach((p) => p.classList.remove('active'));
    $$('.nav-item').forEach((n) => n.classList.remove('active'));
    $(`#page-${name}`)?.classList.add('active');
    $(`.nav-item[data-page="${name}"]`)?.classList.add('active');
    $('#mainNav')?.classList.remove('open');
    if (name === 'variables') {
      renderEnvPicker();
      renderVariables();
    }
  }

  // Navigation
  $$('.nav-item').forEach((btn) => {
    btn.addEventListener('click', () => showPage(btn.dataset.page));
  });
  $('#navToggle')?.addEventListener('click', () => {
    $('#mainNav')?.classList.toggle('open');
  });

  $$('[data-close]').forEach((el) => {
    el.addEventListener('click', () => {
      el.closest('dialog')?.close();
    });
  });

  // --- Environments ---
  function loadSelectedEnvIds() {
    try {
      const raw = localStorage.getItem(ENV_COLUMNS_KEY);
      if (raw) {
        const ids = JSON.parse(raw);
        if (Array.isArray(ids)) return ids.map(Number).filter(Boolean);
      }
    } catch { /* ignore */ }
    return [];
  }

  function saveSelectedEnvIds() {
    localStorage.setItem(ENV_COLUMNS_KEY, JSON.stringify(state.selectedEnvIds));
    updateEnvColumnsSummary();
  }

  function loadPageSize() {
    try {
      const n = parseInt(localStorage.getItem(PAGE_SIZE_KEY), 10);
      if (PAGE_SIZES.includes(n)) return n;
    } catch { /* ignore */ }
    return 10;
  }

  function savePageSize() {
    localStorage.setItem(PAGE_SIZE_KEY, String(state.varPageSize));
  }

  function updateEnvColumnsSummary() {
    const el = $('#envColumnsSummary');
    if (!el) return;
    const names = state.environments
      .filter((e) => state.selectedEnvIds.includes(e.id))
      .map((e) => e.name);
    if (!names.length) {
      el.textContent = 'None selected';
      return;
    }
    if (names.length <= 2) {
      el.textContent = names.join(', ');
      return;
    }
    el.textContent = `${names.slice(0, 2).join(', ')} +${names.length - 2}`;
  }

  function setEnvPanelOpen(open) {
    state.envPanelOpen = open;
    const panel = $('#envColumnsPanel');
    const btn = $('#envColumnsBtn');
    if (panel) panel.hidden = !open;
    if (btn) btn.setAttribute('aria-expanded', open ? 'true' : 'false');
  }

  function clampSelectedEnvIds(ids) {
    return ids.slice(0, MAX_ENV_COLUMNS);
  }

  function syncSelectedEnvIds() {
    const valid = new Set(state.environments.map((e) => e.id));
    let ids = state.selectedEnvIds.filter((id) => valid.has(id));
    if (!ids.length && state.environments.length) {
      ids = state.environments.slice(0, MAX_ENV_COLUMNS).map((e) => e.id);
    }
    const trimmed = clampSelectedEnvIds(ids);
    if (trimmed.length < ids.length) {
      toast(`Maximum ${MAX_ENV_COLUMNS} environment columns`, 'error');
    }
    state.selectedEnvIds = trimmed;
    saveSelectedEnvIds();
  }

  function getVisibleEnvironments() {
    return state.environments.filter((e) => state.selectedEnvIds.includes(e.id));
  }

  async function loadEnvironments() {
    state.environments = await api('/api/environments');
    if (!state.selectedEnvIds.length) {
      state.selectedEnvIds = loadSelectedEnvIds();
    }
    syncSelectedEnvIds();
    renderEnvironments();
    renderEnvPicker();
    fillExportEnvSelect();
    if ($('#page-variables')?.classList.contains('active')) {
      renderVariables();
    }
  }

  function renderEnvironments() {
    const list = $('#envList');
    if (!state.environments.length) {
      list.innerHTML = '<div class="empty">No environments yet. Add your first group above.</div>';
      return;
    }
    list.innerHTML = state.environments
      .map(
        (e) => `
      <div class="list-item" data-id="${e.id}">
        <div class="meta">
          <div class="title">${escapeHtml(e.name)}</div>
          <div class="sub">Order: ${e.sort_order}</div>
        </div>
        <div class="actions">
          <button type="button" class="btn small" data-export-env="${e.id}">Export .env</button>
          <button type="button" class="btn small" data-edit-env="${e.id}">Edit</button>
          <button type="button" class="btn small danger" data-del-env="${e.id}">Delete</button>
        </div>
      </div>`
      )
      .join('');

    list.querySelectorAll('[data-export-env]').forEach((btn) => {
      btn.addEventListener('click', () => openExport('dotenv', null, +btn.dataset.exportEnv));
    });
    list.querySelectorAll('[data-edit-env]').forEach((btn) => {
      btn.addEventListener('click', () => editEnvironment(+btn.dataset.editEnv));
    });
    list.querySelectorAll('[data-del-env]').forEach((btn) => {
      btn.addEventListener('click', () => deleteEnvironment(+btn.dataset.delEnv));
    });
  }

  function resetEnvForm() {
    const form = $('#envForm');
    if (!form) return;
    delete form.dataset.editId;
    $('#envName').value = '';
    $('#envSort').value = '0';
    const btn = form.querySelector('button[type="submit"]');
    if (btn) btn.textContent = 'Add';
  }

  $('#envForm')?.addEventListener('submit', async (ev) => {
    ev.preventDefault();
    const name = $('#envName').value.trim();
    const sort_order = parseInt($('#envSort').value, 10) || 0;
    const editId = $('#envForm').dataset.editId;

    try {
      if (editId) {
        await api(`/api/environments/${editId}`, {
          method: 'PUT',
          body: JSON.stringify({ name, sort_order }),
        });
      } else {
        await api('/api/environments', {
          method: 'POST',
          body: JSON.stringify({ name, sort_order }),
        });
      }
      resetEnvForm();
      await loadEnvironments();
      await loadVariables();
      toast('Environment saved');
    } catch (err) {
      toast(err.message, 'error');
      if (editId && /not found/i.test(err.message)) {
        resetEnvForm();
      }
    }
  });

  function editEnvironment(id) {
    const e = state.environments.find((x) => x.id === id);
    if (!e) return;
    $('#envName').value = e.name;
    $('#envSort').value = e.sort_order;
    $('#envForm').dataset.editId = String(id);
    $('#envForm button[type="submit"]').textContent = 'Update';
    $('#envName').focus();
  }

  async function deleteEnvironment(id) {
    if (!confirm('Delete this environment and all its values?')) return;
    try {
      await api(`/api/environments/${id}`, { method: 'DELETE' });
      await loadEnvironments();
      const editingId = $('#envForm')?.dataset.editId;
      if (editingId && !state.environments.some((e) => e.id === +editingId)) {
        resetEnvForm();
      }
      await loadVariables();
      toast('Deleted');
    } catch (err) {
      toast(err.message, 'error');
    }
  }

  // --- Variables (table view) ---
  function applyEnvColumnSelection(fromPicker) {
    if (fromPicker) {
      const box = $('#varEnvPicker');
      const checked = [...box.querySelectorAll('input:checked')];
      if (checked.length > MAX_ENV_COLUMNS) {
        const last = checked[checked.length - 1];
        last.checked = false;
        state.selectedEnvIds = checked.slice(0, MAX_ENV_COLUMNS).map((x) => +x.value);
        toast(`Maximum ${MAX_ENV_COLUMNS} environment columns`, 'error');
      } else {
        state.selectedEnvIds = checked.map((x) => +x.value);
      }
    }
    if (!state.selectedEnvIds.length && state.environments.length) {
      state.selectedEnvIds = [state.environments[0].id];
      if (fromPicker) renderEnvPicker();
      toast('At least one environment column is required', 'error');
    }
    state.selectedEnvIds = clampSelectedEnvIds(state.selectedEnvIds);
    saveSelectedEnvIds();
    state.varPage = 1;
    renderEnvPicker();
    renderVariables();
  }

  function renderEnvPicker() {
    const box = $('#varEnvPicker');
    if (!box) return;
    if (!state.environments.length) {
      box.innerHTML = '<div class="sub" style="padding:0.5rem 0.75rem">Add environments first.</div>';
      updateEnvColumnsSummary();
      return;
    }
    box.innerHTML = state.environments
      .map(
        (e) => `
      <label>
        <input type="checkbox" value="${e.id}" ${state.selectedEnvIds.includes(e.id) ? 'checked' : ''}>
        <span>${escapeHtml(e.name)}</span>
      </label>`
      )
      .join('');

    const atMax = state.selectedEnvIds.length >= MAX_ENV_COLUMNS;
    box.querySelectorAll('input').forEach((cb) => {
      const disabled = atMax && !cb.checked;
      cb.disabled = disabled;
      cb.closest('label')?.classList.toggle('is-disabled', disabled);
      cb.addEventListener('change', () => applyEnvColumnSelection(true));
    });
    updateEnvColumnsSummary();
  }

  function renderPagination(totalItems) {
    const el = $('#varPagination');
    if (!el) return;

    if (!totalItems) {
      el.innerHTML = '';
      return;
    }

    const totalPages = Math.max(1, Math.ceil(totalItems / state.varPageSize));
    if (state.varPage > totalPages) state.varPage = totalPages;
    if (state.varPage < 1) state.varPage = 1;

    const start = (state.varPage - 1) * state.varPageSize + 1;
    const end = Math.min(state.varPage * state.varPageSize, totalItems);

    const sizeOptions = PAGE_SIZES.map(
      (n) => `<option value="${n}" ${n === state.varPageSize ? 'selected' : ''}>${n}</option>`
    ).join('');

    el.innerHTML = `
      <div class="var-pagination-info">
        Showing ${start}–${end} of ${totalItems} keys
      </div>
      <div class="var-pagination-controls">
        <label>
          Per page
          <select id="varPageSize">${sizeOptions}</select>
        </label>
        <div class="var-page-btns">
          <button type="button" class="btn small" id="varPagePrev" ${state.varPage <= 1 ? 'disabled' : ''}>Prev</button>
          <span>${state.varPage} / ${totalPages}</span>
          <button type="button" class="btn small" id="varPageNext" ${state.varPage >= totalPages ? 'disabled' : ''}>Next</button>
        </div>
      </div>`;

    $('#varPageSize')?.addEventListener('change', (ev) => {
      state.varPageSize = parseInt(ev.target.value, 10);
      savePageSize();
      state.varPage = 1;
      renderVariables();
    });
    $('#varPagePrev')?.addEventListener('click', () => {
      if (state.varPage > 1) {
        state.varPage--;
        renderVariables();
      }
    });
    $('#varPageNext')?.addEventListener('click', () => {
      const tp = Math.ceil(totalItems / state.varPageSize);
      if (state.varPage < tp) {
        state.varPage++;
        renderVariables();
      }
    });
  }

  async function loadVariables() {
    state.variables = await api('/api/variables');
    renderVariables();
  }

  function getValueForEnv(variable, envId) {
    const found = (variable.values || []).find((v) => v.environment_id === envId);
    return found ? found.value : '';
  }

  function sortVariablesByKey(list) {
    return [...list].sort((a, b) => a.key.localeCompare(b.key, undefined, { sensitivity: 'base' }));
  }

  function filterVariablesBySearch(list) {
    const q = state.varSearch.trim().toLowerCase();
    if (!q) return list;
    return list.filter((v) => v.key.toLowerCase().includes(q));
  }

  function renderVariables() {
    const wrap = $('#varTableWrap');
    if (!wrap) return;

    const cols = getVisibleEnvironments();

    if (!state.environments.length) {
      wrap.innerHTML = '<div class="var-table-empty">Create environment groups first (Environments page).</div>';
      if ($('#varPagination')) $('#varPagination').innerHTML = '';
      return;
    }

    if (!cols.length) {
      wrap.innerHTML = '<div class="var-table-empty">Select at least one environment column above.</div>';
      if ($('#varPagination')) $('#varPagination').innerHTML = '';
      return;
    }

    const filtered = filterVariablesBySearch(sortVariablesByKey(state.variables));
    const totalItems = filtered.length;
    const totalPages = Math.max(1, Math.ceil(totalItems / state.varPageSize));
    if (state.varPage > totalPages) state.varPage = totalPages;
    const pageOffset = (state.varPage - 1) * state.varPageSize;
    const pageItems = filtered.slice(pageOffset, pageOffset + state.varPageSize);

    renderPagination(totalItems);

    const colHeaders = cols
      .map((e) => `<th>${escapeHtml(e.name)}</th>`)
      .join('');

    const noSearchHits = state.variables.length > 0 && !filtered.length;
    const bodyRows = pageItems.length
      ? pageItems
          .map((v) => {
            const cells = cols
              .map(
                (e) => `
            <td>
              <input type="text" data-env-id="${e.id}" value="${escapeAttr(getValueForEnv(v, e.id))}" placeholder="—">
            </td>`
              )
              .join('');
            return `
          <tr data-var-id="${v.id}">
            <td class="col-key">
              <input type="text" class="input-key" value="${escapeAttr(v.key)}" title="${escapeAttr(v.description || '')}">
            </td>
            ${cells}
            <td class="col-actions">
              <button type="button" class="btn icon danger" data-del-var="${v.id}" title="Delete">✕</button>
            </td>
          </tr>`;
          })
          .join('')
      : noSearchHits
        ? `<tr><td colspan="${cols.length + 2}" class="var-table-empty">No keys match your search.</td></tr>`
        : '';

    const addCells = cols
      .map(
        (e) => `
        <td>
          <input type="text" data-add-env="${e.id}" placeholder="Value">
        </td>`
      )
      .join('');

    wrap.innerHTML = `
      <table class="var-table">
        <thead>
          <tr>
            <th class="col-key">Key</th>
            ${colHeaders}
            <th class="col-actions"></th>
          </tr>
        </thead>
        <tbody>${bodyRows}</tbody>
        <tfoot>
          <tr id="varAddRow">
            <td class="col-key">
              <input type="text" id="addVarKey" placeholder="NEW_KEY" required>
            </td>
            ${addCells}
            <td class="col-actions">
              <button type="button" class="btn small primary" id="btnAddVariable">Add</button>
            </td>
          </tr>
        </tfoot>
      </table>`;

    wrap.querySelectorAll('tbody tr[data-var-id]').forEach((row) => {
      const varId = +row.dataset.varId;
      row.querySelectorAll('input').forEach((inp) => {
        inp.addEventListener('input', () => row.classList.add('row-dirty'));
        inp.addEventListener('keydown', (ev) => {
          if (ev.key === 'Enter') {
            ev.preventDefault();
            inp.blur();
          }
        });
        inp.addEventListener('blur', () => {
          if (row.classList.contains('row-dirty')) saveVariableRow(varId, row);
        });
      });
    });

    wrap.querySelectorAll('[data-del-var]').forEach((btn) => {
      btn.addEventListener('click', () => deleteVariable(+btn.dataset.delVar));
    });

    $('#btnAddVariable')?.addEventListener('click', addVariableFromFooter);
    $('#addVarKey')?.addEventListener('keydown', (ev) => {
      if (ev.key === 'Enter') {
        ev.preventDefault();
        addVariableFromFooter();
      }
    });
  }

  function readRowPayload(row) {
    const key = row.querySelector('.input-key, #addVarKey')?.value.trim() || '';
    const cols = getVisibleEnvironments();
    const environment_ids = cols.map((e) => e.id);
    const values = cols.map((e) => {
      const inp = row.querySelector(`input[data-env-id="${e.id}"], input[data-add-env="${e.id}"]`);
      return { environment_id: e.id, value: inp?.value ?? '' };
    });
    return { key, description: '', environment_ids, values };
  }

  async function saveVariableRow(varId, row) {
    if (state.savingRows.has(varId)) return;
    const payload = readRowPayload(row);
    if (!payload.key) {
      toast('Key cannot be empty', 'error');
      return;
    }

    const original = state.variables.find((v) => v.id === varId);
    if (original && original.key === payload.key) {
      const unchanged = getVisibleEnvironments().every((e) => {
        const inp = row.querySelector(`input[data-env-id="${e.id}"]`);
        return (inp?.value ?? '') === getValueForEnv(original, e.id);
      });
      if (unchanged) {
        row.classList.remove('row-dirty');
        return;
      }
    }

    state.savingRows.add(varId);
    try {
      await api(`/api/variables/${varId}`, {
        method: 'PUT',
        body: JSON.stringify(payload),
      });
      await loadVariables();
      toast('Saved');
    } catch (err) {
      toast(err.message, 'error');
    } finally {
      state.savingRows.delete(varId);
    }
  }

  async function addVariableFromFooter() {
    const row = $('#varAddRow');
    if (!row) return;
    const payload = readRowPayload(row);
    if (!payload.key) {
      toast('Enter a key name', 'error');
      $('#addVarKey')?.focus();
      return;
    }
    try {
      await api('/api/variables', {
        method: 'POST',
        body: JSON.stringify(payload),
      });
      row.querySelectorAll('input').forEach((inp) => { inp.value = ''; });
      await loadVariables();
      toast('Variable added');
    } catch (err) {
      toast(err.message, 'error');
    }
  }

  async function deleteVariable(id) {
    if (!confirm('Delete this variable?')) return;
    try {
      await api(`/api/variables/${id}`, { method: 'DELETE' });
      await loadVariables();
      toast('Deleted');
    } catch (err) {
      toast(err.message, 'error');
    }
  }

  $('#varSearch')?.addEventListener('input', (ev) => {
    state.varSearch = ev.target.value;
    state.varPage = 1;
    renderVariables();
  });

  $('#envColumnsBtn')?.addEventListener('click', (ev) => {
    ev.stopPropagation();
    setEnvPanelOpen(!state.envPanelOpen);
  });

  $('#envSelectAll')?.addEventListener('click', () => {
    state.selectedEnvIds = state.environments.slice(0, MAX_ENV_COLUMNS).map((e) => e.id);
    if (state.environments.length > MAX_ENV_COLUMNS) {
      toast(`Maximum ${MAX_ENV_COLUMNS} columns (first ${MAX_ENV_COLUMNS} selected)`, 'error');
    }
    applyEnvColumnSelection(false);
  });

  document.addEventListener('click', (ev) => {
    if (!state.envPanelOpen) return;
    const wrap = $('.env-columns-wrap');
    if (wrap && !wrap.contains(ev.target)) setEnvPanelOpen(false);
  });

  // --- Templates ---
  async function loadTemplates() {
    state.templates = await api('/api/templates');
    renderTemplates();
  }

  function renderTemplates() {
    const list = $('#templateList');
    if (!state.templates.length) {
      list.innerHTML = '<div class="empty">No templates yet.</div>';
      return;
    }
    list.innerHTML = state.templates
      .map(
        (t) => `
      <div class="list-item template-item">
        <div class="template-meta">
          <div class="template-title">${escapeHtml(t.name)}</div>
          <pre class="template-preview">${escapeHtml(t.body)}</pre>
        </div>
        <div class="template-actions">
          <button type="button" class="btn small" data-export-tpl="${t.id}">Export</button>
          <button type="button" class="btn small" data-edit-tpl="${t.id}">Edit</button>
          <button type="button" class="btn small danger" data-del-tpl="${t.id}">Delete</button>
        </div>
      </div>`
      )
      .join('');

    list.querySelectorAll('[data-export-tpl]').forEach((btn) => {
      btn.addEventListener('click', () => openExport('template', +btn.dataset.exportTpl));
    });
    list.querySelectorAll('[data-edit-tpl]').forEach((btn) => {
      btn.addEventListener('click', () => openTplModal(+btn.dataset.editTpl));
    });
    list.querySelectorAll('[data-del-tpl]').forEach((btn) => {
      btn.addEventListener('click', () => deleteTemplate(+btn.dataset.delTpl));
    });
  }

  function openTplModal(id = null) {
    state.editingTplId = id;
    $('#tplModalTitle').textContent = id ? 'Edit template' : 'New template';
    $('#tplName').value = '';
    $('#tplBody').value = '';
    if (id) {
      const t = state.templates.find((x) => x.id === id);
      if (t) {
        $('#tplName').value = t.name;
        $('#tplBody').value = t.body;
      }
    } else {
      $('#tplBody').value = `DB_HOST={{.DB_HOST}}
DB_USER={{.DB_USER}}
DB_PASS={{.DB_PASS}}
DB_PORT={{.DB_PORT}}`;
    }
    $('#tplModal').showModal();
  }

  $('#btnNewTemplate')?.addEventListener('click', () => openTplModal());

  function validateTemplateForm(name, body) {
    if (!name) {
      toast('Template name is required', 'error');
      return false;
    }
    if (!body.trim()) {
      toast('Template body is required', 'error');
      return false;
    }
    const dup = state.templates.find(
      (t) =>
        t.name.toLowerCase() === name.toLowerCase() &&
        t.id !== state.editingTplId
    );
    if (dup) {
      toast(`Template name "${name}" already exists`, 'error');
      return false;
    }
    return true;
  }

  $('#tplForm')?.addEventListener('submit', async (ev) => {
    ev.preventDefault();
    const name = $('#tplName').value.trim();
    const body = $('#tplBody').value;
    if (!validateTemplateForm(name, body)) return;
    try {
      if (state.editingTplId) {
        await api(`/api/templates/${state.editingTplId}`, {
          method: 'PUT',
          body: JSON.stringify({ name, body }),
        });
      } else {
        await api('/api/templates', { method: 'POST', body: JSON.stringify({ name, body }) });
      }
      $('#tplModal').close();
      await loadTemplates();
      toast('Template saved');
    } catch (err) {
      toast(err.message, 'error');
    }
  });

  async function deleteTemplate(id) {
    if (!confirm('Delete this template?')) return;
    try {
      await api(`/api/templates/${id}`, { method: 'DELETE' });
      await loadTemplates();
      toast('Deleted');
    } catch (err) {
      toast(err.message, 'error');
    }
  }

  // --- Export ---
  function fillExportEnvSelect() {
    const sel = $('#exportEnv');
    if (!sel) return;
    sel.innerHTML = state.environments
      .map((e) => `<option value="${e.id}">${escapeHtml(e.name)}</option>`)
      .join('');
  }

  function openExport(mode, templateId, envId) {
    state.exportMode = mode;
    state.exportTemplateId = templateId;
    fillExportEnvSelect();
    if (envId) $('#exportEnv').value = envId;
    $('#exportFilename').value = '';
    $('#exportPreview').textContent = '';
    $('#exportModal').showModal();
    runExport();
  }

  async function runExport() {
    const envId = +$('#exportEnv').value;
    const filename = $('#exportFilename').value.trim();
    if (!envId) {
      toast('Select an environment', 'error');
      return;
    }
    try {
      let result;
      if (state.exportMode === 'template' && state.exportTemplateId) {
        result = await api(`/api/templates/${state.exportTemplateId}/render`, {
          method: 'POST',
          body: JSON.stringify({ environment_id: envId, filename }),
        });
      } else {
        result = await api(`/api/environments/${envId}/export`, {
          method: 'POST',
          body: JSON.stringify({ filename }),
        });
      }
      $('#exportPreview').textContent = result.content;
      if (result.filename && !$('#exportFilename').value.trim()) {
        $('#exportFilename').placeholder = result.filename;
      }
      state.lastExport = result;
    } catch (err) {
      toast(err.message, 'error');
    }
  }

  $('#btnRefreshExport')?.addEventListener('click', runExport);
  $('#exportEnv')?.addEventListener('change', runExport);

  $$('[data-close-export]').forEach((el) => {
    el.addEventListener('click', () => $('#exportModal').close());
  });

  $('#btnCopyExport')?.addEventListener('click', async () => {
    const text = $('#exportPreview').textContent;
    if (!text) {
      toast('Generate preview first', 'error');
      return;
    }
    try {
      await navigator.clipboard.writeText(text);
      toast('Copied to clipboard');
    } catch {
      toast('Copy failed — select text manually', 'error');
    }
  });

  $('#btnDownloadExport')?.addEventListener('click', () => {
    const text = $('#exportPreview').textContent;
    if (!text) {
      toast('Generate preview first', 'error');
      return;
    }
    const name =
      $('#exportFilename').value.trim() ||
      state.lastExport?.filename ||
      '.env';
    const blob = new Blob([text], { type: 'text/plain' });
    const a = document.createElement('a');
    a.href = URL.createObjectURL(blob);
    a.download = name;
    a.click();
    URL.revokeObjectURL(a.href);
    toast('Downloaded');
  });

  function escapeHtml(s) {
    return String(s)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;');
  }

  function escapeAttr(s) {
    return escapeHtml(s).replace(/'/g, '&#39;');
  }

  function truncate(s, n) {
    s = String(s).replace(/\n/g, ' ');
    return s.length > n ? s.slice(0, n) + '…' : s;
  }

  // Init
  async function init() {
    state.selectedEnvIds = loadSelectedEnvIds();
    state.varPageSize = loadPageSize();
    try {
      await loadEnvironments();
      await loadVariables();
      await loadTemplates();
      renderEnvPicker();
    } catch (err) {
      toast('Failed to load: ' + err.message, 'error');
    }
  }

  init();
})();
