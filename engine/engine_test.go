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
		Deps: map[string]config.Dep{
			"fake": {Check: "true", Start: config.StringOrList{"true"}},
		},
		Bootstrap: []config.BootstrapStep{
			{Name: "noop", Check: "true", Run: "true"},
		},
		Hooks: config.Hooks{
			PreStart: []config.Hook{{Name: "marker", Run: "touch " + marker}},
		},
		Services: map[string]config.Service{
			"sleeper": {Run: "sleep 60"},
		},
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
	assert.Empty(t, statuses)
}
