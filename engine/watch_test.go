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

func TestWatchRestartsOnCrash(t *testing.T) {
	dir := t.TempDir()
	deckDir := filepath.Join(dir, ".deck")
	counter := filepath.Join(dir, "counter")

	// Service increments a counter file then exits (crashes).
	cfg := &config.Config{
		Name: "watch-test",
		Services: config.MapOf[config.Service](
			"crasher", config.Service{
				Run:     "echo run >> " + counter + " && sleep 0.1 && exit 1",
				Restart: "always",
			},
		),
	}

	eng := New(cfg, dir, deckDir)
	require.NoError(t, eng.Start(nil))

	ctx, cancel := context.WithCancel(context.Background())
	go eng.Watch(ctx)

	// Wait for at least 2 restarts (initial run + 2 restarts = 3 runs).
	time.Sleep(8 * time.Second)
	cancel()
	eng.Stop()

	data, err := os.ReadFile(counter)
	require.NoError(t, err)
	lines := len(splitNonEmpty(string(data)))
	assert.GreaterOrEqual(t, lines, 3, "expected at least 3 runs (initial + 2 restarts), got %d", lines)
}

func TestWatchNoRestartByDefault(t *testing.T) {
	dir := t.TempDir()
	deckDir := filepath.Join(dir, ".deck")
	counter := filepath.Join(dir, "counter")

	cfg := &config.Config{
		Name: "no-restart-test",
		Services: config.MapOf[config.Service](
			"exiter", config.Service{
				Run: "echo run >> " + counter + " && exit 1",
			},
		),
	}

	eng := New(cfg, dir, deckDir)
	require.NoError(t, eng.Start(nil))

	ctx, cancel := context.WithCancel(context.Background())
	go eng.Watch(ctx)

	time.Sleep(5 * time.Second)
	cancel()
	eng.Stop()

	data, err := os.ReadFile(counter)
	require.NoError(t, err)
	lines := len(splitNonEmpty(string(data)))
	assert.Equal(t, 1, lines, "service should only run once without restart policy")
}

func TestWatchOnFailureSkipsCleanExit(t *testing.T) {
	dir := t.TempDir()
	deckDir := filepath.Join(dir, ".deck")
	counter := filepath.Join(dir, "counter")

	cfg := &config.Config{
		Name: "on-failure-test",
		Services: config.MapOf[config.Service](
			"clean", config.Service{
				Run:     "echo run >> " + counter + " && exit 0",
				Restart: "on-failure",
			},
		),
	}

	eng := New(cfg, dir, deckDir)
	require.NoError(t, eng.Start(nil))

	ctx, cancel := context.WithCancel(context.Background())
	go eng.Watch(ctx)

	time.Sleep(5 * time.Second)
	cancel()
	eng.Stop()

	data, err := os.ReadFile(counter)
	require.NoError(t, err)
	lines := len(splitNonEmpty(string(data)))
	assert.Equal(t, 1, lines, "on-failure should not restart on clean exit")
}

func splitNonEmpty(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		if strings.TrimSpace(line) != "" {
			out = append(out, line)
		}
	}
	return out
}
