package engine

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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

	err = eng.Start(nil)
	require.NoError(t, err)

	_, err = os.Stat(marker)
	require.NoError(t, err)

	statuses := eng.Status(nil)
	require.Len(t, statuses, 1)
	assert.Equal(t, "running", statuses[0].Status)

	eng.Stop()

	time.Sleep(200 * time.Millisecond)
	statuses = eng.Status(nil)
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
	err := eng.Start(nil)
	require.Error(t, err)

	// "good" should have been rolled back — no PID files left
	time.Sleep(200 * time.Millisecond)
	entries, _ := os.ReadDir(filepath.Join(deckDir, "pids"))
	assert.Empty(t, entries)
}

func TestStartRespectsTopoOrder(t *testing.T) {
	dir := t.TempDir()
	deckDir := filepath.Join(dir, ".deck")
	orderFile := filepath.Join(dir, "order")

	cfg := &config.Config{
		Name: "topo-test",
		Services: config.MapOf[config.Service](
			"web", config.Service{Run: "echo web >> " + orderFile + " && sleep 60", DependsOn: []string{"api"}},
			"api", config.Service{Run: "echo api >> " + orderFile + " && sleep 60"},
		),
	}

	eng := New(cfg, dir, deckDir)
	err := eng.Start(nil)
	require.NoError(t, err)
	defer eng.Stop()

	time.Sleep(500 * time.Millisecond)
	data, err := os.ReadFile(orderFile)
	require.NoError(t, err)
	lines := strings.TrimSpace(string(data))
	assert.Equal(t, "api\nweb", lines)
}

func TestStartFilterExpandsDeps(t *testing.T) {
	dir := t.TempDir()
	deckDir := filepath.Join(dir, ".deck")

	cfg := &config.Config{
		Name: "expand-test",
		Services: config.MapOf[config.Service](
			"web", config.Service{Run: "sleep 60", DependsOn: []string{"api"}},
			"api", config.Service{Run: "sleep 60"},
			"worker", config.Service{Run: "sleep 60"},
		),
	}

	eng := New(cfg, dir, deckDir)
	// Start only "web" — should auto-start "api" too, but not "worker".
	filter := cfg.ExpandDeps([]string{"web"})
	err := eng.Start(filter)
	require.NoError(t, err)
	defer eng.Stop()

	statuses := eng.Status(nil)
	statusMap := make(map[string]string)
	for _, s := range statuses {
		statusMap[s.Name] = s.Status
	}
	assert.Equal(t, "running", statusMap["api"])
	assert.Equal(t, "running", statusMap["web"])
	assert.Equal(t, "stopped", statusMap["worker"])
}

func TestStartWithReadyCheck(t *testing.T) {
	dir := t.TempDir()
	deckDir := filepath.Join(dir, ".deck")
	readyMarker := filepath.Join(dir, "ready")

	cfg := &config.Config{
		Name: "ready-test",
		Services: config.MapOf[config.Service](
			// Service creates the ready marker after a short delay.
			"svc", config.Service{
				Run:   "sleep 0.5 && touch " + readyMarker + " && sleep 60",
				Ready: "test -f " + readyMarker,
			},
		),
	}

	eng := New(cfg, dir, deckDir)
	err := eng.Start(nil)
	require.NoError(t, err)
	defer eng.Stop()

	// If we got here, Start() waited for the ready check to pass.
	_, err = os.Stat(readyMarker)
	require.NoError(t, err)
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
	require.NoError(t, eng.Start(nil))

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	start := time.Now()
	eng.Shutdown(ctx)
	elapsed := time.Since(start)

	// Should complete in ~1s (context timeout), not 60s (hook sleep).
	assert.Less(t, elapsed, 5*time.Second)
	// Service should still be cleaned up despite hung hook.
	statuses := eng.Status(nil)
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
	require.NoError(t, eng.Start(nil))

	eng.Shutdown(context.Background())

	_, err := os.Stat(hookMarker)
	require.NoError(t, err, "post-stop hook should have run")

	statuses := eng.Status(nil)
	for _, s := range statuses {
		assert.Equal(t, "stopped", s.Status)
	}
}
