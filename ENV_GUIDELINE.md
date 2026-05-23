# Elsa Env Manager - Complete Guide

**Elsa Env Manager** is a local environment variable manager with a built-in web UI. Manage environment groups (local, staging, production), compare values side-by-side, define export templates with Go `text/template` syntax, and generate `.env` files — all stored privately on your machine.

> 🔒 **Privacy**: All data is stored in a local SQLite database under your user config directory. The web server binds to `127.0.0.1` by default and is not published or synced anywhere.

## 🎯 Overview

Elsa Env Manager helps you:

- **Organize environments** as custom groups (local, staging, production, etc.)
- **Manage variables** with a comparison table across up to 5 environments at once
- **Export templates** using Go template syntax (`{{.DB_HOST}}`)
- **Generate files** as `.env` or custom filenames, or copy output to clipboard

The UI is embedded in the Elsa binary (`go:embed`) — a single executable, no separate frontend build step.

## 🏗️ Architecture

```
elsa env serve
      ↓
┌─────────────────────────────────────┐
│  HTTP Server (localhost)            │
│  ├── Embedded Web UI (dark mode)    │
│  └── REST API (/api/...)            │
└─────────────────────────────────────┘
      ↓
┌─────────────────────────────────────┐
│  SQLite (user config directory)     │
│  ├── environments                   │
│  ├── variables + values             │
│  └── templates                      │
└─────────────────────────────────────┘
```

| Component | Path |
|-----------|------|
| CLI command | `cmd/env/` |
| Business logic | `internal/envmanager/` |
| Web UI | `internal/envmanager/web/` |

## 🚀 Quick Start

### 1. Start the web UI

```bash
elsa env serve
```

The server starts at **http://127.0.0.1:1999** (default) and opens your system browser automatically.

### 2. Create environment groups

Open the **Environments** page and add groups such as:

- `local`
- `staging`
- `production`

Names must be unique. Sort order controls display order in lists.

### 3. Add variables

Open the **Variables** page:

1. Select up to **5 environment columns** from the **Columns** dropdown
2. Edit values per row in the comparison table
3. Add new keys in the footer row
4. Use **Search** to filter by key name (sorted A–Z)

Changes save automatically when you leave a cell (blur) or press Enter.

### 4. Create export templates

Open the **Templates** page:

1. Click **+ New template**
2. Set a unique name and template body
3. Use Go template syntax referencing your variable keys

Example template body:

```dotenv
DB_HOST={{.DB_HOST}}
DB_USER={{.DB_USER}}
DB_PASS={{.DB_PASS}}
DB_PORT={{.DB_PORT}}
```

### 5. Export

- **Per environment**: **Export .env** on the Environments page
- **From template**: **Export** on a template → choose environment → **Generate** → **Copy** or **Download**

## 📋 Features by Page

### Environments

| Feature | Description |
|---------|-------------|
| Create group | Custom name + sort order |
| Edit / Delete | Update name or remove group (values for that env are deleted) |
| Export .env | Generate dotenv file for one environment |

**Validation**

- Name is required
- Name must be unique (database constraint)

### Variables

| Feature | Description |
|---------|-------------|
| Comparison table | Keys in the left column; environment values in columns |
| Column picker | Dropdown, max **5** environments at a time |
| Search | Filter by key name |
| Pagination | 10 / 25 / 50 / 100 rows per page |
| Inline add | New key row in table footer |
| Per-row save | Auto-save on blur or Enter |

**Validation**

- Key is required and must be unique
- At least one environment column must be selected

### Templates

| Feature | Description |
|---------|-------------|
| Create / Edit / Delete | Manage named templates |
| Go templates | Use `{{.VARIABLE_KEY}}` — keys match variable names |
| Export | Render template for a chosen environment |
| Preview | Generate, copy, or download output |

**Validation**

- Name is required and unique (case-insensitive)
- Body is required

**Special keys**: If a variable key contains characters invalid for Go template field access (e.g. `MY-KEY`), use:

```gotemplate
{{index . "MY-KEY"}}
```

## 📟 Commands Reference

### `elsa env`

Parent command for environment management.

```bash
elsa env --help
```

### `elsa env reset`

Permanently delete the local env manager database. Requires typing a random **8-character** confirmation code (letters `a-z` / `A-Z` and digits `0-9` only — case-sensitive, no spaces or symbols).

```bash
elsa env reset
elsa env reset --db "C:\path\to\custom.db"
```

Example session:

```
⚠️  WARNING: This will permanently delete ALL env manager data.
Database: C:\Users\you\AppData\Roaming\elsa\elsaenvmanager.db
Type this exact code to confirm (8 letters/digits, case-sensitive):
aB3xK9mZ
> aB3xK9mZ
✅ Database deleted: ...
```

| Flag | Description |
|------|-------------|
| `--db` | Custom SQLite path (same as `serve`) |

> Stop `elsa env serve` first if the database file is locked.

### `elsa env serve`

Start the local web server and UI.

