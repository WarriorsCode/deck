package scaffold

import (
	"os"
	"path/filepath"
	"strings"
)

// Stack represents a detected project stack.
type Stack struct {
	Name    string
	Dir     string // subdirectory, or "." if root
	absDir  string // absolute path for runtime detection
}

// Detect scans the directory for common project indicators.
// Returns all detected stacks in the directory tree (max 1 level deep).
func Detect(dir string) []Stack {
	var stacks []Stack

	// Check root
	for _, s := range detectIn(dir, ".") {
		stacks = append(stacks, s)
	}

	// Check immediate subdirectories
	entries, err := os.ReadDir(dir)
	if err != nil {
		return stacks
	}
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		sub := filepath.Join(dir, e.Name())
		for _, s := range detectIn(sub, e.Name()) {
			stacks = append(stacks, s)
		}
	}
	return stacks
}

type indicator struct {
	file string
	name string
}

var indicators = []indicator{
	{"go.mod", "go"},
	{"package.json", "node"},
	{"pyproject.toml", "python"},
	{"requirements.txt", "python"},
	{"Gemfile", "ruby"},
	{"Cargo.toml", "rust"},
}

func detectIn(dir, rel string) []Stack {
	var stacks []Stack
	for _, ind := range indicators {
		if _, err := os.Stat(filepath.Join(dir, ind.file)); err == nil {
			stacks = append(stacks, Stack{Name: ind.name, Dir: rel, absDir: dir})
			break // one detection per directory
		}
	}
	return stacks
}

// Generate returns a deck.yaml scaffold based on detected stacks.
func Generate(stacks []Stack, projectName string) string {
	if len(stacks) == 0 {
		return defaultScaffold(projectName)
	}

	var b strings.Builder
	b.WriteString("# deck.yaml — local dev stack configuration\n")
	b.WriteString("# See: https://github.com/warriorscode/deck\n\n")
	b.WriteString("name: " + projectName + "\n")

	// Collect bootstrap steps
	var bootstrapLines []string
	for _, s := range stacks {
		if lines := bootstrapFor(s); lines != "" {
			bootstrapLines = append(bootstrapLines, lines)
		}
	}
	if len(bootstrapLines) > 0 {
		b.WriteString("\nbootstrap:\n")
		for _, l := range bootstrapLines {
			b.WriteString(l)
		}
	}

	b.WriteString("\nservices:\n")
	for _, s := range stacks {
		b.WriteString(serviceFor(s))
	}

	return b.String()
}

func bootstrapFor(s Stack) string {
	dir := dirField(s.Dir)
	switch s.Name {
	case "node":
		mgr := detectNodePkgManager(s.absDir)
		return "  - name: Install deps\n" + dir + "    check: test -d node_modules\n    run: " + mgr + " install\n"
	case "python":
		return "  - name: Install deps\n" + dir + "    check: test -d .venv\n    run: python -m venv .venv && .venv/bin/pip install -e .\n"
	case "rust":
		return "  - name: Build\n" + dir + "    check: test -d target/debug\n    run: cargo build\n"
	default:
		return ""
	}
}

func serviceFor(s Stack) string {
	dir := dirField(s.Dir)
	name := serviceName(s)
	switch s.Name {
	case "go":
		return "  " + name + ":\n" + dir + "    run: go run .\n    # port: 8080\n"
	case "node":
		mgr := detectNodePkgManager(s.absDir)
		return "  " + name + ":\n" + dir + "    run: " + mgr + " dev\n    # port: 3000\n"
	case "python":
		return "  " + name + ":\n" + dir + "    run: .venv/bin/python -m flask run\n    # port: 5000\n"
	case "ruby":
		return "  " + name + ":\n" + dir + "    run: bundle exec rails server\n    # port: 3000\n"
	case "rust":
		return "  " + name + ":\n" + dir + "    run: cargo run\n    # port: 8080\n"
	default:
		return "  " + name + ":\n" + dir + "    run: echo 'replace with your start command'\n"
	}
}

func serviceName(s Stack) string {
	if s.Dir == "." {
		return "app"
	}
	return s.Dir
}

func dirField(dir string) string {
	if dir == "." {
		return ""
	}
	return "    dir: ./" + dir + "\n"
}

func detectNodePkgManager(dir string) string {
	for _, lock := range []struct {
		file string
		mgr  string
	}{
		{"pnpm-lock.yaml", "pnpm"},
		{"yarn.lock", "yarn"},
		{"bun.lockb", "bun"},
	} {
		if _, err := os.Stat(filepath.Join(dir, lock.file)); err == nil {
			return lock.mgr
		}
	}
	return "npm"
}

func defaultScaffold(name string) string {
	return `# deck.yaml — local dev stack configuration
# See: https://github.com/warriorscode/deck

name: ` + name + `

# One-time setup tasks. Only run if check fails.
# bootstrap:
#   - name: Install deps
#     check: test -d node_modules
#     run: npm install

# External dependencies.
# deps:
#   postgres:
#     check: pg_isready -h 127.0.0.1
#     start:
#       - docker run -d --name postgres -p 5432:5432 -e POSTGRES_PASSWORD=postgres postgres:16
#     stop:
#       - docker stop postgres && docker rm postgres

# Lifecycle hooks.
# hooks:
#   pre-start:
#     - name: Run migrations
#       run: migrate up
#   post-stop: []

# Services to manage.
services:
  app:
    run: echo "replace with your start command"
    # dir: ./src
    # port: 3000
    # color: cyan
`
}
