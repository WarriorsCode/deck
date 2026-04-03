---
title: Home
layout: home
nav_order: 1
---

<p align="center">
  <img src="{{ site.baseurl }}/assets/icon.png" width="250" alt="deck" />
</p>

# deck
{: .fs-9 .text-center }

Lightweight local dev orchestrator. One command to bootstrap and run your entire stack.
{: .fs-6 .fw-300 .text-center }

[Get Started](#install){: .btn .btn-primary .fs-5 .mb-4 .mb-md-0 .mr-2 }
[View on GitHub](https://github.com/WarriorsCode/deck){: .btn .fs-5 .mb-4 .mb-md-0 }

---

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

`deck init` detects your project stack (Go, Node, Python, Ruby, Rust) and generates a tailored config with the right commands and package manager.

## How It Works

1. **Deps** — checks each dependency, tries start strategies in order until check passes
2. **Bootstrap** — runs setup steps if their check fails (idempotent), supports interactive prompts and env interpolation
3. **Hooks** — pre-start hooks run before services, post-stop hooks run on shutdown
4. **Services** — started in dependency order (topological sort), `ready` checks polled before dependents proceed
5. **Logs** — tailed with colored `[name]` prefixes, ANSI codes stripped, timestamps auto-detected
6. **Shutdown** — post-stop hooks, SIGTERM, 5s grace, SIGKILL, process group kill for child cleanup. Second ctrl+c force-exits.
