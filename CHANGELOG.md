# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [wakapi-v2.17.4-fork.1] - 2026-06-21

Fork of upstream wakapi v2.17.4.

### Added
- Embeddable charts (shareables): per-user, revocable SVG/JSON charts at `/share/@user/<uuid>.{svg,json}` for coding activity, languages, editors, operating systems and categories — SVG for `<img>` embeds, JSON scoped to each chart's type. Managed from the Integrations tab in settings, gated by the `shares_enabled` config flag.