```bash
elsa env serve
elsa env serve --port 8080
elsa env serve --host 127.0.0.1 --port 3000
elsa env serve --db "C:\path\to\custom.db"
elsa env serve --no-browser
```

| Flag | Default | Description |
|------|---------|-------------|
| `--host` | `127.0.0.1` | Host to bind. Use `127.0.0.1` to keep traffic local |
| `--port` | `1999` | HTTP port |
| `--db` | *(auto)* | Custom SQLite database path |
| `--no-browser` | `false` | Do not open the system browser on start |

### Elsafile integration

Add to your project `Elsafile`:

```bash
# Start env manager UI
env:
	elsa env serve

# Custom port, no browser (e.g. CI / remote dev)
env/headless:
	elsa env serve --port 9473 --no-browser
```

Run with:

```bash
elsa run env
elsa run env/headless
```

## 💾 Data Storage

### Default database location

| OS | Path |
|----|------|
| Windows | `%APPDATA%\elsa\elsaenvmanager.db` |
| macOS | `~/Library/Application Support/elsa/elsaenvmanager.db` |
| Linux | `~/.config/elsa/elsaenvmanager.db` |

The config directory is created with restricted permissions (`0700`).

### Custom database

```bash
elsa env serve --db "./my-env.db"
```

Use a custom path when you want project-specific or portable storage.

### What is stored

| Table | Content |
|-------|---------|
| `environments` | Group names (local, staging, …) |
| `variables` | Env keys (`DB_HOST`, …) |
| `values` | Per-environment values |
| `templates` | Export template name + body |

**Not stored**: Generated `.env` files (export is on-demand only).

## 🔒 Security Best Practices

1. **Keep default host** `127.0.0.1` — avoids exposing the UI on your network
2. **Do not commit** database files or exported `.env` files to git
3. **Use `--no-browser`** on shared/headless machines if needed
4. **Backup** the SQLite file before major changes: copy `elsaenvmanager.db`

> ⚠️ Binding to `0.0.0.0` makes the UI reachable from other machines on your network. Only do this on trusted networks.

## 🌐 REST API

The web UI uses a JSON API (for reference or automation):

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/environments` | List environments |
| `POST` | `/api/environments` | Create environment |
| `PUT` | `/api/environments/{id}` | Update environment |
| `DELETE` | `/api/environments/{id}` | Delete environment |
| `POST` | `/api/environments/{id}/export` | Export dotenv for one env |
| `GET` | `/api/variables` | List variables with values |
| `POST` | `/api/variables` | Create variable |
| `PUT` | `/api/variables/{id}` | Update variable |
| `DELETE` | `/api/variables/{id}` | Delete variable |
| `GET` | `/api/templates` | List templates |
| `POST` | `/api/templates` | Create template |
| `PUT` | `/api/templates/{id}` | Update template |
| `DELETE` | `/api/templates/{id}` | Delete template |
| `POST` | `/api/templates/{id}/render` | Render template for an environment |

## 🛠️ Troubleshooting

### Browser does not open

```bash
elsa env serve --no-browser
# Open manually: http://127.0.0.1:1999
```

### Port already in use

```bash
elsa env serve --port 2026
```

### "template name already exists"

Template names are unique (case-insensitive). Choose a different name or edit the existing template.

### "Maximum 5 environment columns"

The comparison table supports at most 5 environment columns. Uncheck one environment before selecting another.

### Database locked (SQLite) / "being used by another process"

This is **normal on Windows** when the database file is still open — not a bug on your PC.

**Common cause:** `elsa env serve` is still running in another terminal.

**Fix:**
1. Find the terminal where `elsa env serve` is running
2. Press **Ctrl+C** to stop it
3. Run `elsa env reset` again

Also close DB browser tools (SQLite viewers) if they have `elsaenvmanager.db` open.

### Reset confirmation failed

The code is **case-sensitive** and must match exactly (8 characters). Run `elsa env reset` again to get a new code.

### Template renders empty values

Ensure:

1. The variable key exists (e.g. `DB_HOST`)
2. A value is set for the selected environment
3. Template syntax matches the key: `{{.DB_HOST}}`

## 💡 Best Practices

1. **Naming**: Use `SCREAMING_SNAKE_CASE` for keys (`DB_HOST`, `API_KEY`)
2. **Environments**: Mirror your deploy targets (`local`, `staging`, `production`)
3. **Templates**: One template per output format (`.env`, `config.yaml`, Docker env file)
4. **Compare**: Select staging + production columns to spot differences quickly
5. **Export**: Use templates for non-dotenv formats; use **Export .env** for standard dotenv

## 🔗 Related Documentation

- [Elsa README](README.md) — Main project overview
- [Elsa Init & Run Guide](INIT_RUN_GUIDELINE.md) — Elsafile custom commands
- [Migration Guide](MIGRATION_GUIDELINE.md) — Database migrations (separate from env manager DB)

---

**Elsa Env Manager** — Keep secrets local, compare environments easily, export with confidence. 🔐
