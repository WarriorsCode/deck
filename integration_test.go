package deck_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/warriorscode/deck/config"
	"github.com/warriorscode/deck/engine"
)

func TestFullLifecycle(t *testing.T) {
	dir := t.TempDir()
	deckDir := filepath.Join(dir, ".deck")
	marker := filepath.Join(dir, "bootstrapped")
	hookMarker := filepath.Join(dir, "hooked")
	stopMarker := filepath.Join(dir, "stopped")

	cfgYAML := `
name: integration-test
bootstrap:
  - name: Bootstrap
    check: test -f ` + marker + `
    run: touch ` + marker + `
hooks:
  pre-start:
    - name: Pre-start hook
      run: touch ` + hookMarker + `
  post-stop:
    - name: Post-stop hook
      run: touch ` + stopMarker + `
services:
  worker:
    run: sleep 60
`
	cfg, err := config.Parse([]byte(cfgYAML))
	require.NoError(t, err)

	eng := engine.New(cfg, dir, deckDir)

	err = eng.Preflight(context.Background())
	require.NoError(t, err)

	_, err = os.Stat(marker)
	require.NoError(t, err)

	_, err = os.Stat(hookMarker)
	require.NoError(t, err)

	err = eng.Start(nil)
	require.NoError(t, err)

	statuses := eng.Status(nil)
	require.Len(t, statuses, 1)
	assert.Equal(t, "running", statuses[0].Status)

	eng.Stop()
	time.Sleep(500 * time.Millisecond)

	_, err = os.Stat(stopMarker)
	require.NoError(t, err)

	entries, _ := os.ReadDir(filepath.Join(deckDir, "pids"))
	assert.Empty(t, entries)
}

func TestBootstrapIdempotent(t *testing.T) {
	dir := t.TempDir()
	deckDir := filepath.Join(dir, ".deck")
	counter := filepath.Join(dir, "counter")

	cfgYAML := `
name: idempotent-test
bootstrap:
  - name: Count runs
    check: test -f ` + counter + `
    run: echo run >> ` + counter + `
services:
  worker:
    run: sleep 60
`
	cfg, err := config.Parse([]byte(cfgYAML))
	require.NoError(t, err)

	eng := engine.New(cfg, dir, deckDir)
	require.NoError(t, eng.Preflight(context.Background()))
	eng.Stop()

	eng2 := engine.New(cfg, dir, deckDir)
	require.NoError(t, eng2.Preflight(context.Background()))
	eng2.Stop()

	data, err := os.ReadFile(counter)
	require.NoError(t, err)
	assert.Equal(t, "run\n", string(data))
}
