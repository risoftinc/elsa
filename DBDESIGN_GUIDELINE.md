# Elsa DB Designer - Complete Guide

**Elsa DB Designer** is a local visual database schema tool with drag-and-drop tables, PK/FK indicators, indexes, bidirectional SQL sync, and diagram export — built into Elsa as a single Go binary (same dark UI style as Env Manager).

> All project data is stored in a local SQLite database under your user config directory. The web server binds to `127.0.0.1` by default.

## Quick Start

```bash
elsa dbdesign serve
```

Opens **http://127.0.0.1:1998** by default.

Optional flags: `--host`, `--port` (default `1998`), `--db` (custom SQLite path), `--no-browser`.

## Workflow

### 1. Projects

Create a workspace with name and optional description. Each project has its own tables, relations, and SQL dialect (**MySQL**, **PostgreSQL**, or **SQLite**). Switch dialect in the toolbar — export and the SQL panel follow the active dialect.

### 2. Tables and columns

- **Add table** — toolbar button or context on canvas
- **Edit** — double-click a table header
- **Move** — drag the table header; positions are saved automatically
- **Columns** — name, type, nullable, **PK** (one per table), **UQ**, default value
- **Reorder columns** — drag rows in the table editor
- **ENUM / SET** — for MySQL-style types, use **Edit values** to manage allowed values (single list or bulk paste)

PK and UQ are mutually exclusive in the UI: marking a column as PK disables UQ on that row.

### 3. Indexes

In the table editor, open the **Indexes** section:

- Add named indexes (composite columns supported)
- Toggle **UNIQUE**
- Reorder index columns with drag handles

Indexes appear on the canvas (truncated with ellipsis when long) and are included in SQL export/import.

### 4. Relations (foreign keys)

- **Create** — drag from a column on the child table to a column on the parent (FK → PK)
- **Edit** — click the relation line on the canvas to open the FK modal:
  - Constraint **name** (default `fk_{table}_{column}`)
  - **ON DELETE** / **ON UPDATE**: `RESTRICT`, `CASCADE`, `SET NULL`, `NO ACTION`, `SET DEFAULT`
- **Delete** — use **Delete relation** inside the FK modal (clicking the line no longer deletes it)

### 5. Auto layout

Toolbar **Auto layout** arranges tables left-to-right by FK depth (referenced tables on the left). Useful after importing SQL or when the canvas is cluttered.

### 6. Multi-select

- Drag on empty canvas to box-select tables
- **Ctrl** / **Shift**+click to add or toggle selection
- Drag any selected header to move all selected tables together
- **Esc** clears selection

### 7. SQL panel (bidirectional sync)

The left **SQL** panel stays in sync with the diagram:

- Diagram changes regenerate SQL automatically
- Editing SQL in the panel imports back into the canvas (debounced)
- **Copy**, **Download**, and dialect selector work from the toolbar
- **Hide SQL** toggles the panel without losing sync

Import replaces the current project schema when parsing succeeds (`replace: true` on the API).

### 8. Canvas vs diagram export

| | Canvas | Export (PNG / SVG) |
|---|--------|------------------|
| Table width | Capped (~280px); long names/index labels ellipsis | Full text width |
| Positions | User drag positions | **Same positions** as canvas; spacing nudged only to avoid overlap after widening |
| Relation lines | Anchored at column row | Anchored at **table box edges** at the same row Y |
| Theme | Dark (app UI) | **Dark** or **Minimal (print)** white background |

**Export diagram** opens a **preview modal** first — review the image, then **Download** or **Cancel**.

## Commands

| Command | Description |
|---------|-------------|
| `elsa dbdesign serve` | Start web UI |
| `--port` | Default `1998` |
| `--host` | Bind address (default `127.0.0.1`) |
| `--db` | Custom SQLite path |
| `--no-browser` | Skip auto-open browser |

## REST API (summary)

| Method | Path | Purpose |
|--------|------|---------|
| GET/POST | `/api/projects` | List / create projects |
| GET/PUT/DELETE | `/api/projects/{id}` | Project CRUD |
| GET | `/api/projects/{id}/schema` | Full schema (tables, columns, relations, indexes) |
| POST | `/api/projects/{id}/export` | Generate SQL for dialect |
| POST | `/api/projects/{id}/import` | Import SQL `{ "sql": "...", "replace": true }` |
| POST | `/api/projects/{id}/tables` | Create table |
| PUT/DELETE | `/api/tables/{id}` | Update / delete table |
| POST | `/api/tables/{id}/columns` | Add column |
| PUT/DELETE | `/api/columns/{id}` | Update / delete column |
| POST | `/api/tables/{id}/indexes` | Create index |
| PUT/DELETE | `/api/indexes/{id}` | Update / delete index |
| POST | `/api/projects/{id}/relations` | Create relation (name, on_delete, on_update) |
| PUT/DELETE | `/api/relations/{id}` | Update / delete relation |

## Data storage

`%APPDATA%\elsa\elsadbdesign.db` (Windows)

Equivalent user config directories on macOS and Linux.

## See also

- [Env Manager Guide](ENV_GUIDELINE.md)
- [README](README.md)
