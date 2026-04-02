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
| `deck up` | Foreground: preflight, start services, tail logs, ctrl+c to stop |
| `deck start` | Detached: preflight, start services, return to shell |
| `deck stop` | Stop all services |
| `deck restart` | Stop + start |
| `deck status` | Show service status (supports `--format json` and Go templates) |
| `deck logs` | Tail logs with colored prefixes |
| `deck init` | Create deck.yaml scaffold and update .gitignore |

## Configuration

```yaml
# deck.yaml
name: myproject

bootstrap:
  - name: Install deps
    check: test -d node_modules
    run: npm install

deps:
  postgres:
    check: pg_isready -h 127.0.0.1
    start:
      - docker run -d --name postgres -p 5432:5432 -e POSTGRES_PASSWORD=postgres postgres:16
      - brew services start postgresql@16
    stop:
      - docker stop postgres && docker rm postgres
      - brew services stop postgresql@16

hooks:
  pre-start:
    - name: Run migrations
      run: goose up
  post-stop: []

services:
  api:
    dir: ./api
    run: go run ./cmd/server
    port: 4000
    color: cyan
  webapp:
    dir: ./webapp
    run: pnpm dev
    port: 5173
    color: magenta
```

### Local overrides

Create `deck.local.yaml` (gitignored) to override team defaults:

```yaml
deps:
  postgres:
    start:
      - brew services start postgresql@16
```

Maps merge by key, lists replace entirely.

### Custom config file

```bash
deck up -f staging.yaml  # no local merge
```

## How It Works

1. **Deps** — checks each dependency, tries start strategies in order until check passes
2. **Bootstrap** — runs setup steps if their check fails (idempotent)
3. **Hooks** — pre-start hooks run before services launch
4. **Services** — started as child processes, managed via PID files
5. **Logs** — tailed with colored `[name]` prefixes, timestamps auto-detected
6. **Shutdown** — SIGTERM → 5s grace → SIGKILL, process group kill for child cleanup

## License

MIT
