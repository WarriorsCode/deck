package engine

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/warriorscode/deck/config"
)

func TestEngineStartStop(t *testing.T) {
	dir := t.TempDir()
	deckDir := filepath.Join(dir, ".deck")
	marker := filepath.Join(dir, "hook-ran")

	cfg := &config.Config{
		Name: "test",
		Deps: config.MapOf[config.Dep](
			"fake", config.Dep{Check: "true", Start: config.StringOrList{"true"}},
		),
		Bootstrap: []config.BootstrapStep{
			{Name: "noop", Check: "true", Run: "true"},
		},
		Hooks: config.Hooks{
			PreStart: []config.Hook{{Name: "marker", Run: "touch " + marker}},
		},
		Services: config.MapOf[config.Service](
			"sleeper", config.Service{Run: "sleep 60"},
		),
	}

	eng := New(cfg, dir, deckDir)

	err := eng.Preflight(context.Background())
	require.NoError(t, err)

	err = eng.Start()
	require.NoError(t, err)

	_, err = os.Stat(marker)
	require.NoError(t, err)

	statuses := eng.Status()
	require.Len(t, statuses, 1)
	assert.Equal(t, "running", statuses[0].Status)

	eng.Stop()

	time.Sleep(200 * time.Millisecond)
	statuses = eng.Status()
	require.Len(t, statuses, 1)
	assert.Equal(t, "stopped", statuses[0].Status)
}

func TestEngineStartRollback(t *testing.T) {
	dir := t.TempDir()
	deckDir := filepath.Join(dir, ".deck")

	cfg := &config.Config{
		Name: "rollback-test",
		Services: config.MapOf[config.Service](
			"good", config.Service{Run: "sleep 60"},
			"bad", config.Service{Dir: "/nonexistent-dir-that-wont-exist", Run: "sleep 60"},
		),
	}

	eng := New(cfg, dir, deckDir)
	err := eng.Start()
	require.Error(t, err)

	// "good" should have been rolled back — no PID files left
	time.Sleep(200 * time.Millisecond)
	entries, _ := os.ReadDir(filepath.Join(deckDir, "pids"))
	assert.Empty(t, entries)
}
