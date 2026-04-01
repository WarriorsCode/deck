package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeOverrideDepStart(t *testing.T) {
	base := `
services:
  api:
    run: go run .
deps:
  postgres:
    check: pg_isready
    start:
      - docker run postgres
    stop:
      - docker stop postgres
`
	local := `
deps:
  postgres:
    start:
      - brew services start postgresql
    stop:
      - brew services stop postgresql
`
	cfg, err := ParseWithOverride([]byte(base), []byte(local))
	require.NoError(t, err)
	pg := mustGet(t, cfg.Deps, "postgres")
	assert.Equal(t, "pg_isready", pg.Check)
	assert.Equal(t, []string{"brew services start postgresql"}, []string(pg.Start))
	assert.Equal(t, []string{"brew services stop postgresql"}, []string(pg.Stop))
}

func TestMergeOverrideServicePort(t *testing.T) {
	base := `
services:
  api:
    run: go run .
    port: 4000
`
	local := `
services:
  api:
    port: 5000
`
	cfg, err := ParseWithOverride([]byte(base), []byte(local))
	require.NoError(t, err)
	api := mustGet(t, cfg.Services, "api")
	assert.Equal(t, "go run .", api.Run)
	assert.Equal(t, 5000, api.Port)
}

func TestMergeAddService(t *testing.T) {
	base := `
services:
  api:
    run: go run .
`
	local := `
services:
  worker:
    run: go run ./cmd/worker
`
	cfg, err := ParseWithOverride([]byte(base), []byte(local))
	require.NoError(t, err)
	assert.Equal(t, 2, cfg.Services.Len())
	worker := mustGet(t, cfg.Services, "worker")
	assert.Equal(t, "go run ./cmd/worker", worker.Run)
}

func TestMergeListReplacesEntirely(t *testing.T) {
	base := `
services:
  api:
    run: go run .
bootstrap:
  - name: Step A
    check: test -f a
    run: touch a
  - name: Step B
    check: test -f b
    run: touch b
`
	local := `
bootstrap:
  - name: Only step
    check: test -f c
    run: touch c
`
	cfg, err := ParseWithOverride([]byte(base), []byte(local))
	require.NoError(t, err)
	assert.Len(t, cfg.Bootstrap, 1)
	assert.Equal(t, "Only step", cfg.Bootstrap[0].Name)
}

func TestMergeNilLocal(t *testing.T) {
	base := `
services:
  api:
    run: go run .
`
	cfg, err := ParseWithOverride([]byte(base), nil)
	require.NoError(t, err)
	assert.Equal(t, 1, cfg.Services.Len())
}

func TestMergePreservesOrder(t *testing.T) {
	base := `
services:
  alpha:
    run: echo a
    port: 1000
  beta:
    run: echo b
  gamma:
    run: echo c
`
	local := `
services:
  beta:
    port: 2000
  delta:
    run: echo d
  epsilon:
    run: echo e
`
	cfg, err := ParseWithOverride([]byte(base), []byte(local))
	require.NoError(t, err)
	// Base order preserved, multiple new keys appended in local order.
	assert.Equal(t, []string{"alpha", "beta", "gamma", "delta", "epsilon"}, cfg.Services.Keys())
	beta := mustGet(t, cfg.Services, "beta")
	assert.Equal(t, 2000, beta.Port)
	assert.Equal(t, "echo b", beta.Run) // not overridden
}
