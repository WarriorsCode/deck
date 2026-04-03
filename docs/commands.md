---
title: Commands
layout: default
nav_order: 2
---

# Commands

| Command | Description |
|---------|-------------|
| `deck up [services...]` | Foreground: preflight, start services, tail logs, ctrl+c to stop |
| `deck start [services...]` | Detached: preflight, start services, return to shell |
| `deck stop [services...]` | Stop services (all if none specified) |
| `deck restart [services...]` | Stop + start |
| `deck status [services...]` | Show service status (supports `--format json` and Go templates) |
| `deck logs [services...]` | Tail logs with colored prefixes (shows last 20 lines of backlog) |
| `deck run <service> -- <cmd>` | Run a one-off command in a service's environment |
| `deck doctor` | Check deps, bootstrap, and config without starting anything |
| `deck init` | Create deck.yaml scaffold and update .gitignore |
| `deck --version` | Print version |

## Selective targeting

All lifecycle commands accept optional service names to operate on a subset of the stack:

```bash
deck up api           # just the API (+ its dependencies)
deck stop webapp      # stop one service
deck restart api      # restart one without touching others
deck logs api webapp  # tail specific services
```

When using `depends_on`, targeting a service auto-starts its transitive dependencies. `deck up webapp` will also start `api` if webapp depends on it.

`deck stop` with service names stops only those services — it does **not** run post-stop hooks.

## deck run

Run a one-off command in a service's resolved environment (dir, env, env_file, global env):

```bash
deck run api -- goose status
deck run api -- go test ./...
deck run webapp -- pnpm test
```

Useful for migrations, tests, REPL sessions, or any command that needs the same environment as the service.

## deck doctor

Walks the full config (deps, bootstrap, hooks, services) and reports status without starting anything:

```
$ deck doctor
✓ postgres         dep
✗ redis            dep
✓ Install deps     bootstrap (done)
✗ Create database  bootstrap (needed)
✓ api              service
⚠ webapp           service
  ⚠ env_file not found: ./etc/missing.env
```

Supports `--json` for machine-readable output.
