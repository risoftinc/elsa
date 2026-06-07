(function () {
  const state = {
    projects: [],
    activeProjectId: null,
    schema: null,
    editingTableId: null,
    editingProjectId: null,
    enumEditRow: null,
    lastExport: null,
    drag: null,
    linkDrag: null,
    pendingRelation: null,
    editingRelationId: null,
    selectedTableIds: new Set(),
    marquee: null,
    lastSyncedSql: '',
    syncFrom: null,
    exportTimer: null,
    importTimer: null,
    sqlPanelHidden: false,
    exportPreview: null,
  };

  const SQL_EXPORT_MS = 400;
  const SQL_IMPORT_MS = 700;
  const SQL_PANEL_KEY = 'elsa-dbdesign-sql-panel';

  const $ = (s) => document.querySelector(s);
  const $$ = (s) => document.querySelectorAll(s);

  const DESIGNER_BTNS = [
    'btnAddTable', 'btnAutoLayout', 'btnReloadSchema', 'btnToggleSql',
    'btnCopySQL', 'btnDownloadSQL', 'btnExportDiagram', 'exportImageStyle', 'exportImageFormat', 'exportDialect',
  ];

  const DIAGRAM_THEMES = {
    dark: {
      bg: '#0d1117',
      grid: true,
      gridDot: '#30363d',
      tableFill: '#1c2128',
      tableStroke: '#30363d',
      headerFill: '#161b22',
      headerLine: '#30363d',
      text: '#e6edf3',
      textMuted: '#8b949e',
      relStroke: '#a371f7',
      relWidth: 2,
      arrowFill: '#a371f7',
      pkText: '#f0883e',
      pkFill: '#f0883e33',
      fkText: '#a371f7',
      fkFill: '#a371f733',
      uqText: '#58a6ff',
      uqFill: '#58a6ff33',
      aiText: '#3fb950',
      aiFill: '#3fb95033',
      idxText: '#3fb950',
      idxFill: '#3fb95033',
      idxUqText: '#a371f7',
      idxUqFill: '#a371f733',
      radius: 10,
      titleBlock: false,
      pngScale: 2,
    },
    minimal: {
      bg: '#ffffff',
      grid: false,
      tableFill: '#ffffff',
      tableStroke: '#222222',
      headerFill: '#f4f4f4',
      headerLine: '#222222',
      text: '#111111',
      textMuted: '#444444',
      relStroke: '#333333',
      relWidth: 1.5,
      arrowFill: '#333333',
      pkText: '#111111',
      pkFill: '#eeeeee',
      fkText: '#111111',
      fkFill: '#eeeeee',
      uqText: '#111111',
      uqFill: '#eeeeee',
      aiText: '#111111',
      aiFill: '#eeeeee',
      idxText: '#111111',
      idxFill: '#eeeeee',
      idxUqText: '#111111',
      idxUqFill: '#dddddd',
      radius: 3,
      titleBlock: true,
      pngScale: 3,
    },
  };

  async function api(path, opts = {}) {
    const res = await fetch(path, { headers: { 'Content-Type': 'application/json' }, ...opts });
    const text = await res.text();
    let data = text ? JSON.parse(text) : null;
    if (!res.ok) throw new Error(data?.error || res.statusText);
    return data;
  }

  function toast(msg, type = 'success') {
    const el = $('#toast');
    el.textContent = msg;
    el.className = 'toast ' + (type === 'error' ? 'error' : '');
    el.hidden = false;
    setTimeout(() => { el.hidden = true; }, 2800);
  }

  function esc(s) {
    return String(s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
  }

  function formatDataTypeDisplay(type) {
    const t = String(type || '').replace(/\s+/g, ' ').trim();
    if (!t) return 'TEXT';

    const enumMatch = t.match(/^ENUM\s*\((.*)\)\s*$/i);
    if (enumMatch) {
      const values = enumMatch[1]
        .split(',')
        .map((v) => v.trim().replace(/^['"]|['"]$/g, ''))
        .filter(Boolean);
      if (values.length === 0) return 'ENUM';
      if (values.length === 1) return `ENUM('${values[0]}')`;
      const preview = values.slice(0, 2).join(', ');
      if (values.length <= 2 && preview.length <= 20) return `ENUM(${preview})`;
      return `ENUM(${values.length})`;
    }

    const setMatch = t.match(/^SET\s*\((.*)\)\s*$/i);
    if (setMatch) {
      const values = setMatch[1]
        .split(',')
        .map((v) => v.trim().replace(/^['"]|['"]$/g, ''))
        .filter(Boolean);
      if (values.length <= 2 && values.join(', ').length <= 18) {
        return `SET(${values.join(', ')})`;
      }
      return `SET(${values.length || '…'})`;
    }

    if (t.length > 24) return `${t.slice(0, 21)}…`;
    return t;
  }

  function escXml(s) {
    return esc(s);
  }

  function slugName(name) {
    return String(name || 'diagram').trim().toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '') || 'diagram';
  }

  function downloadBlob(blob, filename) {
    const a = document.createElement('a');
    a.href = URL.createObjectURL(blob);
    a.download = filename;
    a.click();
    setTimeout(() => URL.revokeObjectURL(a.href), 2000);
  }

  function setDesignerEnabled(on) {
    DESIGNER_BTNS.forEach((id) => {
      const el = $('#' + id);
      if (el) el.disabled = !on;
    });
  }

  function setSqlSyncStatus(msg, type) {
    const toolbar = $('#sqlSyncStatus');
    const panel = $('#sqlPanelStatus');
    [toolbar, panel].forEach((el) => {
      if (!el) return;
      el.textContent = msg || '';
      el.hidden = !msg;
      el.classList.remove('ok', 'error');
      if (type) el.classList.add(type);
    });
  }

  function applySqlPanelVisibility() {
    const split = $('#designerSplit');
    const btn = $('#btnToggleSql');
    if (!split || !btn) return;
    split.classList.toggle('sql-hidden', state.sqlPanelHidden);
    btn.textContent = state.sqlPanelHidden ? 'Show SQL' : 'Hide SQL';
    try {
      localStorage.setItem(SQL_PANEL_KEY, state.sqlPanelHidden ? '1' : '0');
    } catch (_) { /* ignore */ }
    if (!state.sqlPanelHidden) drawRelations();
  }

  function initSqlPanel() {
    try {
      state.sqlPanelHidden = localStorage.getItem(SQL_PANEL_KEY) === '1';
    } catch (_) { /* ignore */ }
    applySqlPanelVisibility();
  }

  $('#btnToggleSql')?.addEventListener('click', () => {
    state.sqlPanelHidden = !state.sqlPanelHidden;
    applySqlPanelVisibility();
  });

  function showPage(name) {
    $$('.page').forEach((p) => p.classList.remove('active'));
    $$('.nav-item').forEach((n) => n.classList.remove('active'));
    $(`#page-${name}`)?.classList.add('active');
    $(`.nav-item[data-page="${name}"]`)?.classList.add('active');
    $('#mainNav')?.classList.remove('open');
    if (name === 'projects') rotateProjectsTagline();
    if (name === 'designer') {
      rotateDesignerHint();
      renderCanvas();
    }
  }

  $$('.nav-item').forEach((b) => b.addEventListener('click', () => showPage(b.dataset.page)));
  $('#navToggle')?.addEventListener('click', () => $('#mainNav')?.classList.toggle('open'));
  $$('[data-close]').forEach((el) => el.addEventListener('click', () => el.closest('dialog')?.close()));

  // --- Projects ---
  const PROJECT_TAGLINES = [
    'One project, one schema — keep it clean. Designer is where you build tables & FKs; MySQL, PostgreSQL, or SQLite lives in the toolbar.',
    'Your schemas chill here until you\'re ready. Open Designer to cook the diagram; pick your SQL dialect from the toolbar.',
    'New project = blank canvas for your DB. Tables, relations, export — all in Designer. Dialect switch is up top.',
    'Lowkey: create a project, hop into Designer, model your tables. MySQL, PostgreSQL, or SQLite? Toolbar has you.',
    'Schema projects, no fluff. Designer has the canvas + live SQL; toolbar picks MySQL, PostgreSQL, or SQLite.',
  ];

  function rotateProjectsTagline() {
    const el = $('#projectsTagline');
    if (!el || !PROJECT_TAGLINES.length) return;
    el.textContent = PROJECT_TAGLINES[Math.floor(Math.random() * PROJECT_TAGLINES.length)];
  }

  const DESIGNER_TIPS = [
    'Select tables: drag on empty canvas or Shift/Ctrl+click · move the group by dragging any selected table.',
    'Double-click a table to edit columns, types, PK/FK flags, and ENUM values.',
    'Drag from one column to another table\'s column to create a foreign key. Click a relation line to set ON DELETE / ON UPDATE.',
    'SQL panel ↔ canvas stay in sync — edit CREATE TABLE here and the diagram updates (and vice versa).',
    'Auto layout tidies tables left-to-right based on FK depth — great after importing SQL.',
    'Export diagram as PNG or SVG; Minimal (print) gives a white background for docs or skripsi.',
    'Pick MySQL, PostgreSQL, or SQLite in the toolbar before Copy or Download SQL.',
    'ENUM or SET? Double-click the table, then Manage values… for a proper list editor.',
    'Hide SQL when you want more canvas — toggle Show/Hide SQL anytime from the toolbar.',
    'Paste existing DDL into the SQL panel to pull tables onto the canvas in one go.',
    'Shift/Ctrl+click adds tables to your selection without clearing the rest.',
    'Reload pulls the latest schema from the server if something feels out of date.',
    'In the column editor, drag the ⋮⋮ handle to reorder columns — canvas and exported SQL follow the same order.',
    'Default values use type-aware controls — pick from dropdown for ENUM/DATETIME, integers block letters.',
    'Double-click a table to manage indexes — composite keys supported, order of columns matters.',
  ];

  function rotateDesignerHint() {
    const el = $('#designerHint');
    if (!el || !DESIGNER_TIPS.length) return;
    el.textContent = DESIGNER_TIPS[Math.floor(Math.random() * DESIGNER_TIPS.length)];
  }

  async function loadProjects() {
    state.projects = await api('/api/projects');
    renderProjects();
  }

  function projectSubtitle(p) {
    const desc = (p.description || '').trim();
    if (desc) return esc(desc);
    return '<span class="sub-empty">No description</span>';
  }

  function renderProjects() {
    const list = $('#projectList');
    if (!state.projects.length) {
      list.innerHTML = '<div class="empty">No projects yet. Create one above.</div>';
      return;
    }
    list.innerHTML = state.projects.map((p) => `
      <div class="list-item project-card ${p.id === state.activeProjectId ? 'active-project' : ''}">
        <div class="meta">
          <div class="title">${esc(p.name)}</div>
          <div class="sub">${projectSubtitle(p)}</div>
        </div>
        <div class="actions">
          <button type="button" class="btn small primary" data-open="${p.id}">Open designer</button>
          <button type="button" class="btn small" data-edit-proj="${p.id}">Edit</button>
          <button type="button" class="btn small danger" data-del-proj="${p.id}">Delete</button>
        </div>
      </div>`).join('');

    list.querySelectorAll('[data-open]').forEach((b) => {
      b.addEventListener('click', async () => {
        state.activeProjectId = +b.dataset.open;
        await loadSchema();
        showPage('designer');
      });
    });
    list.querySelectorAll('[data-edit-proj]').forEach((b) => {
      b.addEventListener('click', () => openProjectModal(+b.dataset.editProj));
    });
    list.querySelectorAll('[data-del-proj]').forEach((b) => {
      b.addEventListener('click', async () => {
        if (!confirm('Delete project and all tables?')) return;
        await api(`/api/projects/${b.dataset.delProj}`, { method: 'DELETE' });
        if (state.activeProjectId === +b.dataset.delProj) {
          state.activeProjectId = null;
          state.schema = null;
          state.lastSyncedSql = '';
          clearTimeout(state.exportTimer);
          clearTimeout(state.importTimer);
          setDesignerEnabled(false);
          setSqlSyncStatus('');
          $('#designerProjectName').textContent = 'No project selected';
          $('#sqlEditor').value = '';
        }
        await loadProjects();
        toast('Project deleted');
      });
    });
  }

  $('#projectForm')?.addEventListener('submit', async (e) => {
    e.preventDefault();
    try {
      await api('/api/projects', {
        method: 'POST',
        body: JSON.stringify({
          name: $('#projectName').value.trim(),
          description: $('#projectDesc')?.value.trim() || '',
        }),
      });
      $('#projectName').value = '';
      if ($('#projectDesc')) $('#projectDesc').value = '';
      await loadProjects();
      toast('Project created');
    } catch (err) {
      toast(err.message, 'error');
    }
  });

  function openProjectModal(projectId) {
    const p = state.projects.find((x) => x.id === projectId);
    if (!p) return;
    state.editingProjectId = projectId;
    $('#projectModalTitle').textContent = 'Edit project';
    $('#editProjectName').value = p.name;
    $('#editProjectDesc').value = p.description || '';
    $('#projectModal').showModal();
  }

  $('#btnSaveProject')?.addEventListener('click', async () => {
    const name = $('#editProjectName')?.value.trim();
    if (!name) return toast('Project name is required', 'error');
    const description = $('#editProjectDesc')?.value.trim() || '';
    try {
      const updated = await api(`/api/projects/${state.editingProjectId}`, {
        method: 'PUT',
        body: JSON.stringify({ name, description }),
      });
      const idx = state.projects.findIndex((x) => x.id === state.editingProjectId);
      if (idx >= 0) state.projects[idx] = updated;
      if (state.activeProjectId === state.editingProjectId) {
        $('#designerProjectName').textContent = updated.name;
        if (state.schema?.project) {
          state.schema.project.name = updated.name;
          state.schema.project.description = updated.description;
        }
      }
      $('#projectModal').close();
      renderProjects();
      toast('Project updated');
    } catch (err) {
      toast(err.message, 'error');
    }
  });

  // --- Schema / Canvas ---
  async function loadSchema(options = {}) {
    const { pushSql = true } = options;
    if (!state.activeProjectId) return;
    state.schema = await api(`/api/projects/${state.activeProjectId}/schema`);
    $('#designerProjectName').textContent = state.schema.project.name;
    const dialect = state.schema.project.dialect || 'mysql';
    if ($('#exportDialect')) $('#exportDialect').value = dialect;
    setDesignerEnabled(true);
    $('#canvasEmpty').style.display = 'none';
    state.selectedTableIds.clear();
    renderCanvas();
    if (pushSql) scheduleExportToEditor();
    else setSqlSyncStatus('Synced', 'ok');
  }

  function scheduleExportToEditor() {
    clearTimeout(state.exportTimer);
    state.exportTimer = setTimeout(() => exportToEditor(), SQL_EXPORT_MS);
  }

  async function exportToEditor() {
    if (!state.activeProjectId || state.syncFrom === 'import') return;
    state.syncFrom = 'export';
    setSqlSyncStatus('Updating SQL…');
    try {
      const dialect = $('#exportDialect')?.value || 'mysql';
      const result = await api(`/api/projects/${state.activeProjectId}/export`, {
        method: 'POST',
        body: JSON.stringify({ dialect }),
      });
      const editor = $('#sqlEditor');
      if (editor && editor.value !== result.sql) {
        editor.value = result.sql;
      }
      state.lastSyncedSql = result.sql;
      state.lastExport = result;
      setSqlSyncStatus('Synced', 'ok');
    } catch (err) {
      setSqlSyncStatus(err.message, 'error');
    } finally {
      state.syncFrom = null;
    }
  }

  function scheduleImportFromEditor() {
    clearTimeout(state.importTimer);
    state.importTimer = setTimeout(() => importFromEditor(), SQL_IMPORT_MS);
  }

  async function importFromEditor() {
    if (!state.activeProjectId || state.syncFrom === 'export') return;
    const sql = $('#sqlEditor')?.value.trim() || '';
    if (!sql || sql === state.lastSyncedSql) {
      if (sql === state.lastSyncedSql) setSqlSyncStatus('Synced', 'ok');
      return;
    }
    state.syncFrom = 'import';
    setSqlSyncStatus('Updating diagram…');
    try {
      await api(`/api/projects/${state.activeProjectId}/import`, {
        method: 'POST',
        body: JSON.stringify({ sql, replace: true }),
      });
      state.lastSyncedSql = sql;
      state.schema = await api(`/api/projects/${state.activeProjectId}/schema`);
      renderCanvas();
      setSqlSyncStatus('Synced', 'ok');
    } catch (err) {
      setSqlSyncStatus(err.message, 'error');
    } finally {
      state.syncFrom = null;
    }
  }

  $('#sqlEditor')?.addEventListener('input', () => {
    if (state.syncFrom === 'export') return;
    setSqlSyncStatus('Pending…');
    scheduleImportFromEditor();
  });

  $('#exportDialect')?.addEventListener('change', async () => {
    scheduleExportToEditor();
    if (!state.activeProjectId || !state.schema) return;
    const dialect = $('#exportDialect')?.value || 'mysql';
    try {
      const updated = await api(`/api/projects/${state.activeProjectId}`, {
        method: 'PUT',
        body: JSON.stringify({
          name: state.schema.project.name,
          description: state.schema.project.description || '',
          dialect,
        }),
      });
      state.schema.project.dialect = updated.dialect;
      const cached = state.projects.find((x) => x.id === state.activeProjectId);
      if (cached) cached.dialect = updated.dialect;
    } catch (err) {
      toast(err.message, 'error');
    }
  });

  function getCanvasPoint(e) {
    const cr = $('#canvas')?.getBoundingClientRect();
    if (!cr) return { x: 0, y: 0 };
    return {
      x: e.clientX - cr.left,
      y: e.clientY - cr.top,
    };
  }

  function syncSelectionUI() {
    $$('.table-node').forEach((node) => {
      const id = +node.dataset.id;
      node.classList.toggle('selected', state.selectedTableIds.has(id));
    });
  }

  function setSelection(ids, { additive = false } = {}) {
    if (!additive) state.selectedTableIds.clear();
    ids.forEach((id) => state.selectedTableIds.add(id));
    syncSelectionUI();
  }

  function clearSelection() {
    state.selectedTableIds.clear();
    syncSelectionUI();
  }

  function rectsIntersect(a, b) {
    return a.left < b.right && a.right > b.left && a.top < b.bottom && a.bottom > b.top;
  }

  function startMarquee(e) {
    if (state.linkDrag || state.drag) return;
    e.preventDefault();
    const additive = e.ctrlKey || e.metaKey || e.shiftKey;
    if (!additive) clearSelection();

    const pt = getCanvasPoint(e);
    const box = document.createElement('div');
    box.className = 'selection-box';
    $('#canvas')?.appendChild(box);
    $('#canvas')?.classList.add('selecting');

    state.marquee = { x0: pt.x, y0: pt.y, box, additive };
    document.addEventListener('mousemove', onMarquee);
    document.addEventListener('mouseup', endMarquee);
  }

  function onMarquee(e) {
    if (!state.marquee) return;
    const pt = getCanvasPoint(e);
    const { x0, y0, box } = state.marquee;
    const x = Math.min(x0, pt.x);
    const y = Math.min(y0, pt.y);
    box.style.left = `${x}px`;
    box.style.top = `${y}px`;
    box.style.width = `${Math.abs(pt.x - x0)}px`;
    box.style.height = `${Math.abs(pt.y - y0)}px`;
  }

  function endMarquee(e) {
    document.removeEventListener('mousemove', onMarquee);
    document.removeEventListener('mouseup', endMarquee);
    $('#canvas')?.classList.remove('selecting');
    if (!state.marquee) return;

    const { x0, y0, box, additive } = state.marquee;
    const pt = getCanvasPoint(e);
    box.remove();
    state.marquee = null;

    if (Math.abs(pt.x - x0) < 5 && Math.abs(pt.y - y0) < 5) {
      if (!additive) clearSelection();
      return;
    }

    const selRect = {
      left: Math.min(x0, pt.x),
      top: Math.min(y0, pt.y),
      right: Math.max(x0, pt.x),
      bottom: Math.max(y0, pt.y),
    };
    const ids = [];
    $$('.table-node').forEach((node) => {
      const left = parseFloat(node.style.left) || 0;
      const top = parseFloat(node.style.top) || 0;
      const tr = {
        left,
        top,
        right: left + node.offsetWidth,
        bottom: top + node.offsetHeight,
      };
      if (rectsIntersect(selRect, tr)) ids.push(+node.dataset.id);
    });
    setSelection(ids, { additive });
  }

  function bindCanvasSelection() {
    const canvas = $('#canvas');
    canvas?.addEventListener('mousedown', (e) => {
      if (e.button !== 0 || state.linkDrag) return;
      if (e.target.closest('.table-node')) return;
      startMarquee(e);
    });
  }

  document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') clearSelection();
  });

  function renderCanvas() {
    if (!state.schema) {
      $('#canvas').innerHTML = '';
      $('#canvasEmpty').style.display = 'flex';
      drawRelations();
      return;
    }
    const canvas = $('#canvas');
    canvas.innerHTML = state.schema.tables.map((t) => renderTableNode(t)).join('');

    canvas.querySelectorAll('.table-node').forEach((node) => {
      const id = +node.dataset.id;
      const header = node.querySelector('.table-header');
      header?.addEventListener('mousedown', (e) => {
        if (e.target.closest('.col-list')) return;
        startDrag(e, id, node);
      });
      node.addEventListener('dblclick', (e) => {
        if (!e.target.closest('.col-list')) openTableModal(id);
      });
      node.querySelectorAll('.col-list li').forEach((li) => {
        li.addEventListener('mousedown', (e) => startLinkDrag(e, id, +li.dataset.colId, li));
      });
    });
    syncSelectionUI();
    drawRelations();
  }

  function renderTableNode(t) {
    const cols = (t.columns || []).map((c) => {
      let badges = '';
      if (c.is_primary_key) badges += '<span class="badge-pk">PK</span>';
      if (c.is_foreign_key) badges += '<span class="badge-fk">FK</span>';
      if (c.is_unique && !c.is_primary_key) badges += '<span class="badge-uq">UQ</span>';
      if (c.is_auto_increment) badges += '<span class="badge-ai">AI</span>';
      const typeLabel = formatDataTypeDisplay(c.data_type);
      const tip = `${c.name} — ${c.data_type}\nDrag column to link FK → PK`;
      return `<li class="col-port" data-col-id="${c.id}" data-table-id="${t.id}" title="${esc(tip)}">${badges}<span class="col-name" title="${esc(c.name)}">${esc(c.name)}</span><span class="col-type">${esc(typeLabel)}</span></li>`;
    }).join('');
    const indexes = renderTableIndexSection(t);
    const tableW = canvasTableWidth(t);
    return `
      <div class="table-node" data-id="${t.id}" style="left:${t.pos_x}px;top:${t.pos_y}px;width:${tableW}px">
        <div class="table-header"><span class="table-title" title="${esc(t.name)}">${esc(t.name)}</span> <span class="edit-hint">dbl-click</span></div>
        <ul class="col-list">${cols}</ul>
        ${indexes}
      </div>`;
  }

  function indexColumnNames(idx, columns) {
    return (idx.column_ids || [])
      .map((id) => columns.find((x) => x.id === id)?.name)
      .filter(Boolean);
  }

  function renderTableIndexSection(t) {
    const columns = t.columns || [];
    const list = [...(t.indexes || [])].sort((a, b) => (a.sort_order - b.sort_order) || (a.id - b.id));
    if (!list.length) return '';
    const items = list.map((idx) => {
      const names = indexColumnNames(idx, columns);
      const kind = idx.is_unique ? 'UQ' : 'IDX';
      const label = `${idx.name} (${names.join(', ') || '—'})`;
      const tip = `${idx.name}${idx.is_unique ? ' (unique)' : ''}\nColumns: ${names.join(', ') || '—'}`;
      return `<li class="idx-item" data-idx-id="${idx.id}" title="${esc(tip)}">
        <span class="badge-idx${idx.is_unique ? ' badge-idx-uq' : ''}">${kind}</span>
        <span class="idx-item-label">${esc(label)}</span>
      </li>`;
    }).join('');
    return `<div class="idx-section"><div class="idx-section-head">Indexes</div><ul class="idx-list">${items}</ul></div>`;
  }

  function startDrag(e, tableId, node) {
    if (e.button !== 0 || state.linkDrag) return;
    e.preventDefault();
    e.stopPropagation();

    const additive = e.ctrlKey || e.metaKey || e.shiftKey;
    if (additive) {
      if (state.selectedTableIds.has(tableId)) state.selectedTableIds.delete(tableId);
      else state.selectedTableIds.add(tableId);
      syncSelectionUI();
      return;
    }

    if (!state.selectedTableIds.has(tableId)) {
      setSelection([tableId]);
    }

    const pt = getCanvasPoint(e);
    const anchorX = parseFloat(node.style.left) || 0;
    const anchorY = parseFloat(node.style.top) || 0;
    const nodes = [];
    state.selectedTableIds.forEach((id) => {
      const n = document.querySelector(`.table-node[data-id="${id}"]`);
      if (!n) return;
      nodes.push({
        id,
        node: n,
        startX: parseFloat(n.style.left) || 0,
        startY: parseFloat(n.style.top) || 0,
      });
    });

    state.drag = {
      tableId,
      nodes,
      offX: pt.x - anchorX,
      offY: pt.y - anchorY,
      anchorX,
      anchorY,
    };
    document.addEventListener('mousemove', onDrag);
    document.addEventListener('mouseup', endDrag);
  }

  function onDrag(e) {
    if (!state.drag) return;
    const pt = getCanvasPoint(e);
    const anchorX = Math.max(0, pt.x - state.drag.offX);
    const anchorY = Math.max(0, pt.y - state.drag.offY);
    const dx = anchorX - state.drag.anchorX;
    const dy = anchorY - state.drag.anchorY;

    state.drag.nodes.forEach(({ node, startX, startY }) => {
      node.style.left = `${Math.max(0, startX + dx)}px`;
      node.style.top = `${Math.max(0, startY + dy)}px`;
    });
    drawRelations();
  }

  async function endDrag() {
    document.removeEventListener('mousemove', onDrag);
    document.removeEventListener('mouseup', endDrag);
    if (!state.drag) return;

    const updates = state.drag.nodes.map(({ id, node }) => ({
      id,
      posX: parseFloat(node.style.left) || 0,
      posY: parseFloat(node.style.top) || 0,
    }));
    state.drag = null;

    try {
      await Promise.all(
        updates.map((u) =>
          api(`/api/tables/${u.id}`, {
            method: 'PUT',
            body: JSON.stringify({ pos_x: u.posX, pos_y: u.posY }),
          }),
        ),
      );
      updates.forEach((u) => {
        const t = state.schema.tables.find((x) => x.id === u.id);
        if (t) {
          t.pos_x = u.posX;
          t.pos_y = u.posY;
        }
      });
    } catch (err) {
      toast(err.message, 'error');
      await loadSchema({ pushSql: false });
    }
  }

  function colRectInCanvas(el, canvas) {
    const canvasRect = canvas.getBoundingClientRect();
    const r = el.getBoundingClientRect();
    return {
      left: r.left - canvasRect.left,
      top: r.top - canvasRect.top,
      width: r.width,
      height: r.height,
    };
  }

  function colAnchor(tableId, colId, side = 'center') {
    const li = document.querySelector(`.table-node[data-id="${tableId}"] li[data-col-id="${colId}"]`);
    const canvas = $('#canvas');
    if (!li || !canvas) return null;

    const r = colRectInCanvas(li, canvas);
    const cy = r.top + r.height / 2;
    if (side === 'right') return { x: r.left + r.width, y: cy };
    if (side === 'left') return { x: r.left, y: cy };
    return { x: r.left + r.width / 2, y: cy };
  }

  function relationAnchors(fromTableId, fromColId, toTableId, toColId) {
    const aCenter = colAnchor(fromTableId, fromColId, 'center');
    const bCenter = colAnchor(toTableId, toColId, 'center');
    if (!aCenter || !bCenter) return null;
    if (aCenter.x <= bCenter.x) {
      return {
        a: colAnchor(fromTableId, fromColId, 'right'),
        b: colAnchor(toTableId, toColId, 'left'),
      };
    }
    return {
      a: colAnchor(fromTableId, fromColId, 'left'),
      b: colAnchor(toTableId, toColId, 'right'),
    };
  }

  function resizeSvgToCanvas() {
    const canvas = $('#canvas');
    const svg = $('#relationSvg');
    if (!canvas || !svg) return;
    const w = canvas.offsetWidth;
    const h = canvas.offsetHeight;
    svg.setAttribute('width', w);
    svg.setAttribute('height', h);
    svg.style.width = w + 'px';
    svg.style.height = h + 'px';
  }

  function pathD(x1, y1, x2, y2) {
    const midX = x1 + (x2 - x1) / 2;
    return `M${x1},${y1} C${midX},${y1} ${midX},${y2} ${x2},${y2}`;
  }

  function relationArrowHead(x1, y1, x2, y2, size = 7) {
    const dir = x2 >= x1 ? 1 : -1;
    const baseX = x2 - dir * size;
    return `M${x2},${y2} L${baseX},${y2 - size * 0.55} L${baseX},${y2 + size * 0.55} Z`;
  }

  function relationPathSvg(x1, y1, x2, y2, stroke, strokeWidth, arrowFill) {
    return [
      `<path d="${pathD(x1, y1, x2, y2)}" stroke="${stroke}" stroke-width="${strokeWidth}" fill="none"/>`,
      `<path d="${relationArrowHead(x1, y1, x2, y2)}" fill="${arrowFill}" stroke="none"/>`,
    ].join('');
  }

  function ensureArrowMarker(svg) {
    if (svg.querySelector('#arrow')) return;
    const defs = document.createElementNS('http://www.w3.org/2000/svg', 'defs');
    defs.innerHTML = '<marker id="arrow" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0,0 L6,3 L0,6 Z" fill="#a371f7"/></marker>';
    svg.appendChild(defs);
  }

  async function deleteRelation(relId) {
    if (!confirm('Delete this relation?')) return;
    try {
      await api(`/api/relations/${relId}`, { method: 'DELETE' });
      await loadSchema();
      toast('Relation deleted');
    } catch (err) {
      toast(err.message, 'error');
    }
  }

  function relationSummary(fromTableId, fromColId, toTableId, toColId) {
    const fromTable = state.schema?.tables?.find((t) => t.id === fromTableId);
    const toTable = state.schema?.tables?.find((t) => t.id === toTableId);
    const fromCol = fromTable?.columns?.find((c) => c.id === fromColId);
    const toCol = toTable?.columns?.find((c) => c.id === toColId);
    return `${fromTable?.name || '?'}.${fromCol?.name || '?'} → ${toTable?.name || '?'}.${toCol?.name || '?'}`;
  }

  function setFKSelect(select, value) {
    if (!select) return;
    const v = value || 'RESTRICT';
    if ([...select.options].some((o) => o.value === v)) select.value = v;
    else select.value = 'RESTRICT';
  }

  function defaultFKName(fromTableId, fromColId) {
    const fromTable = state.schema?.tables?.find((t) => t.id === fromTableId);
    const fromCol = fromTable?.columns?.find((c) => c.id === fromColId);
    const t = (fromTable?.name || 'table').toLowerCase().replace(/[^a-z0-9_]+/g, '_').replace(/^_+|_+$/g, '') || 'table';
    const c = (fromCol?.name || 'col').toLowerCase().replace(/[^a-z0-9_]+/g, '_').replace(/^_+|_+$/g, '') || 'col';
    return `fk_${t}_${c}`;
  }

  function closeRelationModal() {
    state.editingRelationId = null;
    state.pendingRelation = null;
    $('#relationModal')?.close();
  }

  function openRelationModal(relOrPending) {
    const modal = $('#relationModal');
    if (!modal || !relOrPending) return;
    const isExisting = !!relOrPending.id;
    state.editingRelationId = isExisting ? relOrPending.id : null;
    state.pendingRelation = isExisting ? null : relOrPending;

    $('#relationModalTitle').textContent = isExisting ? 'Edit foreign key' : 'New foreign key';
    const delBtn = $('#btnDeleteRelation');
    if (delBtn) delBtn.hidden = !isExisting;

    $('#relationModalSummary').textContent = relationSummary(
      relOrPending.from_table_id,
      relOrPending.from_column_id,
      relOrPending.to_table_id,
      relOrPending.to_column_id,
    );
    setFKSelect($('#relOnDelete'), isExisting ? relOrPending.on_delete : 'RESTRICT');
    setFKSelect($('#relOnUpdate'), isExisting ? relOrPending.on_update : 'RESTRICT');
    const nameInput = $('#relName');
    if (nameInput) {
      nameInput.value = isExisting
        ? (relOrPending.name || defaultFKName(relOrPending.from_table_id, relOrPending.from_column_id))
        : defaultFKName(relOrPending.from_table_id, relOrPending.from_column_id);
    }
    modal.showModal();
  }

  async function reloadSchemaFromServer() {
    clearTimeout(state.exportTimer);
    clearTimeout(state.importTimer);
    await loadSchema({ pushSql: true });
    toast('Schema reloaded');
  }

  function drawRelationPair(svg, x1, y1, x2, y2, relId) {
    const d = pathD(x1, y1, x2, y2);
    const hit = document.createElementNS('http://www.w3.org/2000/svg', 'path');
    hit.setAttribute('d', d);
    hit.setAttribute('class', 'rel-hit');
    hit.addEventListener('click', (e) => {
      e.stopPropagation();
      const rel = state.schema?.relations?.find((r) => r.id === relId);
      if (rel) openRelationModal(rel);
    });

    const line = document.createElementNS('http://www.w3.org/2000/svg', 'path');
    line.setAttribute('d', d);
    line.setAttribute('class', 'rel-line');
    line.setAttribute('marker-end', 'url(#arrow)');

    svg.appendChild(hit);
    svg.appendChild(line);
  }

  function drawPreviewPath(svg, x1, y1, x2, y2) {
    const path = document.createElementNS('http://www.w3.org/2000/svg', 'path');
    path.setAttribute('d', pathD(x1, y1, x2, y2));
    path.setAttribute('class', 'link-preview');
    svg.appendChild(path);
  }

  function drawRelations() {
    const svg = $('#relationSvg');
    if (!svg) return;
    resizeSvgToCanvas();
    svg.innerHTML = '';
    ensureArrowMarker(svg);

    if (!state.schema) return;

    (state.schema.relations || []).forEach((rel) => {
      const anchors = relationAnchors(
        rel.from_table_id, rel.from_column_id,
        rel.to_table_id, rel.to_column_id,
      );
      if (!anchors?.a || !anchors?.b) return;
      drawRelationPair(svg, anchors.a.x, anchors.a.y, anchors.b.x, anchors.b.y, rel.id);
    });

    if (state.linkDrag?.preview) {
      const p = state.linkDrag.preview;
      drawPreviewPath(svg, p.x1, p.y1, p.x2, p.y2);
    }
  }

  function startLinkDrag(e, tableId, columnId, li) {
    if (e.button !== 0) return;
    e.preventDefault();
    e.stopPropagation();
    const start = colAnchor(tableId, columnId, 'right');
    if (!start) return;
    state.linkDrag = {
      fromTableId: tableId,
      fromColumnId: columnId,
      preview: { x1: start.x, y1: start.y, x2: start.x, y2: start.y },
    };
    li.classList.add('col-link-source');
    document.addEventListener('mousemove', onLinkDrag);
    document.addEventListener('mouseup', endLinkDrag);
    drawRelations();
  }

  function pointerInCanvas(e) {
    return getCanvasPoint(e);
  }

  function onLinkDrag(e) {
    if (!state.linkDrag) return;
    const pt = pointerInCanvas(e);
    state.linkDrag.preview.x2 = pt.x;
    state.linkDrag.preview.y2 = pt.y;
    $$('.col-port').forEach((el) => el.classList.remove('col-link-target', 'col-link-invalid'));
    const hit = document.elementFromPoint(e.clientX, e.clientY)?.closest('.col-port');
    if (hit) {
      const toTable = +hit.dataset.tableId;
      if (toTable === state.linkDrag.fromTableId) hit.classList.add('col-link-invalid');
      else hit.classList.add('col-link-target');
      state.linkDrag.hover = { tableId: toTable, columnId: +hit.dataset.colId, el: hit };
    } else {
      state.linkDrag.hover = null;
    }
    drawRelations();
  }

  async function endLinkDrag() {
    document.removeEventListener('mousemove', onLinkDrag);
    document.removeEventListener('mouseup', endLinkDrag);
    $$('.col-port').forEach((el) => el.classList.remove('col-link-source', 'col-link-target', 'col-link-invalid'));
    if (!state.linkDrag) return;

    const from = { tableId: state.linkDrag.fromTableId, columnId: state.linkDrag.fromColumnId };
    const hover = state.linkDrag.hover;
    state.linkDrag = null;
    drawRelations();

    if (!hover) return;
    if (hover.tableId === from.tableId) {
      toast('Cannot link column to same table', 'error');
      return;
    }
    openRelationModal({
      from_table_id: from.tableId,
      from_column_id: from.columnId,
      to_table_id: hover.tableId,
      to_column_id: hover.columnId,
    });
  }

  const LAYOUT = {
    headerH: 36,
    colListPad: 8,
    colRowH: 26,
    indexHeadH: 22,
    indexRowH: 21,
    indexPad: 6,
    bottomPad: 8,
    gapY: 44,
    gapX: 96,
    tableW: 260,
    componentGap: 72,
  };

  const CANVAS_TABLE = {
    minW: 200,
    maxW: 280,
  };

  const EXPORT_TABLE = {
    minW: 200,
    maxW: 2400,
  };

  function measureTextWidth(text, fontSize, fontFamily) {
    const canvas = document.createElement('canvas');
    const ctx = canvas.getContext('2d');
    if (!ctx) return String(text || '').length * fontSize * 0.55;
    ctx.font = `${fontSize}px ${fontFamily}`;
    return ctx.measureText(String(text || '')).width;
  }

  function estimateTableWidth(table, options = {}) {
    const headerFont = options.headerFont || 'Segoe UI, sans-serif';
    const mono = 'Consolas, monospace';
    const minW = options.minW || 200;
    const maxCap = options.maxW || 520;
    let maxW = minW;

    maxW = Math.max(maxW, measureTextWidth(table.name, 13, headerFont) + 60);

    (table.columns || []).forEach((col) => {
      let badges = 0;
      if (col.is_primary_key) badges += 26;
      if (col.is_foreign_key) badges += 26;
      if (col.is_unique && !col.is_primary_key) badges += 26;
      if (col.is_auto_increment) badges += 26;
      maxW = Math.max(
        maxW,
        12 + badges + measureTextWidth(col.name, 11, mono) + measureTextWidth(formatDataTypeDisplay(col.data_type), 10, mono) + 24,
      );
    });

    (table.indexes || []).forEach((idx) => {
      const cols = indexColumnNames(idx, table.columns || []);
      const label = `${idx.name} (${cols.join(', ') || '—'})`;
      maxW = Math.max(maxW, measureTextWidth(label, 10, mono) + 42);
    });

    return Math.ceil(Math.min(Math.max(maxW, minW), maxCap));
  }

  function canvasTableWidth(table) {
    const headerNeed = Math.ceil(measureTextWidth(table.name, 13, 'Segoe UI, sans-serif') + 72);
    return Math.min(CANVAS_TABLE.maxW, Math.max(CANVAS_TABLE.minW, headerNeed));
  }

  function exportTableWidth(table, styleKey, baseW) {
    const headerFont = styleKey === 'minimal'
      ? 'Georgia, Times New Roman, serif'
      : 'Segoe UI, sans-serif';
    return estimateTableWidth(table, {
      headerFont,
      minW: Math.max(baseW || 0, EXPORT_TABLE.minW),
      maxW: EXPORT_TABLE.maxW,
    });
  }

  function estimateTableHeight(table) {
    const cols = Math.max(table?.columns?.length || 0, 1);
    let h = LAYOUT.headerH + LAYOUT.colListPad + cols * LAYOUT.colRowH + LAYOUT.bottomPad;
    const idxCount = table?.indexes?.length || 0;
    if (idxCount > 0) {
      h += LAYOUT.indexHeadH + idxCount * LAYOUT.indexRowH + LAYOUT.indexPad;
    }
    return h;
  }

  function layoutComponent(tables, relations, opts = {}) {
    const tableWidth = opts.tableWidth || canvasTableWidth;
    const tableHeight = opts.tableHeight || estimateTableHeight;
    const GAP_X = opts.gapX ?? LAYOUT.gapX;
    const GAP_Y = opts.gapY ?? LAYOUT.gapY;
    const START_X = opts.startX ?? 48;
    const START_Y = opts.startY ?? 0;
    const layer = new Map();
    tables.forEach((t) => layer.set(t.id, 0));

    for (let pass = 0; pass < tables.length; pass++) {
      relations.forEach((rel) => {
        const from = rel.from_table_id;
        const to = rel.to_table_id;
        const next = (layer.get(to) || 0) + 1;
        if (next > (layer.get(from) || 0)) layer.set(from, next);
      });
    }

    const layers = new Map();
    tables.forEach((t) => {
      const L = layer.get(t.id) || 0;
      if (!layers.has(L)) layers.set(L, []);
      layers.get(L).push(t);
    });

    const TABLE_W = LAYOUT.tableW;
    const positions = new Map();
    const sortedLayers = [...layers.keys()].sort((a, b) => a - b);
    let layerX = START_X;

    sortedLayers.forEach((L) => {
      let group = [...layers.get(L)];
      if (L > 0) {
        group.sort((a, b) => {
          const avgTargetY = (tableId) => {
            const targets = relations
              .filter((r) => r.from_table_id === tableId)
              .map((r) => positions.get(r.to_table_id)?.y ?? 0);
            if (!targets.length) return 0;
            return targets.reduce((s, v) => s + v, 0) / targets.length;
          };
          return avgTargetY(a.id) - avgTargetY(b.id) || a.name.localeCompare(b.name);
        });
      } else {
        group.sort((a, b) => a.name.localeCompare(b.name));
      }

      const layerMaxW = Math.max(TABLE_W, ...group.map((t) => tableWidth(t)));
      const x = layerX;
      let y = START_Y;
      group.forEach((t) => {
        positions.set(t.id, { x, y });
        y += tableHeight(t) + GAP_Y;
      });
      layerX += layerMaxW + GAP_X;
    });

    return positions;
  }

  function findTableComponents(tables, relations) {
    const parent = new Map();
    const find = (id) => {
      if (!parent.has(id)) parent.set(id, id);
      if (parent.get(id) !== id) parent.set(id, find(parent.get(id)));
      return parent.get(id);
    };
    const union = (a, b) => {
      parent.set(find(a), find(b));
    };
    tables.forEach((t) => find(t.id));
    relations.forEach((r) => union(r.from_table_id, r.to_table_id));

    const groups = new Map();
    tables.forEach((t) => {
      const root = find(t.id);
      if (!groups.has(root)) groups.set(root, []);
      groups.get(root).push(t);
    });
    return [...groups.values()].sort((a, b) => b.length - a.length);
  }

  function computeAutoLayout(schema, opts = {}) {
    const tableHeight = opts.tableHeight || estimateTableHeight;
    const tables = schema.tables || [];
    const relations = schema.relations || [];
    if (!tables.length) return new Map();

    const components = findTableComponents(tables, relations);
    const positions = new Map();
    const COMPONENT_GAP = opts.componentGap ?? LAYOUT.componentGap;
    let offsetY = opts.offsetY ?? 48;

    components.forEach((compTables) => {
      const compIds = new Set(compTables.map((t) => t.id));
      const compRelations = relations.filter(
        (r) => compIds.has(r.from_table_id) && compIds.has(r.to_table_id),
      );
      const local = layoutComponent(compTables, compRelations, opts);
      let blockMaxY = 0;

      compTables.forEach((t) => {
        const pos = local.get(t.id) || { x: 48, y: 0 };
        const y = pos.y + offsetY;
        positions.set(t.id, { x: pos.x, y });
        blockMaxY = Math.max(blockMaxY, y + tableHeight(t));
      });

      offsetY = blockMaxY + COMPONENT_GAP;
    });

    return positions;
  }

  function refinePositionsByLayer(positions, schema, tableHeight, gapY = LAYOUT.gapY) {
    const nodes = (schema?.tables || []).map((table) => {
      const pos = positions.get(table.id);
      if (!pos) return null;
      return {
        id: table.id,
        x: pos.x,
        y: pos.y,
        h: tableHeight(table),
      };
    }).filter(Boolean);
    if (!nodes.length) return;

    const xBucket = (x) => Math.round(x / 40) * 40;
    const layers = new Map();
    nodes.forEach((n) => {
      const key = xBucket(n.x);
      if (!layers.has(key)) layers.set(key, []);
      layers.get(key).push(n);
    });

    layers.forEach((group) => {
      group.sort((a, b) => a.y - b.y || a.id - b.id);
      let cursor = group[0]?.y ?? 0;
      group.forEach((n, i) => {
        if (i === 0) {
          cursor = n.y;
        } else {
          const minY = cursor + gapY;
          if (n.y < minY) n.y = minY;
          else cursor = n.y;
        }
        cursor = n.y + n.h;
        positions.set(n.id, { x: n.x, y: n.y });
      });
    });
  }

  async function applyAutoLayout() {
    if (!state.activeProjectId || !state.schema?.tables?.length) {
      return toast('No tables to layout', 'error');
    }

    const positions = computeAutoLayout(state.schema);
    try {
      state.schema.tables.forEach((t) => {
        const pos = positions.get(t.id);
        if (!pos) return;
        t.pos_x = pos.x;
        t.pos_y = pos.y;
      });
      renderCanvas();
      refineLayoutFromDOM();
      await Promise.all(
        state.schema.tables.map((t) =>
          api(`/api/tables/${t.id}`, {
            method: 'PUT',
            body: JSON.stringify({ pos_x: t.pos_x, pos_y: t.pos_y }),
          }),
        ),
      );
      renderCanvas();
      scheduleExportToEditor();
      $('#canvasScroll')?.scrollTo({ top: 0, left: 0, behavior: 'smooth' });
      toast('Tables arranged');
    } catch (err) {
      toast(err.message, 'error');
    }
  }

  function refineLayoutFromDOM() {
    const canvas = $('#canvas');
    if (!canvas || !state.schema?.tables?.length) return;

    const positions = new Map();
    const heights = new Map();
    canvas.querySelectorAll('.table-node').forEach((node) => {
      const id = +node.dataset.id;
      const table = state.schema.tables.find((t) => t.id === id);
      if (!table) return;
      positions.set(id, {
        x: parseFloat(node.style.left) || 0,
        y: parseFloat(node.style.top) || 0,
      });
      heights.set(id, node.offsetHeight || estimateTableHeight(table));
    });

    refinePositionsByLayer(positions, state.schema, (table) => heights.get(table.id) || estimateTableHeight(table));

    positions.forEach((pos, id) => {
      const node = canvas.querySelector(`.table-node[data-id="${id}"]`);
      const table = state.schema.tables.find((t) => t.id === id);
      if (!node || !table) return;
      node.style.top = `${pos.y}px`;
      table.pos_x = pos.x;
      table.pos_y = pos.y;
    });
  }

  $('#btnAutoLayout')?.addEventListener('click', () => applyAutoLayout());

  $('#btnAddTable')?.addEventListener('click', async () => {
    const name = prompt('Table name:');
    if (!name?.trim()) return;
    const n = state.schema.tables.length;
    try {
      await api(`/api/projects/${state.activeProjectId}/tables`, {
        method: 'POST',
        body: JSON.stringify({ name: name.trim(), pos_x: 40 + n * 30, pos_y: 40 + n * 30 }),
      });
      await loadSchema();
      toast('Table added');
    } catch (err) {
      toast(err.message, 'error');
    }
  });

  $('#btnReloadSchema')?.addEventListener('click', () => reloadSchemaFromServer());

  $('#btnCopySQL')?.addEventListener('click', async () => {
    const t = $('#sqlEditor').value;
    if (!t.trim()) return toast('No SQL to copy', 'error');
    await navigator.clipboard.writeText(t);
    toast('Copied');
  });

  $('#btnDownloadSQL')?.addEventListener('click', () => {
    const t = $('#sqlEditor').value;
    if (!t.trim()) return toast('No SQL to download', 'error');
    const a = document.createElement('a');
    a.href = URL.createObjectURL(new Blob([t], { type: 'text/sql' }));
    a.download = state.lastExport?.filename || 'schema.sql';
    a.click();
  });

  // --- Diagram image export ---
  function measureExportDiagram(styleKey) {
    const canvasLayout = measureDiagram();
    if (!canvasLayout) return null;

    const gapX = LAYOUT.gapX;
    const gapY = LAYOUT.gapY;
    const pad = 40;

    const entries = canvasLayout.tables.map((entry) => ({
      ...entry,
      exportW: exportTableWidth(entry.table, styleKey, entry.w),
    }));

    const syncExportTableAnchors = (entry) => {
      const left = entry.x;
      const right = entry.x + entry.exportW;
      entry.columns.forEach((c) => {
        c.anchorLeft = left;
        c.anchorRight = right;
      });
    };

    const shiftEntryX = (entry, newX) => {
      if (newX === entry.x) return;
      entry.x = newX;
      syncExportTableAnchors(entry);
    };

    const shiftEntryY = (entry, newY) => {
      const deltaY = newY - entry.y;
      if (!deltaY) return;
      entry.y = newY;
      entry.columns.forEach((c) => {
        c.anchorY += deltaY;
      });
    };

    entries.forEach((entry) => syncExportTableAnchors(entry));

    const xBucket = (x) => Math.round(x / 40) * 40;
    const colGroups = new Map();
    entries.forEach((e) => {
      const key = xBucket(e.x);
      if (!colGroups.has(key)) colGroups.set(key, []);
      colGroups.get(key).push(e);
    });

    const sortedKeys = [...colGroups.keys()].sort((a, b) => a - b);
    let prevRight = -Infinity;

    sortedKeys.forEach((key) => {
      const group = colGroups.get(key);
      const colX = Math.min(...group.map((e) => e.x));
      const colMaxW = Math.max(...group.map((e) => e.exportW));
      let newX = colX;
      if (prevRight > -Infinity && newX < prevRight + gapX) {
        newX = prevRight + gapX;
      }
      group.forEach((e) => shiftEntryX(e, newX));
      prevRight = newX + colMaxW;
    });

    sortedKeys.forEach((key) => {
      const group = colGroups.get(key);
      group.sort((a, b) => a.y - b.y || a.table.id - b.table.id);
      let cursor = group[0]?.y ?? 0;
      group.forEach((e, i) => {
        if (i === 0) {
          cursor = e.y;
        } else {
          const minY = cursor + gapY;
          if (e.y < minY) shiftEntryY(e, minY);
          else cursor = e.y;
        }
        cursor = e.y + e.h;
      });
    });

    entries.forEach((entry) => syncExportTableAnchors(entry));

    let minX = Infinity;
    let minY = Infinity;
    let maxX = -Infinity;
    let maxY = -Infinity;
    entries.forEach((e) => {
      minX = Math.min(minX, e.x);
      minY = Math.min(minY, e.y);
      maxX = Math.max(maxX, e.x + e.exportW);
      maxY = Math.max(maxY, e.y + e.h);
    });

    const ox = minX - pad;
    const oy = minY - pad;
    return {
      tables: entries,
      ox,
      oy,
      width: maxX - minX + pad * 2,
      height: maxY - minY + pad * 2,
      relations: canvasLayout.relations,
    };
  }

  function measureDiagram() {
    if (!state.schema?.tables?.length) return null;
    const canvas = $('#canvas');
    const nodes = canvas?.querySelectorAll('.table-node');
    if (!nodes?.length || !canvas) return null;

    const pad = 40;
    let minX = Infinity;
    let minY = Infinity;
    let maxX = -Infinity;
    let maxY = -Infinity;
    const tables = [];

    nodes.forEach((node) => {
      const id = +node.dataset.id;
      const table = state.schema.tables.find((t) => t.id === id);
      if (!table) return;

      const nodeRect = colRectInCanvas(node, canvas);
      const x = nodeRect.left;
      const y = nodeRect.top;
      const w = nodeRect.width;
      const h = nodeRect.height;
      minX = Math.min(minX, x);
      minY = Math.min(minY, y);
      maxX = Math.max(maxX, x + w);
      maxY = Math.max(maxY, y + h);

      const header = node.querySelector('.table-header');
      const headerH = header ? colRectInCanvas(header, canvas).height : 36;
      const columns = [];
      node.querySelectorAll('.col-list li').forEach((li) => {
        const colId = +li.dataset.colId;
        const col = table.columns.find((c) => c.id === colId);
        const lr = colRectInCanvas(li, canvas);
        columns.push({
          col,
          rowY: lr.top - y,
          rowH: lr.height,
          anchorLeft: lr.left,
          anchorRight: lr.left + lr.width,
          anchorY: lr.top + lr.height / 2,
        });
      });

      let idxSection = null;
      const idxSecEl = node.querySelector('.idx-section');
      if (idxSecEl && (table.indexes?.length || 0) > 0) {
        const secRect = colRectInCanvas(idxSecEl, canvas);
        const headEl = idxSecEl.querySelector('.idx-section-head');
        const headRect = headEl ? colRectInCanvas(headEl, canvas) : null;
        const rows = [];
        idxSecEl.querySelectorAll('.idx-item').forEach((li) => {
          const idxId = +li.dataset.idxId;
          const index = table.indexes.find((i) => i.id === idxId);
          const lr = colRectInCanvas(li, canvas);
          rows.push({ index, rowY: lr.top - y, rowH: lr.height });
        });
        idxSection = {
          sectionY: secRect.top - y,
          headH: headRect ? headRect.height : 20,
          rows,
        };
      }

      tables.push({ table, x, y, w, h, headerH, columns, idxSection });
    });

    if (!tables.length) return null;

    const ox = minX - pad;
    const oy = minY - pad;
    return {
      tables,
      ox,
      oy,
      width: maxX - minX + pad * 2,
      height: maxY - minY + pad * 2,
      relations: state.schema.relations || [],
    };
  }

  function buildFallbackIdxSection(table, columns, headerH) {
    const list = [...(table.indexes || [])].sort((a, b) => (a.sort_order - b.sort_order) || (a.id - b.id));
    if (!list.length) return null;
    const lastCol = columns.reduce((best, c) => (!best || c.rowY > best.rowY ? c : best), null);
    const startY = lastCol ? lastCol.rowY + lastCol.rowH + 2 : headerH + 8;
    return {
      sectionY: startY,
      headH: 22,
      rows: list.map((index, i) => ({
        index,
        rowY: startY + 22 + i * 21,
        rowH: 21,
      })),
    };
  }

  function resolveIdxExportData(table, columns, headerH, idxSection) {
    let idxData = idxSection;
    if (!idxData?.rows?.length && (table.indexes?.length || 0) > 0) {
      idxData = buildFallbackIdxSection(table, columns, headerH);
    }
    return idxData;
  }

  function computeExportTableWidth(table, columns, idxData, baseW, styleKey) {
    return exportTableWidth(table, styleKey, baseW);
  }

  function buildDiagramSvg(layout, styleKey) {
    const theme = DIAGRAM_THEMES[styleKey] || DIAGRAM_THEMES.dark;
    const { tables, ox, oy, width, height, relations } = layout;
    const titleOffset = theme.titleBlock ? 58 : 0;
    const totalH = height + titleOffset;
    const projectName = state.schema?.project?.name || 'Database Schema';

    function exportRelationAnchors(fromColId, toColId) {
      let fromCol;
      let toCol;
      tables.forEach((t) => {
        t.columns.forEach((c) => {
          if (c.col?.id === fromColId) fromCol = c;
          if (c.col?.id === toColId) toCol = c;
        });
      });
      if (!fromCol || !toCol) return null;
      const fromCx = (fromCol.anchorLeft + fromCol.anchorRight) / 2;
      const toCx = (toCol.anchorLeft + toCol.anchorRight) / 2;
      if (fromCx <= toCx) {
        return {
          a: { x: fromCol.anchorRight - ox, y: fromCol.anchorY - oy + titleOffset },
          b: { x: toCol.anchorLeft - ox, y: toCol.anchorY - oy + titleOffset },
        };
      }
      return {
        a: { x: fromCol.anchorLeft - ox, y: fromCol.anchorY - oy + titleOffset },
        b: { x: toCol.anchorRight - ox, y: toCol.anchorY - oy + titleOffset },
      };
    }

    const parts = [];

    const exportTables = tables.map((entry) => {
      const idxData = resolveIdxExportData(entry.table, entry.columns, entry.headerH, entry.idxSection);
      const exportW = entry.exportW || computeExportTableWidth(entry.table, entry.columns, idxData, entry.w, styleKey);
      return { ...entry, idxData, exportW };
    });

    let layoutWidth = width;
    exportTables.forEach(({ x, exportW }) => {
      layoutWidth = Math.max(layoutWidth, x - ox + exportW + 40);
    });

    parts.push(`<svg xmlns="http://www.w3.org/2000/svg" width="${layoutWidth}" height="${totalH}" viewBox="0 0 ${layoutWidth} ${totalH}">`);
    parts.push('<defs>');
    if (theme.grid) {
      parts.push(`<pattern id="grid" width="20" height="20" patternUnits="userSpaceOnUse"><circle cx="1" cy="1" r="1" fill="${theme.gridDot}"/></pattern>`);
    }
    parts.push('</defs>');
    parts.push(`<rect width="100%" height="100%" fill="${theme.bg}"/>`);
    if (theme.grid) {
      parts.push(`<rect width="100%" height="100%" fill="url(#grid)"/>`);
    }

    if (theme.titleBlock) {
      parts.push(`<text x="${layoutWidth / 2}" y="26" text-anchor="middle" fill="${theme.text}" font-family="Georgia, 'Times New Roman', serif" font-size="17" font-weight="600">${escXml(projectName)}</text>`);
      parts.push(`<text x="${layoutWidth / 2}" y="44" text-anchor="middle" fill="${theme.textMuted}" font-family="Segoe UI, system-ui, sans-serif" font-size="11">Entity Relationship Diagram</text>`);
      parts.push(`<line x1="32" y1="52" x2="${layoutWidth - 32}" y2="52" stroke="#cccccc" stroke-width="1"/>`);
    }

    exportTables.forEach(({ table, x, y, h, headerH, columns, idxData, exportW }) => {
      const tx = x - ox;
      const ty = y - oy + titleOffset;
      const w = exportW;
      const r = theme.radius;
      const badgeFont = 'Segoe UI, sans-serif';
      parts.push('<g>');
      parts.push(`<rect x="${tx}" y="${ty}" width="${w}" height="${h}" rx="${r}" fill="${theme.tableFill}" stroke="${theme.tableStroke}" stroke-width="1"/>`);
      parts.push(`<rect x="${tx}" y="${ty}" width="${w}" height="${headerH}" rx="${r}" fill="${theme.headerFill}" stroke="none"/>`);
      parts.push(`<line x1="${tx}" y1="${ty + headerH}" x2="${tx + w}" y2="${ty + headerH}" stroke="${theme.headerLine}" stroke-width="1"/>`);
      const headerFont = styleKey === 'minimal'
        ? 'Georgia, \'Times New Roman\', serif'
        : 'Segoe UI, system-ui, sans-serif';
      parts.push(`<text x="${tx + 12}" y="${ty + headerH / 2 + 5}" fill="${theme.text}" font-family="${headerFont}" font-size="13" font-weight="600">${escXml(table.name)}</text>`);

      const drawBadge = (rowTop, midY, cx, label, fill, textColor) => {
        parts.push(`<rect x="${cx}" y="${rowTop + 4}" width="20" height="13" rx="2" fill="${fill}" stroke="${theme.tableStroke}" stroke-width="0.5"/>`);
        parts.push(`<text x="${cx + 4}" y="${midY}" fill="${textColor}" font-family="${badgeFont}" font-size="8" font-weight="700">${label}</text>`);
        return cx + 26;
      };

      columns.forEach(({ col, rowY, rowH }, rowIdx) => {
        if (!col) return;
        const rowTop = ty + rowY;
        const midY = rowTop + rowH / 2 + 4;
        if (styleKey === 'minimal' && rowIdx > 0) {
          parts.push(`<line x1="${tx + 8}" y1="${rowTop}" x2="${tx + w - 8}" y2="${rowTop}" stroke="#e8e8e8" stroke-width="1"/>`);
        }
        let cx = tx + 12;
        if (col.is_primary_key) cx = drawBadge(rowTop, midY, cx, 'PK', theme.pkFill, theme.pkText);
        if (col.is_foreign_key) cx = drawBadge(rowTop, midY, cx, 'FK', theme.fkFill, theme.fkText);
        if (col.is_unique && !col.is_primary_key) cx = drawBadge(rowTop, midY, cx, 'UQ', theme.uqFill, theme.uqText);
        if (col.is_auto_increment) cx = drawBadge(rowTop, midY, cx, 'AI', theme.aiFill, theme.aiText);
        const colFont = 'Consolas, monospace';
        parts.push(`<text x="${cx}" y="${midY}" fill="${theme.text}" font-family="${colFont}" font-size="11">${escXml(col.name)}</text>`);
        const typeLabel = formatDataTypeDisplay(col.data_type);
        parts.push(`<text x="${tx + w - 12}" y="${midY}" fill="${theme.textMuted}" font-family="${colFont}" font-size="10" text-anchor="end">${escXml(typeLabel)}</text>`);
      });

      if (idxData?.rows?.length) {
        const sepY = ty + idxData.sectionY;
        parts.push(`<line x1="${tx + 6}" y1="${sepY}" x2="${tx + w - 6}" y2="${sepY}" stroke="${theme.headerLine}" stroke-width="1" stroke-dasharray="4 3"/>`);
        parts.push(`<text x="${tx + 10}" y="${sepY + idxData.headH / 2 + 3}" fill="${theme.textMuted}" font-family="${badgeFont}" font-size="9" font-weight="600" letter-spacing="0.06em">INDEXES</text>`);

        idxData.rows.forEach(({ index, rowY, rowH }) => {
          if (!index) return;
          const rowTop = ty + rowY;
          const midY = rowTop + rowH / 2 + 4;
          const kind = index.is_unique ? 'UQ' : 'IDX';
          const badgeFill = index.is_unique ? theme.idxUqFill : theme.idxFill;
          const badgeColor = index.is_unique ? theme.idxUqText : theme.idxText;
          const cols = indexColumnNames(index, table.columns || []);
          const label = `${index.name} (${cols.join(', ') || '—'})`;
          let cx = tx + 10;
          parts.push(`<rect x="${cx}" y="${rowTop + 3}" width="24" height="13" rx="2" fill="${badgeFill}" stroke="${theme.tableStroke}" stroke-width="0.6"/>`);
          parts.push(`<text x="${cx + 3}" y="${midY}" fill="${badgeColor}" font-family="${badgeFont}" font-size="7.5" font-weight="700">${kind}</text>`);
          cx += 30;
          const idxColor = styleKey === 'minimal' ? theme.text : theme.textMuted;
          parts.push(`<text x="${cx}" y="${midY}" fill="${idxColor}" font-family="Consolas, monospace" font-size="10">${escXml(label)}</text>`);
        });
      }

      parts.push('</g>');
    });

    relations.forEach((rel) => {
      const anchors = exportRelationAnchors(rel.from_column_id, rel.to_column_id);
      if (!anchors) return;
      parts.push(relationPathSvg(
        anchors.a.x, anchors.a.y, anchors.b.x, anchors.b.y,
        theme.relStroke, theme.relWidth, theme.arrowFill,
      ));
    });

    parts.push('</svg>');
    return { svg: parts.join(''), height: totalH, width: layoutWidth, theme };
  }

  function svgToPng(svgString, width, height, scale) {
    return new Promise((resolve, reject) => {
      const img = new Image();
      const url = URL.createObjectURL(new Blob([svgString], { type: 'image/svg+xml;charset=utf-8' }));
      img.onload = () => {
        const s = scale || 2;
        const canvas = document.createElement('canvas');
        canvas.width = Math.ceil(width * s);
        canvas.height = Math.ceil(height * s);
        const ctx = canvas.getContext('2d');
        if (s > 1) ctx.scale(s, s);
        ctx.drawImage(img, 0, 0);
        URL.revokeObjectURL(url);
        canvas.toBlob((blob) => {
          if (blob) resolve(blob);
          else reject(new Error('Failed to create PNG'));
        }, 'image/png');
      };
      img.onerror = () => {
        URL.revokeObjectURL(url);
        reject(new Error('Failed to render diagram'));
      };
      img.src = url;
    });
  }

  async function ensureDiagramExportReady() {
    if (!state.activeProjectId || !state.schema?.tables?.length) {
      throw new Error('No tables to export');
    }
    if (!$('#page-designer')?.classList.contains('active')) {
      showPage('designer');
      await new Promise((r) => requestAnimationFrame(() => requestAnimationFrame(r)));
    } else if (!$('#canvas')?.querySelector('.table-node')) {
      renderCanvas();
      await new Promise((r) => requestAnimationFrame(r));
    }
  }

  async function buildDiagramExportBundle(styleKey, format) {
    await ensureDiagramExportReady();
    const layout = measureExportDiagram(styleKey);
    if (!layout) throw new Error('Could not measure diagram');
    const { svg, height, width, theme } = buildDiagramSvg(layout, styleKey);
    const base = slugName(state.schema.project?.name);
    const styleSuffix = styleKey === 'minimal' ? '-minimal' : '';
    const ext = format === 'svg' ? 'svg' : 'png';
    const filename = `${base}-diagram${styleSuffix}.${ext}`;
    const styleLabel = styleKey === 'minimal' ? 'Minimal (print)' : 'Dark';

    if (format === 'svg') {
      return {
        format,
        styleKey,
        styleLabel,
        filename,
        svg,
        width,
        height,
        theme,
        blob: new Blob([svg], { type: 'image/svg+xml;charset=utf-8' }),
      };
    }

    const png = await svgToPng(svg, width, height, theme.pngScale);
    return {
      format,
      styleKey,
      styleLabel,
      filename,
      svg,
      width,
      height,
      theme,
      blob: png,
    };
  }

  function clearExportPreview() {
    if (state.exportPreview?.previewUrl) {
      URL.revokeObjectURL(state.exportPreview.previewUrl);
    }
    state.exportPreview = null;
    const box = $('#exportPreviewContent');
    if (box) box.innerHTML = '';
  }

  function closeExportPreviewModal() {
    clearExportPreview();
    $('#exportPreviewModal')?.close();
  }

  function renderExportPreview(bundle) {
    const meta = $('#exportPreviewMeta');
    if (meta) {
      meta.textContent = `${bundle.format.toUpperCase()} · ${bundle.styleLabel} · ${Math.round(bundle.width)} × ${Math.round(bundle.height)} px`;
    }
    const box = $('#exportPreviewContent');
    if (!box) return;
    box.innerHTML = '';

    if (bundle.format === 'svg') {
      const wrap = document.createElement('div');
      wrap.innerHTML = bundle.svg;
      const svgEl = wrap.querySelector('svg');
      if (svgEl) {
        svgEl.removeAttribute('width');
        svgEl.removeAttribute('height');
        svgEl.style.maxWidth = '100%';
        svgEl.style.height = 'auto';
        box.appendChild(svgEl);
      }
      return;
    }

    const url = URL.createObjectURL(bundle.blob);
    bundle.previewUrl = url;
    const img = document.createElement('img');
    img.src = url;
    img.alt = 'Diagram export preview';
    box.appendChild(img);
  }

  async function openExportPreview() {
    clearExportPreview();
    const format = $('#exportImageFormat')?.value || 'png';
    const styleKey = $('#exportImageStyle')?.value || 'dark';
    const modal = $('#exportPreviewModal');
    const meta = $('#exportPreviewMeta');
    const box = $('#exportPreviewContent');

    try {
      if (meta) meta.textContent = 'Generating preview…';
      if (box) box.innerHTML = '<p class="export-preview-loading">Rendering diagram…</p>';
      modal?.showModal();

      const bundle = await buildDiagramExportBundle(styleKey, format);
      state.exportPreview = bundle;
      renderExportPreview(bundle);
    } catch (err) {
      closeExportPreviewModal();
      toast(err.message, 'error');
    }
  }

  function confirmExportDiagramDownload() {
    const bundle = state.exportPreview;
    if (!bundle?.blob) return;
    downloadBlob(bundle.blob, bundle.filename);
    toast(bundle.styleKey === 'minimal' ? 'Minimal diagram exported' : 'Diagram exported');
    closeExportPreviewModal();
  }

  $('#btnExportDiagram')?.addEventListener('click', () => openExportPreview());
  $$('[data-close-export-preview]').forEach((btn) => btn.addEventListener('click', closeExportPreviewModal));
  $('#btnConfirmExportDiagram')?.addEventListener('click', confirmExportDiagramDownload);
  $('#exportPreviewModal')?.addEventListener('close', clearExportPreview);

  // --- Column type editor ---
  const COLUMN_TYPES = [
    { id: 'int', label: 'INT' },
    { id: 'bigint', label: 'BIGINT' },
    { id: 'smallint', label: 'SMALLINT' },
    { id: 'tinyint', label: 'TINYINT' },
    { id: 'varchar', label: 'VARCHAR', hasLength: true, lengthDefault: '255', lengthPlaceholder: '255' },
    { id: 'char', label: 'CHAR', hasLength: true, lengthDefault: '1', lengthPlaceholder: '1' },
    { id: 'text', label: 'TEXT' },
    { id: 'mediumtext', label: 'MEDIUMTEXT' },
    { id: 'longtext', label: 'LONGTEXT' },
    { id: 'decimal', label: 'DECIMAL', hasPrecision: true, lengthDefault: '15,2', lengthPlaceholder: '15,2' },
    { id: 'float', label: 'FLOAT' },
    { id: 'double', label: 'DOUBLE' },
    { id: 'boolean', label: 'BOOLEAN' },
    { id: 'date', label: 'DATE' },
    { id: 'datetime', label: 'DATETIME' },
    { id: 'timestamp', label: 'TIMESTAMP' },
    { id: 'time', label: 'TIME' },
    { id: 'json', label: 'JSON' },
    { id: 'blob', label: 'BLOB' },
    { id: 'uuid', label: 'UUID' },
    { id: 'enum', label: 'ENUM', hasEnum: true },
    { id: 'set', label: 'SET', hasEnum: true },
    { id: 'custom', label: 'Custom…', isCustom: true },
  ];

  function columnTypeDef(id) {
    return COLUMN_TYPES.find((t) => t.id === id);
  }

  function parseEnumValuesList(raw) {
    return String(raw || '')
      .split(',')
      .map((v) => v.trim().replace(/^['"]|['"]$/g, ''))
      .filter(Boolean);
  }

  function parseColumnType(dataType) {
    const raw = String(dataType || '').replace(/\s+/g, ' ').trim();
    if (!raw) return { kind: 'varchar', length: '255', enumValues: [], custom: '' };

    const enumMatch = raw.match(/^ENUM\s*\((.*)\)/i);
    if (enumMatch) {
      return { kind: 'enum', length: '', enumValues: parseEnumValuesList(enumMatch[1]), custom: '' };
    }
    const setMatch = raw.match(/^SET\s*\((.*)\)/i);
    if (setMatch) {
      return { kind: 'set', length: '', enumValues: parseEnumValuesList(setMatch[1]), custom: '' };
    }

    const withParen = raw.match(/^([A-Za-z_]+)\s*\(\s*([^)]+)\s*\)$/i);
    if (withParen) {
      const base = withParen[1].toLowerCase();
      const len = withParen[2].trim();
      const def = COLUMN_TYPES.find((t) => t.id === base || t.label.toLowerCase() === base);
      if (def && !def.isCustom) {
        return { kind: def.id, length: len, enumValues: [], custom: '' };
      }
    }

    const baseOnly = raw.match(/^([A-Za-z_]+)/);
    if (baseOnly) {
      const base = baseOnly[1].toLowerCase();
      const def = COLUMN_TYPES.find((t) => t.id === base || t.label.toLowerCase() === base);
      if (def && !def.isCustom) {
        return {
          kind: def.id,
          length: def.lengthDefault || '',
          enumValues: [],
          custom: '',
        };
      }
    }

    return { kind: 'custom', length: '', enumValues: [], custom: raw };
  }

  function buildColumnType(parsed) {
    const def = columnTypeDef(parsed.kind);
    if (!def || def.isCustom) return (parsed.custom || 'TEXT').trim() || 'TEXT';
    if (def.hasEnum) {
      const values = (parsed.enumValues || []).map((v) => String(v).trim()).filter(Boolean);
      if (!values.length) return def.label;
      const quoted = values.map((v) => `'${v.replace(/'/g, "''")}'`).join(', ');
      return `${def.label}(${quoted})`;
    }
    if ((def.hasLength || def.hasPrecision) && parsed.length) {
      return `${def.label}(${String(parsed.length).trim()})`;
    }
    return def.label;
  }

  function typeSelectOptions(selectedKind) {
    return COLUMN_TYPES.map((t) =>
      `<option value="${t.id}" ${t.id === selectedKind ? 'selected' : ''}>${esc(t.label)}</option>`,
    ).join('');
  }

  function formatEnumSummary(values) {
    const v = (values || []).map((x) => String(x).trim()).filter(Boolean);
    if (!v.length) return 'No values yet';
    if (v.length <= 3) return `${v.length} value${v.length > 1 ? 's' : ''}: ${v.join(', ')}`;
    return `${v.length} values: ${v.slice(0, 2).join(', ')}…`;
  }

  const SQL_DEFAULT_KEYWORDS = new Set([
    'CURRENT_TIMESTAMP', 'CURRENT_DATE', 'CURRENT_TIME', 'NOW()', 'NULL', 'TRUE', 'FALSE',
  ]);

  function defaultValueCategory(kind) {
    if (kind === 'blob') return 'none';
    if (kind === 'enum') return 'enum';
    if (kind === 'set') return 'set';
    if (['int', 'bigint', 'smallint', 'tinyint'].includes(kind)) return 'integer';
    if (['decimal', 'float', 'double'].includes(kind)) return 'number';
    if (kind === 'boolean') return 'boolean';
    if (['varchar', 'char', 'text', 'mediumtext', 'longtext'].includes(kind)) return 'string';
    if (kind === 'date') return 'date';
    if (['datetime', 'timestamp'].includes(kind)) return 'datetime';
    if (kind === 'time') return 'time';
    if (kind === 'json') return 'json';
    if (kind === 'uuid') return 'uuid';
    if (kind === 'custom') return 'raw';
    return 'string';
  }

  function defaultPlaceholder(kind, enumValues = []) {
    const cat = defaultValueCategory(kind);
    if (cat === 'none') return '—';
    if (cat === 'integer') return '0';
    if (cat === 'number') return '0.00';
    if (cat === 'boolean') return '0 or 1';
    if (cat === 'date') return '2024-01-01';
    if (cat === 'datetime') return 'CURRENT_TIMESTAMP';
    if (cat === 'time') return '12:00:00';
    if (cat === 'json') return '{"key":"value"}';
    if (cat === 'uuid') return 'uuid';
    if (cat === 'enum') return enumValues[0] || 'enum value';
    if (cat === 'set') return enumValues.slice(0, 2).join(',') || 'a,b';
    if (cat === 'raw') return 'SQL expression';
    return 'optional';
  }

  function defaultDisplayValue(stored) {
    if (!stored || !String(stored).trim()) return '';
    const s = String(stored).trim();
    if (s.toUpperCase() === 'NULL') return 'NULL';
    if (SQL_DEFAULT_KEYWORDS.has(s.toUpperCase()) || s.toUpperCase().startsWith('CURRENT_')) return s;
    if ((s.startsWith("'") && s.endsWith("'")) || (s.startsWith('"') && s.endsWith('"'))) {
      const q = s[0];
      return s.slice(1, -1).replace(new RegExp(q + q, 'g'), q);
    }
    return s;
  }

  function defaultControlMode(kind) {
    const cat = defaultValueCategory(kind);
    if (cat === 'enum') return 'select-enum';
    if (cat === 'set') return 'select-set';
    if (cat === 'boolean') return 'select-boolean';
    if (cat === 'datetime') return 'datetime-combo';
    if (kind === 'date') return 'date-combo';
    if (kind === 'time') return 'time-combo';
    if (cat === 'integer') return 'integer';
    if (cat === 'number') return 'number';
    if (cat === 'none') return 'none';
    return 'text';
  }

  function formatDatetimeLocal(val) {
    if (!val) return '';
    const [d, t = '00:00'] = val.split('T');
    const parts = t.split(':');
    const hh = parts[0] || '00';
    const mm = parts[1] || '00';
    const ss = parts[2] || '00';
    return `${d} ${hh}:${mm}:${ss}`;
  }

  function parseToDatetimeLocal(display) {
    const m = String(display || '').match(/^(\d{4}-\d{2}-\d{2}) (\d{2}:\d{2})(?::(\d{2}))?$/);
    if (!m) return '';
    return `${m[1]}T${m[2]}`;
  }

  function resolveDefaultForUI(kind, storedRaw, enumValues) {
    const display = String(storedRaw || '').trim();
    if (!display) return { value: '', literal: '', selected: [] };
    const upper = display.toUpperCase();
    const mode = defaultControlMode(kind);

    if (mode === 'select-boolean') {
      if (upper === 'TRUE' || upper === 'FALSE') return { value: upper, literal: '', selected: [] };
      if (display === '1' || upper === 'TRUE') return { value: '1', literal: '', selected: [] };
      if (display === '0' || upper === 'FALSE') return { value: '0', literal: '', selected: [] };
      return { value: display, literal: '', selected: [] };
    }
    if (mode === 'select-enum') {
      const match = enumValues.find((v) => v === display || v.toLowerCase() === display.toLowerCase());
      return { value: match || '', literal: '', selected: [] };
    }
    if (mode === 'select-set') {
      const parts = display.split(',').map((s) => s.trim()).filter(Boolean);
      const selected = parts.map((p) => enumValues.find((v) => v === p || v.toLowerCase() === p.toLowerCase()) || p);
      return { value: '', literal: '', selected: selected.filter(Boolean) };
    }
    if (mode === 'datetime-combo') {
      if (upper === 'CURRENT_TIMESTAMP' || upper === 'NOW()') return { value: 'CURRENT_TIMESTAMP', literal: '', selected: [] };
      if (upper === 'NULL') return { value: 'NULL', literal: '', selected: [] };
      return { value: '__literal__', literal: parseToDatetimeLocal(display), selected: [] };
    }
    if (mode === 'date-combo') {
      if (upper === 'CURRENT_DATE') return { value: 'CURRENT_DATE', literal: '', selected: [] };
      if (upper === 'NULL') return { value: 'NULL', literal: '', selected: [] };
      return { value: '__literal__', literal: display, selected: [] };
    }
    if (mode === 'time-combo') {
      if (upper === 'CURRENT_TIME') return { value: 'CURRENT_TIME', literal: '', selected: [] };
      if (upper === 'NULL') return { value: 'NULL', literal: '', selected: [] };
      const lit = display.length === 8 ? display : (display.length === 5 ? `${display}:00` : display);
      return { value: '__literal__', literal: lit.slice(0, 8), selected: [] };
    }
    return { value: display, literal: '', selected: [] };
  }

  function readDefaultFromRow(row) {
    const wrap = row.querySelector('.c-default-wrap');
    if (!wrap || wrap.classList.contains('is-inactive')) return '';
    const mode = wrap.dataset.mode || defaultControlMode(row.querySelector('.c-kind')?.value || 'varchar');

    if (mode === 'datetime-combo' || mode === 'date-combo' || mode === 'time-combo') {
      const sel = wrap.querySelector('.c-default-select')?.value || '';
      if (!sel) return '';
      if (sel === '__literal__') {
        const lit = wrap.querySelector('.c-default-literal')?.value || '';
        if (!lit) return '';
        if (mode === 'datetime-combo') return formatDatetimeLocal(lit);
        if (mode === 'time-combo') return lit.length === 5 ? `${lit}:00` : lit;
        return lit;
      }
      return sel;
    }
    if (mode === 'select-set') {
      return [...(wrap.querySelector('.c-default-set')?.selectedOptions || [])]
        .map((o) => o.value)
        .filter(Boolean)
        .join(',');
    }
    return wrap.querySelector('.c-default')?.value.trim() || '';
  }

  function bindIntegerDefaultInput(input) {
    input.addEventListener('keydown', (e) => {
      if (e.ctrlKey || e.metaKey || e.altKey) return;
      if (['Backspace', 'Delete', 'Tab', 'ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(e.key)) return;
      if (e.key === '-' && input.selectionStart === 0 && !input.value.includes('-')) return;
      if (/^\d$/.test(e.key)) return;
      e.preventDefault();
    });
    input.addEventListener('paste', (e) => {
      e.preventDefault();
      const text = (e.clipboardData.getData('text') || '').trim();
      if (/^-?\d+$/.test(text)) input.value = text;
    });
    input.addEventListener('input', () => {
      const cleaned = input.value.replace(/[^\d-]/g, '');
      const neg = cleaned.startsWith('-') ? '-' : '';
      const digits = cleaned.replace(/-/g, '');
      input.value = neg + digits;
    });
  }

  function bindNumberDefaultInput(input) {
    input.addEventListener('keydown', (e) => {
      if (e.ctrlKey || e.metaKey || e.altKey) return;
      if (['Backspace', 'Delete', 'Tab', 'ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(e.key)) return;
      if (e.key === '-' && input.selectionStart === 0 && !input.value.includes('-')) return;
      if (e.key === '.' && !input.value.includes('.')) return;
      if (/^\d$/.test(e.key)) return;
      e.preventDefault();
    });
    input.addEventListener('paste', (e) => {
      e.preventDefault();
      const text = (e.clipboardData.getData('text') || '').trim();
      if (/^-?\d+(\.\d+)?$/.test(text)) input.value = text;
    });
    input.addEventListener('input', () => {
      let v = input.value.replace(/[^\d.-]/g, '');
      const neg = v.startsWith('-') ? '-' : '';
      v = v.replace(/-/g, '');
      const dot = v.indexOf('.');
      if (dot >= 0) v = v.slice(0, dot + 1) + v.slice(dot + 1).replace(/\./g, '');
      input.value = neg + v;
    });
  }

  function syncComboLiteralVisibility(wrap) {
    const sel = wrap.querySelector('.c-default-select');
    const lit = wrap.querySelector('.c-default-literal');
    if (!sel || !lit) return;
    const show = sel.value === '__literal__';
    lit.hidden = !show;
    lit.disabled = !show;
    if (!show) lit.value = '';
  }

  function markDefaultInvalid(row, invalid) {
    row.querySelector('.c-default-wrap')?.classList.toggle('is-invalid', !!invalid);
  }

  function bindDefaultControlEvents(wrap, row) {
    const mode = wrap.dataset.mode;
    const defEl = wrap.querySelector('.c-default');
    if (mode === 'integer' && defEl) bindIntegerDefaultInput(defEl);
    if (mode === 'number' && defEl) bindNumberDefaultInput(defEl);

    if (mode === 'datetime-combo' || mode === 'date-combo' || mode === 'time-combo') {
      const sel = wrap.querySelector('.c-default-select');
      sel?.addEventListener('change', () => syncComboLiteralVisibility(wrap));
      syncComboLiteralVisibility(wrap);
    }

    const validate = () => {
      const result = validateAndFormatDefault(row);
      markDefaultInvalid(row, !result.ok && !!readDefaultFromRow(row));
    };
    wrap.querySelectorAll('.c-default, .c-default-select, .c-default-literal, .c-default-set')
      .forEach((el) => {
        el.addEventListener('change', validate);
        el.addEventListener('blur', validate);
      });
  }

  function renderDefaultControl(row) {
    const wrap = row.querySelector('.c-default-wrap');
    if (!wrap) return;

    const kind = row.querySelector('.c-kind')?.value || 'varchar';
    const def = columnTypeDef(kind);
    const enumValues = getEnumValuesFromRow(row);
    const isPK = row.querySelector('.c-pk')?.checked;
    const cat = defaultValueCategory(kind);
    const noDefault = isPK || cat === 'none';
    const mode = defaultControlMode(kind);

    const currentStored = wrap.dataset.mode
      ? readDefaultFromRow(row)
      : (wrap.dataset.defaultValue || '');
    if (wrap.dataset.defaultValue !== undefined) delete wrap.dataset.defaultValue;
    const resolved = resolveDefaultForUI(kind, currentStored, enumValues);

    wrap.dataset.mode = mode;
    wrap.classList.remove('is-combo', 'is-invalid');
    wrap.classList.toggle('is-inactive', noDefault);

    if (noDefault) {
      wrap.innerHTML = '<select class="c-default" disabled title="No default"><option value="">—</option></select>';
      return;
    }

    if (mode === 'select-boolean') {
      wrap.innerHTML = `
        <select class="c-default" title="Default value">
          <option value="">None</option>
          <option value="0">0 (false)</option>
          <option value="1">1 (true)</option>
          <option value="TRUE">TRUE</option>
          <option value="FALSE">FALSE</option>
        </select>`;
      wrap.querySelector('.c-default').value = resolved.value || '';
    } else if (mode === 'select-enum') {
      const opts = enumValues.length
        ? enumValues.map((v) => `<option value="${esc(v)}">${esc(v)}</option>`).join('')
        : '<option value="" disabled>No ENUM values yet</option>';
      wrap.innerHTML = `
        <select class="c-default" title="Default value" ${enumValues.length ? '' : 'disabled'}>
          <option value="">None</option>
          ${opts}
        </select>`;
      if (enumValues.length) wrap.querySelector('.c-default').value = resolved.value || '';
    } else if (mode === 'select-set') {
      const opts = enumValues.length
        ? enumValues.map((v) => `<option value="${esc(v)}">${esc(v)}</option>`).join('')
        : '<option value="" disabled>No SET values yet</option>';
      wrap.innerHTML = `
        <select class="c-default-set" multiple size="${Math.min(4, Math.max(2, enumValues.length || 2))}"
          title="Default SET values (Ctrl+click for multiple)" ${enumValues.length ? '' : 'disabled'}>
          ${opts}
        </select>`;
      if (enumValues.length && resolved.selected.length) {
        const sel = wrap.querySelector('.c-default-set');
        resolved.selected.forEach((v) => {
          [...sel.options].forEach((o) => { if (o.value === v) o.selected = true; });
        });
      }
    } else if (mode === 'datetime-combo') {
      wrap.classList.add('is-combo');
      wrap.innerHTML = `
        <select class="c-default-select" title="Default value">
          <option value="">None</option>
          <option value="CURRENT_TIMESTAMP">CURRENT_TIMESTAMP</option>
          <option value="NULL">NULL</option>
          <option value="__literal__">Custom date/time…</option>
        </select>
        <input type="datetime-local" class="c-default-literal" step="1" hidden>`;
      wrap.querySelector('.c-default-select').value = resolved.value || '';
      if (resolved.literal) wrap.querySelector('.c-default-literal').value = resolved.literal;
    } else if (mode === 'date-combo') {
      wrap.classList.add('is-combo');
      wrap.innerHTML = `
        <select class="c-default-select" title="Default value">
          <option value="">None</option>
          <option value="CURRENT_DATE">CURRENT_DATE</option>
          <option value="NULL">NULL</option>
          <option value="__literal__">Custom date…</option>
        </select>
        <input type="date" class="c-default-literal" hidden>`;
      wrap.querySelector('.c-default-select').value = resolved.value || '';
      if (resolved.literal) wrap.querySelector('.c-default-literal').value = resolved.literal;
    } else if (mode === 'time-combo') {
      wrap.classList.add('is-combo');
      wrap.innerHTML = `
        <select class="c-default-select" title="Default value">
          <option value="">None</option>
          <option value="CURRENT_TIME">CURRENT_TIME</option>
          <option value="NULL">NULL</option>
          <option value="__literal__">Custom time…</option>
        </select>
        <input type="time" class="c-default-literal" step="1" hidden>`;
      wrap.querySelector('.c-default-select').value = resolved.value || '';
      if (resolved.literal) {
        const lit = wrap.querySelector('.c-default-literal');
        lit.value = resolved.literal.slice(0, 5);
      }
    } else if (mode === 'integer') {
      wrap.innerHTML = `<input type="text" class="c-default" inputmode="numeric" autocomplete="off"
        value="${esc(resolved.value)}" placeholder="0" title="Integer default">`;
    } else if (mode === 'number') {
      wrap.innerHTML = `<input type="text" class="c-default" inputmode="decimal" autocomplete="off"
        value="${esc(resolved.value)}" placeholder="0.00" title="Numeric default">`;
    } else {
      wrap.innerHTML = `<input type="text" class="c-default" value="${esc(resolved.value)}"
        placeholder="${esc(defaultPlaceholder(kind, enumValues))}" title="Default for ${esc(def?.label || kind)}">`;
    }

    bindDefaultControlEvents(wrap, row);
  }

  function escapeSqlString(s) {
    return String(s).replace(/'/g, "''");
  }

  function validateAndFormatDefault(row) {
    const kind = row.querySelector('.c-kind')?.value || 'varchar';
    const isPK = row.querySelector('.c-pk')?.checked;
    const input = readDefaultFromRow(row);
    const enumValues = getEnumValuesFromRow(row);
    const colName = row.querySelector('.c-name')?.value.trim() || 'column';
    const cat = defaultValueCategory(kind);

    if (isPK || cat === 'none') {
      if (input) return { ok: false, message: `${colName}: default is not allowed for this column` };
      return { ok: true, sql: '' };
    }
    if (!input) return { ok: true, sql: '' };

    const upper = input.toUpperCase();
    if (upper === 'NULL') return { ok: true, sql: 'NULL' };
    if (['datetime', 'timestamp'].includes(kind) && (upper === 'CURRENT_TIMESTAMP' || upper === 'NOW()')) {
      return { ok: true, sql: 'CURRENT_TIMESTAMP' };
    }
    if (kind === 'date' && upper === 'CURRENT_DATE') return { ok: true, sql: 'CURRENT_DATE' };
    if (kind === 'time' && upper === 'CURRENT_TIME') return { ok: true, sql: 'CURRENT_TIME' };
    if (cat === 'boolean' && (upper === 'TRUE' || upper === 'FALSE')) {
      return { ok: true, sql: upper };
    }
    if (SQL_DEFAULT_KEYWORDS.has(upper) || upper.startsWith('CURRENT_')) {
      return { ok: false, message: `${colName}: ${input} is not valid for ${kind.toUpperCase()}` };
    }

    switch (cat) {
      case 'integer':
        if (!/^-?\d+$/.test(input)) return { ok: false, message: `${colName}: default must be an integer` };
        return { ok: true, sql: input };
      case 'number':
        if (!/^-?\d+(\.\d+)?$/.test(input)) return { ok: false, message: `${colName}: default must be a number` };
        return { ok: true, sql: input };
      case 'boolean':
        if (/^(0|1)$/.test(input)) return { ok: true, sql: input };
        if (/^true$/i.test(input)) return { ok: true, sql: '1' };
        if (/^false$/i.test(input)) return { ok: true, sql: '0' };
        return { ok: false, message: `${colName}: use 0, 1, true, or false` };
      case 'date':
        if (!/^\d{4}-\d{2}-\d{2}$/.test(input)) return { ok: false, message: `${colName}: use YYYY-MM-DD` };
        return { ok: true, sql: `'${escapeSqlString(input)}'` };
      case 'datetime':
        if (!/^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$/.test(input)) {
          return { ok: false, message: `${colName}: use YYYY-MM-DD HH:MM:SS or CURRENT_TIMESTAMP` };
        }
        return { ok: true, sql: `'${escapeSqlString(input)}'` };
      case 'time':
        if (!/^\d{2}:\d{2}:\d{2}$/.test(input)) return { ok: false, message: `${colName}: use HH:MM:SS` };
        return { ok: true, sql: `'${escapeSqlString(input)}'` };
      case 'uuid':
        if (!/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(input)) {
          return { ok: false, message: `${colName}: invalid UUID format` };
        }
        return { ok: true, sql: `'${escapeSqlString(input)}'` };
      case 'json':
        try {
          JSON.parse(input);
        } catch (_) {
          return { ok: false, message: `${colName}: default must be valid JSON` };
        }
        return { ok: true, sql: `'${escapeSqlString(input)}'` };
      case 'enum': {
        if (!enumValues.length) return { ok: false, message: `${colName}: define ENUM values first` };
        const match = enumValues.find((v) => v === input || v.toLowerCase() === input.toLowerCase());
        if (!match) return { ok: false, message: `${colName}: default must be one of: ${enumValues.join(', ')}` };
        return { ok: true, sql: `'${escapeSqlString(match)}'` };
      }
      case 'set': {
        if (!enumValues.length) return { ok: false, message: `${colName}: define SET values first` };
        const parts = input.split(',').map((p) => p.trim()).filter(Boolean);
        if (!parts.length) return { ok: false, message: `${colName}: enter comma-separated SET values` };
        const canonical = [];
        for (const p of parts) {
          const match = enumValues.find((v) => v === p || v.toLowerCase() === p.toLowerCase());
          if (!match) return { ok: false, message: `${colName}: "${p}" is not in SET values` };
          if (!canonical.includes(match)) canonical.push(match);
        }
        return { ok: true, sql: `'${escapeSqlString(canonical.join(','))}'` };
      }
      case 'string':
        return { ok: true, sql: `'${escapeSqlString(input)}'` };
      case 'raw':
        if (/[;]/.test(input)) return { ok: false, message: `${colName}: invalid default expression` };
        return { ok: true, sql: input };
      default:
        return { ok: true, sql: `'${escapeSqlString(input)}'` };
    }
  }

  function getEnumValuesFromRow(row) {
    const jsonEl = row.querySelector('.c-enum-json');
    if (jsonEl?.value) {
      try {
        const parsed = JSON.parse(jsonEl.value);
        if (Array.isArray(parsed)) return parsed.map((x) => String(x).trim()).filter(Boolean);
      } catch (_) { /* fall through */ }
    }
    return [];
  }

  function setEnumValuesOnRow(row, values) {
    const v = (values || []).map((x) => String(x).trim()).filter(Boolean);
    const store = row.querySelector('.c-enum-store');
    if (store) {
      store.innerHTML = `<input type="hidden" class="c-enum-json" value="${esc(JSON.stringify(v))}">`;
    }
    const summary = row.querySelector('.c-enum-summary');
    if (summary) summary.textContent = formatEnumSummary(v);
  }

  function parseBulkEnumValues(text) {
    return String(text || '')
      .split(/[\n,]+/)
      .map((s) => s.trim().replace(/^['"]|['"]$/g, ''))
      .filter(Boolean);
  }

  function renderEnumModalList(values) {
    const list = $('#enumModalList');
    if (!list) return;
    const rows = (values.length ? values : ['']).map((v, i) => `
      <div class="enum-modal-item" data-idx="${i}">
        <input type="text" class="enum-modal-val" value="${esc(v)}" placeholder="value">
        <button type="button" class="btn icon enum-modal-rm" title="Remove">✕</button>
      </div>`).join('');
    list.innerHTML = rows;
    list.querySelectorAll('.enum-modal-rm').forEach((btn) => {
      btn.addEventListener('click', () => {
        const item = btn.closest('.enum-modal-item');
        const items = list.querySelectorAll('.enum-modal-item');
        if (items.length <= 1) {
          item.querySelector('.enum-modal-val').value = '';
          return;
        }
        item?.remove();
      });
    });
  }

  function readEnumModalValues() {
    return [...$$('#enumModalList .enum-modal-val')]
      .map((el) => el.value.trim())
      .filter(Boolean);
  }

  function openEnumModal(row) {
    const kind = row.querySelector('.c-kind')?.value || 'enum';
    const colName = row.querySelector('.c-name')?.value.trim() || 'column';
    const typeLabel = kind === 'set' ? 'SET' : 'ENUM';
    state.enumEditRow = row;
    $('#enumModalTitle').textContent = `${typeLabel} values — ${colName}`;
    $('#enumModalSubtitle').textContent = `Define allowed values for ${typeLabel} column "${colName}".`;
    $('#enumBulkPaste').value = '';
    renderEnumModalList(getEnumValuesFromRow(row));
    $('#enumModal').showModal();
    $('#enumModalList .enum-modal-val')?.focus();
  }

  function closeEnumModal() {
    state.enumEditRow = null;
    $('#enumModal')?.close();
  }

  $('#btnEnumAddRow')?.addEventListener('click', () => {
    const list = $('#enumModalList');
    if (!list) return;
    const item = document.createElement('div');
    item.className = 'enum-modal-item';
    item.innerHTML = `
      <input type="text" class="enum-modal-val" placeholder="value">
      <button type="button" class="btn icon enum-modal-rm" title="Remove">✕</button>`;
    list.appendChild(item);
    item.querySelector('.enum-modal-rm')?.addEventListener('click', () => {
      if (list.querySelectorAll('.enum-modal-item').length <= 1) {
        item.querySelector('.enum-modal-val').value = '';
        return;
      }
      item.remove();
    });
    item.querySelector('.enum-modal-val')?.focus();
  });

  $('#btnEnumApplyBulk')?.addEventListener('click', () => {
    const values = parseBulkEnumValues($('#enumBulkPaste')?.value);
    if (!values.length) return toast('Nothing to paste', 'error');
    renderEnumModalList(values);
    toast(`${values.length} value(s) loaded`);
  });

  $('#btnSaveEnum')?.addEventListener('click', () => {
    if (!state.enumEditRow) return;
    const values = readEnumModalValues();
    if (!values.length) return toast('Add at least one value', 'error');
    setEnumValuesOnRow(state.enumEditRow, values);
    updateColumnTypeUI(state.enumEditRow);
    closeEnumModal();
    toast('Values updated');
  });

  $$('#enumModal [data-close]').forEach((el) => {
    el.addEventListener('click', () => closeEnumModal());
  });

  function readColumnTypeFromRow(row) {
    const kind = row.querySelector('.c-kind')?.value || 'varchar';
    const length = row.querySelector('.c-length')?.value.trim() || '';
    const custom = row.querySelector('.c-custom-type')?.value.trim() || '';
    const enumValues = getEnumValuesFromRow(row);
    return buildColumnType({ kind, length, enumValues, custom });
  }

  function updateColumnTypeUI(row) {
    const kind = row.querySelector('.c-kind')?.value || 'varchar';
    const def = columnTypeDef(kind);
    const lenEl = row.querySelector('.c-length');
    const customEl = row.querySelector('.c-custom-type');
    const customRow = row.querySelector('.c-custom-row');
    const enumBar = row.querySelector('.c-enum-bar');
    if (!lenEl || !customEl || !enumBar || !row.querySelector('.c-default-wrap')) return;

    const needsLen = !!(def?.hasLength || def?.hasPrecision);
    lenEl.disabled = !needsLen;
    lenEl.classList.toggle('is-inactive', !needsLen);
    lenEl.placeholder = def?.hasPrecision ? '15,2' : (def?.lengthPlaceholder || '255');
    if (needsLen && !lenEl.value && def?.lengthDefault) lenEl.value = def.lengthDefault;

    const isCustom = kind === 'custom';
    customEl.hidden = !isCustom;
    customRow?.classList.toggle('visible', isCustom);
    enumBar.hidden = !def?.hasEnum;

    if (!def?.hasEnum && state.enumEditRow === row) closeEnumModal();

    if (def?.hasEnum && !getEnumValuesFromRow(row).length) {
      setEnumValuesOnRow(row, []);
    }

    renderDefaultControl(row);
    updateAutoIncrementUI(row);
  }

  function rowSupportsAutoIncrement(row) {
    const kind = row.querySelector('.c-kind')?.value || 'varchar';
    if (['int', 'bigint', 'smallint', 'tinyint'].includes(kind)) return true;
    if (kind === 'custom') {
      const dt = (row.querySelector('.c-custom-type')?.value || '').toUpperCase();
      return /\b(INT|INTEGER|BIGINT|SMALLINT|TINYINT|SERIAL|BIGSERIAL|SMALLSERIAL)\b/.test(dt);
    }
    return false;
  }

  function updateAutoIncrementUI(row) {
    const ai = row?.querySelector('.c-ai');
    if (!ai) return;
    const ok = rowSupportsAutoIncrement(row);
    ai.disabled = !ok;
    if (!ok) ai.checked = false;
  }

  function setColumnRowAutoIncrement(row, enabled) {
    const ai = row?.querySelector('.c-ai');
    if (!ai || ai.disabled) return;
    ai.checked = !!enabled;
  }

  function syncAutoIncrementSelection(activeRow) {
    const container = $('#columnEditor');
    if (!container || !activeRow?.querySelector('.c-ai')?.checked) return;
    container.querySelectorAll('.col-row').forEach((other) => {
      if (other === activeRow) return;
      setColumnRowAutoIncrement(other, false);
    });
  }

  function setColumnRowPrimaryKey(row, enabled) {
    const pk = row?.querySelector('.c-pk');
    if (!pk) return;
    pk.checked = !!enabled;
    const uq = row.querySelector('.c-unique');
    if (uq) {
      uq.disabled = !!enabled;
      if (enabled) uq.checked = false;
    }
    updateColumnTypeUI(row);
  }

  function syncPrimaryKeySelection(activeRow) {
    const container = $('#columnEditor');
    if (!container || !activeRow?.querySelector('.c-pk')?.checked) return;
    container.querySelectorAll('.col-row').forEach((other) => {
      if (other === activeRow) return;
      setColumnRowPrimaryKey(other, false);
    });
  }

  function bindColumnRowEvents(row) {
    row.querySelector('.c-kind')?.addEventListener('change', () => {
      updateColumnTypeUI(row);
      const kind = row.querySelector('.c-kind')?.value;
      if (kind === 'enum' || kind === 'set') {
        openEnumModal(row);
      }
    });
    row.querySelector('.c-custom-type')?.addEventListener('input', () => updateAutoIncrementUI(row));
    row.querySelector('.c-pk')?.addEventListener('change', () => {
      if (row.querySelector('.c-pk')?.checked) {
        syncPrimaryKeySelection(row);
      }
      setColumnRowPrimaryKey(row, !!row.querySelector('.c-pk')?.checked);
    });
    row.querySelector('.c-ai')?.addEventListener('change', () => {
      if (row.querySelector('.c-ai')?.checked) {
        syncAutoIncrementSelection(row);
      }
      setColumnRowAutoIncrement(row, !!row.querySelector('.c-ai')?.checked);
    });
    row.querySelector('.c-enum-manage')?.addEventListener('click', () => openEnumModal(row));
    updateColumnTypeUI(row);
  }

  function columnRowHTML(c) {
    const parsed = parseColumnType(c.data_type);
    const def = columnTypeDef(parsed.kind);
    const showEnum = def?.hasEnum;
    const showCustom = parsed.kind === 'custom';
    const needsLen = !!(def?.hasLength || def?.hasPrecision);
    const defaultVal = defaultDisplayValue(c.default_value);
    return `
      <div class="col-row" data-col-id="${c.id}">
        <div class="col-row-cells">
          <span class="c-drag-handle" draggable="true" title="Drag to reorder">⋮⋮</span>
          <input type="text" class="c-name" value="${esc(c.name)}" placeholder="Column name">
          <select class="c-kind">${typeSelectOptions(parsed.kind)}</select>
          <input type="text" class="c-length${needsLen ? '' : ' is-inactive'}" value="${esc(parsed.length)}" placeholder="255" title="Length / precision">
          <div class="c-default-wrap" data-default-value="${esc(defaultVal)}"></div>
          <div class="col-flags">
            <label title="Primary key (one per table)"><input type="checkbox" class="c-pk" ${c.is_primary_key ? 'checked' : ''}> PK</label>
            <label title="Auto increment (one per table, integer types)"><input type="checkbox" class="c-ai" ${c.is_auto_increment ? 'checked' : ''}> AI</label>
            <label><input type="checkbox" class="c-fk" ${c.is_foreign_key ? 'checked' : ''}> FK</label>
            <label><input type="checkbox" class="c-unique" ${c.is_unique && !c.is_primary_key ? 'checked' : ''} ${c.is_primary_key ? 'disabled' : ''}> UQ</label>
            <label><input type="checkbox" class="c-null" ${c.is_nullable ? 'checked' : ''}> Null</label>
          </div>
          <button type="button" class="btn small danger" data-del-col="${c.id}" title="Delete column">✕</button>
        </div>
        <div class="c-custom-row${showCustom ? ' visible' : ''}">
          <label>Custom type<input type="text" class="c-custom-type" value="${esc(parsed.custom)}" placeholder="e.g. INT UNSIGNED"></label>
        </div>
        <div class="c-enum-bar col-enum-panel" ${showEnum ? '' : 'hidden'}>
          <span class="c-enum-summary">${esc(formatEnumSummary(parsed.enumValues))}</span>
          <button type="button" class="btn small c-enum-manage">Manage values…</button>
          <div class="c-enum-store" hidden aria-hidden="true">
            <input type="hidden" class="c-enum-json" value="${esc(JSON.stringify(parsed.enumValues || []))}">
          </div>
        </div>
      </div>`;
  }

  function bindColumnDragReorder(container) {
    if (!container) return;
    let dragRow = null;

    const clearDropHints = () => {
      container.querySelectorAll('.col-row').forEach((r) => {
        r.classList.remove('drag-over-top', 'drag-over-bottom');
      });
    };

    const moveDragRow = (targetRow, after) => {
      if (!dragRow || !targetRow || dragRow === targetRow) return;
      if (after) {
        targetRow.after(dragRow);
      } else {
        targetRow.before(dragRow);
      }
    };

    container.querySelectorAll('.col-row').forEach((row) => {
      const handle = row.querySelector('.c-drag-handle');
      if (!handle) return;

      handle.addEventListener('dragstart', (e) => {
        dragRow = row;
        row.classList.add('is-dragging');
        e.dataTransfer.effectAllowed = 'move';
        e.dataTransfer.setData('text/plain', row.dataset.colId || 'col');
      });

      handle.addEventListener('dragend', () => {
        row.classList.remove('is-dragging');
        dragRow = null;
        clearDropHints();
      });

      row.addEventListener('dragover', (e) => {
        if (!dragRow || dragRow === row) return;
        e.preventDefault();
        e.dataTransfer.dropEffect = 'move';
        clearDropHints();
        const rect = row.getBoundingClientRect();
        const after = e.clientY > rect.top + rect.height / 2;
        row.classList.add(after ? 'drag-over-bottom' : 'drag-over-top');
        moveDragRow(row, after);
      });

      row.addEventListener('drop', (e) => {
        e.preventDefault();
        clearDropHints();
        dragRow = null;
      });
    });
  }

  // --- Table modal ---
  function openTableModal(tableId) {
    state.editingTableId = tableId;
    const t = state.schema.tables.find((x) => x.id === tableId);
    if (!t) return;
    $('#tableModalTitle').textContent = t.name;
    $('#tableNameInput').value = t.name;
    renderColumnEditor(t);
    renderIndexEditor(t);
    $('#tableModal').showModal();
  }

  function indexColTagHTML(col) {
    return `<span class="idx-col-tag" data-col-id="${col.id}">
      <span class="idx-col-drag" draggable="true" title="Drag to reorder">⋮⋮</span>
      <span class="idx-col-label">${esc(col.name)}</span>
      <button type="button" class="idx-rm-col" title="Remove">×</button>
    </span>`;
  }

  function bindIndexColDragReorder(container) {
    if (!container || container.dataset.idxDragBound) return;
    container.dataset.idxDragBound = '1';
    let dragTag = null;

    const clearHints = () => {
      container.querySelectorAll('.idx-col-tag').forEach((t) => {
        t.classList.remove('is-dragging', 'drag-over-left', 'drag-over-right');
      });
    };

    const moveTag = (target, after) => {
      if (!dragTag || !target || dragTag === target) return;
      if (after) target.after(dragTag);
      else target.before(dragTag);
    };

    container.addEventListener('dragstart', (e) => {
      const handle = e.target.closest('.idx-col-drag');
      if (!handle || !container.contains(handle)) return;
      dragTag = handle.closest('.idx-col-tag');
      if (!dragTag) return;
      dragTag.classList.add('is-dragging');
      e.dataTransfer.effectAllowed = 'move';
      e.dataTransfer.setData('text/plain', dragTag.dataset.colId || 'col');
    });

    container.addEventListener('dragend', () => {
      dragTag?.classList.remove('is-dragging');
      dragTag = null;
      clearHints();
    });

    container.addEventListener('dragover', (e) => {
      if (!dragTag) return;
      const tag = e.target.closest('.idx-col-tag');
      if (!tag || !container.contains(tag) || dragTag === tag) return;
      e.preventDefault();
      e.dataTransfer.dropEffect = 'move';
      clearHints();
      const rect = tag.getBoundingClientRect();
      const after = e.clientX > rect.left + rect.width / 2;
      tag.classList.add(after ? 'drag-over-right' : 'drag-over-left');
      moveTag(tag, after);
    });

    container.addEventListener('drop', (e) => {
      if (!dragTag) return;
      e.preventDefault();
      clearHints();
      dragTag = null;
    });
  }

  function readIndexColumnIds(row) {
    return [...row.querySelectorAll('.idx-col-tag')]
      .map((el) => +el.dataset.colId)
      .filter(Boolean);
  }

  function addColToIndexRow(row, colId, columns) {
    const c = columns.find((x) => x.id === colId);
    if (!c) return;
    if (row.querySelector(`.idx-col-tag[data-col-id="${colId}"]`)) return;
    const tags = row.querySelector('.idx-col-tags');
    if (!tags) return;
    tags.querySelector('.idx-empty')?.remove();
    const span = document.createElement('span');
    span.className = 'idx-col-tag';
    span.dataset.colId = String(colId);
    span.innerHTML = `
      <span class="idx-col-drag" draggable="true" title="Drag to reorder">⋮⋮</span>
      <span class="idx-col-label">${esc(c.name)}</span>
      <button type="button" class="idx-rm-col" title="Remove">×</button>`;
    span.querySelector('.idx-rm-col')?.addEventListener('click', () => span.remove());
    tags.appendChild(span);
  }

  function bindIndexRowEvents(row, columns) {
    row.querySelector('.idx-add-col')?.addEventListener('change', (e) => {
      const id = +e.target.value;
      if (!id) return;
      addColToIndexRow(row, id, columns);
      e.target.value = '';
    });
    row.querySelectorAll('.idx-rm-col').forEach((btn) => {
      btn.addEventListener('click', () => btn.closest('.idx-col-tag')?.remove());
    });
  }

  function indexRowHTML(idx, columns) {
    const colIds = idx.column_ids || [];
    const tags = colIds.map((id) => {
      const c = columns.find((x) => x.id === id);
      if (!c) return '';
      return indexColTagHTML(c);
    }).join('');
    const addOpts = columns.map((c) => `<option value="${c.id}">${esc(c.name)}</option>`).join('');
    return `
      <div class="idx-row" data-idx-id="${idx.id}">
        <input type="text" class="idx-name" value="${esc(idx.name)}" placeholder="Index name" title="Index name">
        <label class="idx-unique-label"><input type="checkbox" class="idx-unique" ${idx.is_unique ? 'checked' : ''}> Unique</label>
        <div class="idx-cols-wrap">
          <div class="idx-col-tags">${tags || '<span class="idx-empty">No columns — add below</span>'}</div>
          <select class="idx-add-col" title="Add column to index">
            <option value="">+ Add column…</option>
            ${addOpts}
          </select>
        </div>
        <button type="button" class="btn small danger" data-del-idx="${idx.id}" title="Delete index">✕</button>
      </div>`;
  }

  function renderIndexEditor(t) {
    const box = $('#indexEditor');
    if (!box) return;
    const columns = [...(t.columns || [])].sort((a, b) => (a.sort_order - b.sort_order) || (a.id - b.id));
    const indexes = [...(t.indexes || [])].sort((a, b) => (a.sort_order - b.sort_order) || (a.id - b.id));
    if (!indexes.length) {
      box.innerHTML = '<div class="empty idx-empty-state">No indexes yet. Add one to speed up queries on selected columns.</div>';
      return;
    }
    box.innerHTML = `
      <div class="idx-editor-table">
        <div class="idx-editor-header">
          <span>Name</span>
          <span>Unique</span>
          <span>Columns</span>
          <span></span>
        </div>
        <div class="idx-editor-rows">${indexes.map((idx) => indexRowHTML(idx, columns)).join('')}</div>
      </div>`;
    box.querySelectorAll('.idx-row').forEach((row) => bindIndexRowEvents(row, columns));
    box.querySelectorAll('.idx-col-tags').forEach((tags) => bindIndexColDragReorder(tags));
    box.querySelectorAll('[data-del-idx]').forEach((b) => {
      b.addEventListener('click', async () => {
        if (!confirm('Delete this index?')) return;
        await api(`/api/indexes/${b.dataset.delIdx}`, { method: 'DELETE' });
        await loadSchema();
        openTableModal(state.editingTableId);
      });
    });
  }

  $('#btnAddIndex')?.addEventListener('click', async () => {
    const t = state.schema?.tables?.find((x) => x.id === state.editingTableId);
    if (!t) return;
    const cols = [...(t.columns || [])].sort((a, b) => (a.sort_order - b.sort_order) || (a.id - b.id));
    const pick = cols.find((c) => !c.is_primary_key) || cols[0];
    if (!pick) return toast('Add columns before creating an index', 'error');
    const base = (t.name || 'table').toLowerCase().replace(/[^a-z0-9_]+/g, '_');
    const n = (t.indexes?.length || 0) + 1;
    try {
      await api(`/api/tables/${state.editingTableId}/indexes`, {
        method: 'POST',
        body: JSON.stringify({
          name: `idx_${base}_${n}`,
          is_unique: false,
          column_ids: [pick.id],
          sort_order: (t.indexes?.length || 0),
        }),
      });
      await loadSchema();
      openTableModal(state.editingTableId);
    } catch (err) {
      toast(err.message, 'error');
    }
  });

  function renderColumnEditor(t) {
    const box = $('#columnEditor');
    const cols = [...(t.columns || [])].sort((a, b) => (a.sort_order - b.sort_order) || (a.id - b.id));
    let pkAssigned = false;
    let aiAssigned = false;
    const rows = cols.map((c) => {
      const col = { ...c };
      if (col.is_primary_key) {
        if (pkAssigned) col.is_primary_key = false;
        else pkAssigned = true;
      }
      if (col.is_auto_increment) {
        if (aiAssigned) col.is_auto_increment = false;
        else aiAssigned = true;
      }
      return columnRowHTML(col);
    }).join('');
    box.innerHTML = `
      <div class="col-editor-table">
        <div class="col-editor-header">
          <span aria-hidden="true"></span>
          <span>Column</span>
          <span>Type</span>
          <span>Length</span>
          <span>Default</span>
          <span>Flags</span>
          <span></span>
        </div>
        <div class="col-editor-rows">${rows || '<div class="empty" style="padding:1rem;margin:0;border:none;">No columns yet.</div>'}</div>
      </div>`;
    const list = box.querySelector('.col-editor-rows');
    list?.querySelectorAll('.col-row').forEach((row) => bindColumnRowEvents(row));
    bindColumnDragReorder(list);
    box.querySelectorAll('[data-del-col]').forEach((b) => {
      b.addEventListener('click', async () => {
        if (!confirm('Delete column?')) return;
        await api(`/api/columns/${b.dataset.delCol}`, { method: 'DELETE' });
        await loadSchema();
        openTableModal(state.editingTableId);
      });
    });
  }

  $('#btnAddColumn')?.addEventListener('click', async () => {
    const name = prompt('Column name:');
    if (!name?.trim()) return;
    const sortOrder = $('#columnEditor').querySelectorAll('.col-row').length;
    try {
      await api(`/api/tables/${state.editingTableId}/columns`, {
        method: 'POST',
        body: JSON.stringify({
          name: name.trim(),
          data_type: 'VARCHAR(255)',
          is_nullable: true,
          sort_order: sortOrder,
        }),
      });
      await loadSchema();
      openTableModal(state.editingTableId);
    } catch (err) {
      toast(err.message, 'error');
    }
  });

  $('#btnSaveTable')?.addEventListener('click', async () => {
    const rows = [...$('#columnEditor').querySelectorAll('.col-row')];
    const pkRow = rows.find((row) => row.querySelector('.c-pk')?.checked) || null;
    const aiRow = rows.find((row) => {
      const ai = row.querySelector('.c-ai');
      return ai?.checked && !ai.disabled;
    }) || null;
    try {
      await api(`/api/tables/${state.editingTableId}`, {
        method: 'PUT',
        body: JSON.stringify({ name: $('#tableNameInput').value.trim() }),
      });
      for (const [index, row] of rows.entries()) {
        const id = +row.dataset.colId;
        const defResult = validateAndFormatDefault(row);
        if (!defResult.ok) {
          markDefaultInvalid(row, true);
          toast(defResult.message, 'error');
          return;
        }
        const isPrimaryKey = row === pkRow;
        const payload = {
          name: row.querySelector('.c-name').value.trim(),
          data_type: readColumnTypeFromRow(row),
          is_primary_key: isPrimaryKey,
          is_auto_increment: row === aiRow,
          is_foreign_key: row.querySelector('.c-fk').checked,
          is_nullable: row.querySelector('.c-null').checked,
          is_unique: !!(row.querySelector('.c-unique')?.checked && !isPrimaryKey),
          default_value: defResult.sql,
          sort_order: index,
        };
        await api(`/api/columns/${id}`, { method: 'PUT', body: JSON.stringify(payload) });
      }
      const idxRows = $('#indexEditor')?.querySelectorAll('.idx-row') || [];
      for (const [index, row] of [...idxRows].entries()) {
        const name = row.querySelector('.idx-name')?.value.trim() || '';
        const columnIds = readIndexColumnIds(row);
        if (!name) {
          toast('Index name is required', 'error');
          return;
        }
        if (!columnIds.length) {
          toast(`Index "${name}" must include at least one column`, 'error');
          return;
        }
        await api(`/api/indexes/${row.dataset.idxId}`, {
          method: 'PUT',
          body: JSON.stringify({
            name,
            is_unique: row.querySelector('.idx-unique')?.checked || false,
            column_ids: columnIds,
            sort_order: index,
          }),
        });
      }
      $('#tableModal').close();
      await loadSchema();
      toast('Table saved');
    } catch (err) {
      toast(err.message, 'error');
    }
  });

  $('#btnDeleteTable')?.addEventListener('click', async () => {
    if (!confirm('Delete this table?')) return;
    try {
      await api(`/api/tables/${state.editingTableId}`, { method: 'DELETE' });
      $('#tableModal').close();
      await loadSchema();
      toast('Table deleted');
    } catch (err) {
      toast(err.message, 'error');
    }
  });

  $$('[data-close-rel]').forEach((btn) => btn.addEventListener('click', closeRelationModal));

  $('#btnSaveRelation')?.addEventListener('click', async () => {
    const pending = state.pendingRelation;
    const nameRaw = $('#relName')?.value.trim()
      || (pending ? defaultFKName(pending.from_table_id, pending.from_column_id) : '');
    if (!/^[a-zA-Z0-9_]+$/.test(nameRaw)) {
      toast('Constraint name may only contain letters, numbers, and underscores', 'error');
      return;
    }
    const payload = {
      name: nameRaw,
      on_delete: $('#relOnDelete')?.value || 'RESTRICT',
      on_update: $('#relOnUpdate')?.value || 'RESTRICT',
    };
    const isNew = !state.editingRelationId;
    try {
      if (state.editingRelationId) {
        await api(`/api/relations/${state.editingRelationId}`, {
          method: 'PUT',
          body: JSON.stringify(payload),
        });
      } else if (state.pendingRelation) {
        await api(`/api/projects/${state.activeProjectId}/relations`, {
          method: 'POST',
          body: JSON.stringify({ ...state.pendingRelation, ...payload }),
        });
      } else {
        return;
      }
      closeRelationModal();
      await loadSchema();
      toast(isNew ? 'Relation created' : 'Relation updated');
    } catch (err) {
      toast(err.message, 'error');
    }
  });

  $('#btnDeleteRelation')?.addEventListener('click', async () => {
    if (!state.editingRelationId || !confirm('Delete this relation?')) return;
    try {
      await api(`/api/relations/${state.editingRelationId}`, { method: 'DELETE' });
      closeRelationModal();
      await loadSchema();
      toast('Relation deleted');
    } catch (err) {
      toast(err.message, 'error');
    }
  });

  $('#canvasScroll')?.addEventListener('scroll', () => drawRelations());
  window.addEventListener('resize', () => drawRelations());

  async function init() {
    initSqlPanel();
    bindCanvasSelection();
    setDesignerEnabled(false);
    rotateProjectsTagline();
    try {
      await loadProjects();
    } catch (err) {
      toast('Failed to load: ' + err.message, 'error');
    }
  }
  init();
})();
