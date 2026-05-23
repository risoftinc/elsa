# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]
### Added
- 

### Features
- 

## [1.0.5] - 2026-05-23

- **Elsa Env Manager** — local environment variable manager with embedded web UI
  - `elsa env serve` — start localhost web server (default `127.0.0.1:1999`)
  - `elsa env reset` — delete local database with 8-character random confirmation code
  - **Environments** — custom groups (local, staging, production, etc.) with sort order
  - **Variables** — comparison table across environments, per-row edit, search, pagination (10/25/50/100)
  - **Templates** — Go `text/template` export (`.env` or custom filename, copy/download)
  - Environment column picker (dropdown, max 5 columns at a time)
  - Auto-open system browser when `serve` starts (`--no-browser` to disable)
  - SQLite storage at user config dir: `elsa/elsaenvmanager.db` (local only, not published)
  - REST API for environments, variables, and templates
  - `ENV_GUIDELINE.md` — complete feature documentation
