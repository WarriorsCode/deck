package engine

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/warriorscode/deck/config"
)

func TestBootstrapSkipsIfCheckPasses(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "should-not-exist")
	steps := []config.BootstrapStep{
		{Name: "Skip me", Check: "true", Run: "touch " + marker},
	}
	err := RunBootstrap(context.Background(), ".", steps, nil)
	require.NoError(t, err)
	_, err = os.Stat(marker)
	require.True(t, os.IsNotExist(err))
}

func TestBootstrapRunsIfCheckFails(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "created")
	steps := []config.BootstrapStep{
		{Name: "Create file", Check: "test -f " + marker, Run: "touch " + marker},
	}
	err := RunBootstrap(context.Background(), ".", steps, nil)
	require.NoError(t, err)
	_, err = os.Stat(marker)
	require.NoError(t, err)
}

func TestBootstrapFailFast(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "should-not-exist")
	steps := []config.BootstrapStep{
		{Name: "Fail", Check: "false", Run: "false"},
		{Name: "Never runs", Check: "false", Run: "touch " + marker},
	}
	err := RunBootstrap(context.Background(), ".", steps, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Fail")
	_, err = os.Stat(marker)
	require.True(t, os.IsNotExist(err))
}
