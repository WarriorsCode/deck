# Changelog

All notable changes to this project will be documented in this file.

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
