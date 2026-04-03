---
title: Changelog
layout: default
nav_order: 4
---

# Changelog

All notable changes to this project will be documented in this file.

## [v0.7.0](https://github.com/WarriorsCode/deck/releases/tag/v0.7.0) — 2026-04-03

### Added
- Env vars now support object syntax with `value`, `script`, and `file` modes
- `file` mode reads values from structured files (JSON, YAML, TOML, INI/conf) using `path | key.path` dot-separated traversal
- `script` mode runs a shell command and captures stdout as the value
- INI/conf parser supports `[section]` headers, comments, and quoted values

### Changed
- `Env` type internally uses `EnvVar` struct instead of plain strings — YAML unmarshalling is fully backward-compatible (plain strings still work)

## [v0.6.0](https://github.com/WarriorsCode/deck/releases/tag/v0.6.0) — 2026-04-03

### Added
- `env_file` field on bootstrap steps — services, hooks, and bootstrap now all support dotenv files
- `deck run <service> -- <cmd>` command — run one-off commands in a service's environment (dir, env, env_file)
- Selective service targeting — all commands (`up`, `start`, `stop`, `restart`, `status`, `logs`) accept `[services...]`
- `deck doctor` command — check deps, bootstrap, and config status without starting anything (supports `--json`)
- `depends_on` and `ready` fields on services — dependency-ordered startup with readiness polling
- `deck init` stack detection — detects Go, Node, Python, Ruby, Rust and generates tailored config with correct package manager
- `restart` field on services (`always`, `on-failure`) — automatic crash recovery during `deck up`

### Changed
- Services start in topological order based on `depends_on` graph (cycle detection at config parse time)
- Selective targeting auto-expands to include transitive dependencies on start/up/restart
- `deck stop` with service names stops only those services without running post-stop hooks

## [v0.5.0](https://github.com/WarriorsCode/deck/releases/tag/v0.5.0) — 2026-04-03

### Added
- Per-step `env` field on bootstrap steps, hooks, and services with `$(…)` shell interpolation
- Env values containing `$(…)` are evaluated at runtime (not config load time) via `sh -c`
- Step-level `env` merges on top of global `env` — step values win on conflict
- Shell interpolation runs in the step's working directory for correct relative path resolution
- Failed `$(…)` commands produce an empty string and log a warning; the step continues normally

### Changed
- Introduced `config.Env` named type with `Merge` and `ToSlice` methods, replacing raw `map[string]string`
- Service `env` values now support `$(…)` interpolation (previously only literal values)

## [v0.4.0](https://github.com/WarriorsCode/deck/releases/tag/v0.4.0) — 2026-04-02

### Added
- `--version` flag with ldflags injection at build time and git fallback at runtime

## [v0.3.0](https://github.com/WarriorsCode/deck/releases/tag/v0.3.0) — 2026-04-02

### Added
- Interactive `prompt` field on bootstrap steps — reads multi-line input, exposes `$DECK_INPUT` and `$DECK_INPUT_FILE`
- Non-TTY detection — prompts fail gracefully in CI/pipes instead of blocking

## [v0.2.0](https://github.com/WarriorsCode/deck/releases/tag/v0.2.0) — 2026-04-02

### Added
- Global `env` map — injected into all commands (bootstrap, hooks, services)
- Per-service `env` and `env_file` — load dotenv files and override env vars per service
- Per-hook `env_file` — load dotenv files for lifecycle hooks
- `dir` field on bootstrap steps and hooks
- Log backlog — shows last 20 lines when tailing starts
- ANSI escape code stripping from service log output

### Fixed
- Broadened ANSI stripping to cover all CSI sequences (dim, underline, cursor, etc.)

## [v0.1.0](https://github.com/WarriorsCode/deck/releases/tag/v0.1.0) — 2026-04-02

Initial release.

### Added
- Config parsing with `deck.yaml` + `deck.local.yaml` deep merge (order-preserving)
- `StringOrList` type — dep `start`/`stop` accept string or list
- Duplicate key rejection in config
- Dependency checker with multi-strategy fallback and polling
- Bootstrap step runner with idempotent check/run pattern
- Lifecycle hooks — pre-start (fail-fast) and post-stop (best-effort)
- Process manager — PID files, process group kill, SIGTERM → SIGKILL, stale PID cleanup
- Startup rollback — stops already-started services if a later service fails
- Log tailing with colored `[name]` prefixes and timestamp auto-detection
- Status formatters — table, JSON, Go template output
- Status shows all configured services with ports and log paths
- Engine orchestrator wiring the full lifecycle
- CLI commands: `up`, `start`, `stop`, `restart`, `status`, `logs`, `init`
- Graceful shutdown with 30s timeout, second signal force-exits
- Correct shutdown ordering: `deck up` runs post-stop hooks before kill, `deck stop` kills first
- goreleaser with Homebrew cask publishing
