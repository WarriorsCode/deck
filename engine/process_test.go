package engine

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/warriorscode/deck/config"
)

func TestStartAndStopService(t *testing.T) {
	deckDir := filepath.Join(t.TempDir(), ".deck")
	svc := config.Service{Run: "sleep 60"}
	pm := NewProcessManager(deckDir)

	err := pm.Start("testsvc", svc, nil)
	require.NoError(t, err)

	pidFile := filepath.Join(deckDir, "pids", "testsvc.pid")
	_, err = os.Stat(pidFile)
	require.NoError(t, err)

	statuses := pm.Status()
	require.Len(t, statuses, 1)
	assert.Equal(t, "running", statuses[0].Status)

	err = pm.Stop("testsvc")
	require.NoError(t, err)

	_, err = os.Stat(pidFile)
	require.True(t, os.IsNotExist(err))
}

func TestStopAlreadyDead(t *testing.T) {
	deckDir := filepath.Join(t.TempDir(), ".deck")
	svc := config.Service{Run: "true"}
	pm := NewProcessManager(deckDir)

	err := pm.Start("shortsvc", svc, nil)
	require.NoError(t, err)

	time.Sleep(500 * time.Millisecond)

	err = pm.Stop("shortsvc")
	require.NoError(t, err)
}

func TestStaleDetection(t *testing.T) {
	deckDir := filepath.Join(t.TempDir(), ".deck")
	require.NoError(t, os.MkdirAll(filepath.Join(deckDir, "pids"), 0755))

	pidFile := filepath.Join(deckDir, "pids", "stale.pid")
	require.NoError(t, os.WriteFile(pidFile, []byte("999999"), 0644))

	pm := NewProcessManager(deckDir)
	stale, running := pm.CheckStale()
	assert.Contains(t, stale, "stale")
	assert.Empty(t, running)
}

func TestCleanStale(t *testing.T) {
	deckDir := filepath.Join(t.TempDir(), ".deck")
	require.NoError(t, os.MkdirAll(filepath.Join(deckDir, "pids"), 0755))

	pidFile := filepath.Join(deckDir, "pids", "stale.pid")
	require.NoError(t, os.WriteFile(pidFile, []byte("999999"), 0644))

	pm := NewProcessManager(deckDir)
	pm.CleanStale()

	_, err := os.Stat(pidFile)
	require.True(t, os.IsNotExist(err))
}

func TestStopAll(t *testing.T) {
	deckDir := filepath.Join(t.TempDir(), ".deck")
	pm := NewProcessManager(deckDir)

	err := pm.Start("svc1", config.Service{Run: "sleep 60"}, nil)
	require.NoError(t, err)
	err = pm.Start("svc2", config.Service{Run: "sleep 60"}, nil)
	require.NoError(t, err)

	pm.StopAll()

	entries, _ := os.ReadDir(filepath.Join(deckDir, "pids"))
	assert.Empty(t, entries)
}
