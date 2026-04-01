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
	assert.Equal(t, "pg_isready", cfg.Deps["postgres"].Check)
	assert.Equal(t, []string{"brew services start postgresql"}, []string(cfg.Deps["postgres"].Start))
	assert.Equal(t, []string{"brew services stop postgresql"}, []string(cfg.Deps["postgres"].Stop))
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
	assert.Equal(t, "go run .", cfg.Services["api"].Run)
	assert.Equal(t, 5000, cfg.Services["api"].Port)
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
	assert.Len(t, cfg.Services, 2)
	assert.Equal(t, "go run ./cmd/worker", cfg.Services["worker"].Run)
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
	assert.Len(t, cfg.Services, 1)
}
