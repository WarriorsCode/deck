<p align="center">
  <img src="icon.png" width="300" alt="deck icon" />
</p>

# deck

Lightweight local dev orchestrator. One command to bootstrap and run your entire stack.

Named after the cyberdeck from Shadowrun — the rig a decker jacks into to run programs, manage connections, and control their environment. `deck` does the same for your local dev stack: one config, one command, everything running.

## Install

```bash
# Homebrew
brew install --cask warriorscode/tap/deck

# Go
go install github.com/warriorscode/deck/cmd/deck@latest
```

## Quick Start

```bash
deck init        # creates deck.yaml + updates .gitignore
# edit deck.yaml with your services
deck up          # starts everything, ctrl+c to stop
```

## Commands

| Command | Description |
|---------|-------------|
| `deck up [services...]` | Foreground: preflight, start services, tail logs, ctrl+c to stop |
| `deck start [services...]` | Detached: preflight, start services, return to shell |
| `deck stop [services...]` | Stop services (all if none specified) |
| `deck restart [services...]` | Stop + start |
| `deck status [services...]` | Show service status (supports `--format json` and Go templates) |
| `deck logs [services...]` | Tail logs with colored prefixes (shows last 20 lines of backlog) |
| `deck run <service> -- <cmd>` | Run a one-off command in a service's environment (dir, env, env_file) |
| `deck doctor` | Check deps, bootstrap, and config without starting anything (supports `--json`) |
| `deck init` | Create deck.yaml scaffold and update .gitignore |
| `deck --version` | Print version |

## Configuration

```yaml
# deck.yaml
name: myproject

# Global environment variables — injected into all commands.
env:
  PGPASSWORD: postgres
  PG_HOST: 127.0.0.1

# External dependencies the stack needs running.
deps:
  postgres:
    check: pg_isready -h 127.0.0.1
    start:
      - docker run -d --name postgres -p 5432:5432 -e POSTGRES_PASSWORD=postgres postgres:16
      - brew services start postgresql@16
    stop:
      - docker stop postgres && docker rm postgres
      - brew services stop postgresql@16

# One-time setup tasks. Only run if check fails.
bootstrap:
  - name: Install deps
    dir: ./webapp
    check: test -d node_modules
    run: pnpm install

  - name: Create database
    env:
      PG_HOST:
        file: "api/etc/app.conf | db.host"
      PG_USER:
        file: "api/etc/app.conf | db.user"
    check: psql -h $PG_HOST -U $PG_USER -d myapp -c 'SELECT 1' 2>/dev/null
    run: createdb -h $PG_HOST -U $PG_USER myapp

  - name: Set auth key
    check: "! grep -q \"AUTH_KEY=''\" .env"
    prompt: |
      Paste the PEM public key from your auth provider.
      Press Enter on an empty line when done.
    run: ./scripts/set-auth-key.sh "$DECK_INPUT_FILE"

# Lifecycle hooks.
hooks:
  pre-start:
    - name: Run migrations
      dir: ./api
      env_file: ./etc/app.env
      run: goose up
  post-stop: []

# Services to manage.
services:
  api:
    dir: ./api
    run: go run ./cmd/server
    port: 4000
    color: cyan
    env_file: ./etc/app.env
    env:
      NO_COLOR: "1"
    ready: curl -sf http://localhost:4000/healthz
  webapp:
    dir: ./webapp
    run: pnpm dev
    port: 5173
    color: magenta
    depends_on: [api]
```

### Config fields

| Field | Where | Description |
|-------|-------|-------------|
| `env` | top-level, service, bootstrap, hook | Env vars — string value, `$(…)` interpolation, or object with `value`/`script`/`file` (see below). |
| `env_file` | service, hook, bootstrap | Path to a dotenv file loaded before running. |
| `dir` | service, bootstrap, hook | Working directory for the command. |
| `check` | dep, bootstrap | Shell command — exit 0 means "already done, skip". |
| `start`/`stop` | dep | String or list of strategies tried in order. |
| `prompt` | bootstrap | Interactive multi-line prompt. Input available as `$DECK_INPUT` and `$DECK_INPUT_FILE`. |
| `color` | service | Log prefix color (cyan, magenta, yellow, green, blue, red). Auto-assigned if omitted. |
| `timestamp` | service | Inject timestamps into log lines (default true, auto-detects existing timestamps). |
| `depends_on` | service | List of services that must start (and be ready) first. |
| `ready` | service | Shell command polled after start — exit 0 means ready. Blocks dependents until passing. |
| `restart` | service | Restart policy: `always`, `on-failure`, or omit for no restart. Active during `deck up`. |
| `port` | service | For status display only. |

### Env var syntax

Env vars can be a plain string or an object:

```yaml
env:
  # Plain string
  NO_COLOR: "1"

  # Shell interpolation (legacy syntax, still supported)
  PG_HOST: "$(grep host config.ini | cut -d= -f2)"

  # Explicit script — stdout becomes the value
  PG_HOST:
    script: "grep host config.ini | cut -d= -f2"

  # Read from structured file — supports .json, .yaml, .toml, .conf/.ini
  PG_HOST:
    file: "api/etc/app.conf | db.host"

  # Static value (same as plain string)
  PG_PORT:
    value: "5432"
```

The `file` format is `path | key.path` where the key path uses dots to traverse nested sections. For INI/conf files, section headers become the first key level: `[db]` + `host = localhost` → `db.host`.

### Local overrides

Create `deck.local.yaml` (gitignored) to override team defaults:

```yaml
deps:
  postgres:
    start:
      - brew services start postgresql@16
```

Maps merge by key (preserving order), lists replace entirely.

### Custom config file

```bash
deck up -f staging.yaml  # no local merge
```

## How It Works

1. **Deps** — checks each dependency, tries start strategies in order until check passes
2. **Bootstrap** — runs setup steps if their check fails (idempotent), supports interactive prompts and `$(…)` env interpolation
3. **Hooks** — pre-start hooks run before services, post-stop hooks run on shutdown
4. **Services** — started in dependency order (topological sort), `ready` checks polled before dependents proceed
5. **Logs** — tailed with colored `[name]` prefixes, ANSI codes stripped, timestamps auto-detected
6. **Shutdown** — post-stop hooks → SIGTERM → 5s grace → SIGKILL, process group kill for child cleanup. Second ctrl+c force-exits.

## License

MIT
