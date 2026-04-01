package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseValidConfig(t *testing.T) {
	yaml := `
name: testproject
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
deps:
  postgres:
    check: pg_isready -h 127.0.0.1
    start:
      - docker run -d --name pg postgres:16
      - brew services start postgresql@16
    stop:
      - docker stop pg
bootstrap:
  - name: Install deps
    check: test -d node_modules
    run: pnpm install
hooks:
  pre-start:
    - name: Create DB
      run: createdb myapp
  post-stop:
    - name: Cleanup
      run: rm -rf tmp/cache
`
	cfg, err := Parse([]byte(yaml))
	require.NoError(t, err)
	assert.Equal(t, "testproject", cfg.Name)
	assert.Len(t, cfg.Services, 2)
	assert.Equal(t, "go run ./cmd/server", cfg.Services["api"].Run)
	assert.Equal(t, 4000, cfg.Services["api"].Port)
	assert.Equal(t, "cyan", cfg.Services["api"].Color)
	assert.True(t, cfg.Services["api"].TimestampEnabled())
	assert.Len(t, cfg.Deps, 1)
	assert.Equal(t, "pg_isready -h 127.0.0.1", cfg.Deps["postgres"].Check)
	assert.Len(t, cfg.Deps["postgres"].Start, 2)
	assert.Len(t, cfg.Bootstrap, 1)
	assert.Len(t, cfg.Hooks.PreStart, 1)
	assert.Len(t, cfg.Hooks.PostStop, 1)
}

func TestParseMinimalConfig(t *testing.T) {
	yaml := `
services:
  api:
    run: go run ./cmd/server
`
	cfg, err := Parse([]byte(yaml))
	require.NoError(t, err)
	assert.Len(t, cfg.Services, 1)
	assert.True(t, cfg.Services["api"].TimestampEnabled())
}

func TestValidateNoServices(t *testing.T) {
	yaml := `
name: empty
`
	_, err := Parse([]byte(yaml))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one service")
}

func TestValidateServiceMissingRun(t *testing.T) {
	yaml := `
services:
  api:
    dir: ./api
`
	_, err := Parse([]byte(yaml))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "run")
}

func TestValidateDepMissingCheck(t *testing.T) {
	yaml := `
services:
  api:
    run: go run .
deps:
  postgres:
    start:
      - docker run postgres
`
	_, err := Parse([]byte(yaml))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "check")
}

func TestValidateDepMissingStart(t *testing.T) {
	yaml := `
services:
  api:
    run: go run .
deps:
  postgres:
    check: pg_isready
`
	_, err := Parse([]byte(yaml))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "start")
}

func TestValidateBootstrapMissingCheck(t *testing.T) {
	yaml := `
services:
  api:
    run: go run .
bootstrap:
  - name: Install
    run: pnpm install
`
	_, err := Parse([]byte(yaml))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "check")
}

func TestValidateInvalidHookName(t *testing.T) {
	yaml := `
services:
  api:
    run: go run .
hooks:
  on-crash:
    - name: Notify
      run: echo crashed
`
	_, err := Parse([]byte(yaml))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "hook")
}

func TestStartStringOrList(t *testing.T) {
	yaml := `
services:
  api:
    run: go run .
deps:
  redis:
    check: redis-cli ping
    start: docker run -d redis
    stop: docker stop redis
`
	cfg, err := Parse([]byte(yaml))
	require.NoError(t, err)
	assert.Len(t, cfg.Deps["redis"].Start, 1)
	assert.Equal(t, "docker run -d redis", cfg.Deps["redis"].Start[0])
	assert.Len(t, cfg.Deps["redis"].Stop, 1)
}

func TestLoadFile(t *testing.T) {
	dir := t.TempDir()
	base := []byte(`
services:
  api:
    run: go run .
    port: 4000
`)
	local := []byte(`
services:
  api:
    port: 5000
`)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "deck.yaml"), base, 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "deck.local.yaml"), local, 0644))

	cfg, err := LoadFile(filepath.Join(dir, "deck.yaml"))
	require.NoError(t, err)
	assert.Equal(t, 5000, cfg.Services["api"].Port)
}

func TestLoadFileNoLocal(t *testing.T) {
	dir := t.TempDir()
	base := []byte(`
services:
  api:
    run: go run .
`)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "deck.yaml"), base, 0644))

	cfg, err := LoadFile(filepath.Join(dir, "deck.yaml"))
	require.NoError(t, err)
	assert.Len(t, cfg.Services, 1)
}

func TestLoadFileNotFound(t *testing.T) {
	_, err := LoadFile("/nonexistent/deck.yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "deck.yaml")
}
