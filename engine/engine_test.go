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

func TestShutdownTimesOutHungHook(t *testing.T) {
	dir := t.TempDir()
	deckDir := filepath.Join(dir, ".deck")

	cfg := &config.Config{
		Name: "hung-hook-test",
		Hooks: config.Hooks{
			PostStop: []config.Hook{{Name: "hang", Run: "sleep 60"}},
		},
		Services: config.MapOf[config.Service](
			"svc", config.Service{Run: "sleep 60"},
		),
	}

	eng := New(cfg, dir, deckDir)
	require.NoError(t, eng.Start())

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	start := time.Now()
	eng.Shutdown(ctx)
	elapsed := time.Since(start)

	// Should complete in ~1s (context timeout), not 60s (hook sleep).
	assert.Less(t, elapsed, 5*time.Second)
	// Service should still be cleaned up despite hung hook.
	statuses := eng.Status()
	for _, s := range statuses {
		assert.Equal(t, "stopped", s.Status)
	}
}

func TestShutdownRunsPostStopBeforeKill(t *testing.T) {
	dir := t.TempDir()
	deckDir := filepath.Join(dir, ".deck")
	hookMarker := filepath.Join(dir, "post-stop-ran")

	cfg := &config.Config{
		Name: "shutdown-order-test",
		Hooks: config.Hooks{
			PostStop: []config.Hook{{Name: "mark", Run: "touch " + hookMarker}},
		},
		Services: config.MapOf[config.Service](
			"svc", config.Service{Run: "sleep 60"},
		),
	}

	eng := New(cfg, dir, deckDir)
	require.NoError(t, eng.Start())

	eng.Shutdown(context.Background())

	_, err := os.Stat(hookMarker)
	require.NoError(t, err, "post-stop hook should have run")

	statuses := eng.Status()
	for _, s := range statuses {
		assert.Equal(t, "stopped", s.Status)
	}
}
