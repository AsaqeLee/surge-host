# Changelog

All notable changes to **surge-host** are documented here.  
Format follows [Keep a Changelog](https://keepachangelog.com/).

## [2.4.1] — 2026-07-01 — Final release

**Project status: complete.** Feature scope is frozen; only critical fixes are expected going forward.

### Added

- Unified editorial industrial UI (`web/static/css/system.css`) across home, upload, files, and edit pages
- Bracket-style header navigation: `[ Home ]` `[ Upload ]` `[ Manage ]`
- Homepage system readout, platform panels, endpoint specification, and public registry
- One-click and click-to-copy Raw URL (clipboard API with `execCommand` fallback)
- Startup warnings for weak admin password and default JWT secret
- Right-click to copy Raw URL in file list and editor

### Fixed

- Upload path normalization via `PrepareUserPath` (duplicate user prefix)
- HTML escaping for Git history `message` and `hash` fields (XSS)
- Homepage copy buttons: direct event binding, cache-busted `app.js`
- Homepage header now includes `[ Upload ]` on all pages

### Changed

- Replaced split `style.css` / `dashboard.css` with single `system.css` design system
- English README repositioned as multi-platform proxy config hosting
- Typography: Instrument Serif (titles) + JetBrains Mono (system metadata)

### Production

- Live instance: [rules.asaqe.site](https://rules.asaqe.site)
- Raw URL pattern: `https://rules.asaqe.site/raw/{user}/{filename}`

---

## [2.4.0] — prior

- Multi-platform support: Surge, Meta/Mihomo, sing-box
- JSON validation, YAML/JSON syntax highlighting in editor
- Docker Compose + 1Panel deployment path
- Git per-file versioning and rollback API