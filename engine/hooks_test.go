package engine

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/warriorscode/deck/config"
)

func TestHooksRunInOrder(t *testing.T) {
	dir := t.TempDir()
	fileA := filepath.Join(dir, "a")
	fileB := filepath.Join(dir, "b")
	hooks := []config.Hook{
		{Name: "Create A", Run: "touch " + fileA},
		{Name: "Create B", Run: "touch " + fileB},
	}
	err := RunHooks(context.Background(), ".", hooks, false)
	require.NoError(t, err)
	_, err = os.Stat(fileA)
	require.NoError(t, err)
	_, err = os.Stat(fileB)
	require.NoError(t, err)
}

func TestHooksFailFast(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "should-not-exist")
	hooks := []config.Hook{
		{Name: "Fail", Run: "false"},
		{Name: "Never runs", Run: "touch " + marker},
	}
	err := RunHooks(context.Background(), ".", hooks, false)
	require.Error(t, err)
	_, err = os.Stat(marker)
	require.True(t, os.IsNotExist(err))
}

func TestHooksBestEffort(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "created")
	hooks := []config.Hook{
		{Name: "Fail", Run: "false"},
		{Name: "Still runs", Run: "touch " + marker},
	}
	err := RunHooks(context.Background(), ".", hooks, true)
	require.NoError(t, err)
	_, err = os.Stat(marker)
	require.NoError(t, err)
}
