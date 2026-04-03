---
title: Configuration
layout: default
nav_order: 3
---

# Configuration

deck uses a `deck.yaml` file at the project root. Run `deck init` to generate one.

## Full example

```yaml
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

## Config fields

| Field | Where | Description |
|-------|-------|-------------|
| `env` | top-level, service, bootstrap, hook | Env vars — string, `$(…)` interpolation, or object with `value`/`script`/`file`. |
| `env_file` | service, hook, bootstrap | Path to a dotenv file loaded before running. |
| `dir` | service, bootstrap, hook | Working directory for the command. |
| `check` | dep, bootstrap | Shell command — exit 0 means "already done, skip". |
| `start`/`stop` | dep | String or list of strategies tried in order. |
| `prompt` | bootstrap | Interactive multi-line prompt. Input available as `$DECK_INPUT` and `$DECK_INPUT_FILE`. |
| `color` | service | Log prefix color (cyan, magenta, yellow, green, blue, red). Auto-assigned if omitted. |
| `timestamp` | service | Inject timestamps into log lines (default true, auto-detects existing timestamps). |
| `depends_on` | service | List of services that must start (and be ready) first. |
| `ready` | service | Shell command polled after start — exit 0 means ready. Blocks dependents. |
| `restart` | service | Restart policy: `always`, `on-failure`, or omit for no restart. Active during `deck up`. |
| `port` | service | For status display only. |

## Env var syntax

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

The `file` format is `path | key.path` where the key path uses dots to traverse nested sections. For INI/conf files, section headers become the first key level: `[db]` + `host = localhost` is `db.host`.

## Service dependencies

Services support `depends_on` for ordered startup and `ready` for readiness gating:

```yaml
services:
  api:
    run: go run ./cmd/server
    port: 4000
    ready: curl -sf http://localhost:4000/healthz
  webapp:
    run: pnpm dev
    depends_on: [api]
```

- Services start in topological order (dependencies first)
- Cycles are detected at config parse time
- `ready` is polled every 500ms with a 30s timeout
- Selective targeting auto-expands to include transitive dependencies

## Crash recovery

Services can be automatically restarted when they crash during `deck up`:

```yaml
services:
  api:
    run: go run ./cmd/server
    restart: on-failure  # or "always"
```

- `always` — restart on any exit
- `on-failure` — restart only on non-zero exit code
- Omit or leave empty for no restart (default)

The watch loop checks every 2 seconds.

## Local overrides

Create `deck.local.yaml` (gitignored) to override team defaults:

```yaml
deps:
  postgres:
    start:
      - brew services start postgresql@16
```

Maps merge by key (preserving order), lists replace entirely.

## Custom config file

```bash
deck up -f staging.yaml  # no local merge
```
