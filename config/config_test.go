package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustGet[V any](t *testing.T, m Map[V], key string) V {
	t.Helper()
	v, ok := m.Get(key)
	require.True(t, ok, "key %q not found", key)
	return v
}

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
	assert.Equal(t, 2, cfg.Services.Len())

	api := mustGet(t, cfg.Services, "api")
	assert.Equal(t, "go run ./cmd/server", api.Run)
	assert.Equal(t, 4000, api.Port)
	assert.Equal(t, "cyan", api.Color)
	assert.True(t, api.TimestampEnabled())

	assert.Equal(t, 1, cfg.Deps.Len())
	pg := mustGet(t, cfg.Deps, "postgres")
	assert.Equal(t, "pg_isready -h 127.0.0.1", pg.Check)
	assert.Len(t, pg.Start, 2)
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
	assert.Equal(t, 1, cfg.Services.Len())
	api := mustGet(t, cfg.Services, "api")
	assert.True(t, api.TimestampEnabled())
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
	redis := mustGet(t, cfg.Deps, "redis")
	assert.Len(t, redis.Start, 1)
	assert.Equal(t, "docker run -d redis", redis.Start[0])
	assert.Len(t, redis.Stop, 1)
}

func TestParsePreservesServiceOrder(t *testing.T) {
	yaml := `
services:
  alpha:
    run: echo a
  beta:
    run: echo b
  gamma:
    run: echo c
`
	cfg, err := Parse([]byte(yaml))
	require.NoError(t, err)
	assert.Equal(t, []string{"alpha", "beta", "gamma"}, cfg.Services.Keys())
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
	api := mustGet(t, cfg.Services, "api")
	assert.Equal(t, 5000, api.Port)
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
	assert.Equal(t, 1, cfg.Services.Len())
}

func TestLoadFileNotFound(t *testing.T) {
	_, err := LoadFile("/nonexistent/deck.yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "deck.yaml")
}
