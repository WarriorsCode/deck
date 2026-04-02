package engine

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunShellSuccess(t *testing.T) {
	err := RunShell(context.Background(), ".", "true", nil)
	require.NoError(t, err)
}

func TestRunShellFailure(t *testing.T) {
	err := RunShell(context.Background(), ".", "false", nil)
	require.Error(t, err)
}

func TestRunShellDir(t *testing.T) {
	dir := t.TempDir()
	err := RunShell(context.Background(), dir, "test $(pwd) = "+dir, nil)
	require.NoError(t, err)
}

func TestCheckShellPass(t *testing.T) {
	ok := CheckShell(context.Background(), ".", "true", nil)
	assert.True(t, ok)
}

func TestCheckShellFail(t *testing.T) {
	ok := CheckShell(context.Background(), ".", "false", nil)
	assert.False(t, ok)
}

func TestRunShellWithEnv(t *testing.T) {
	env := []string{"DECK_TEST_VAR=hello"}
	err := RunShell(context.Background(), ".", "test \"$DECK_TEST_VAR\" = hello", env)
	require.NoError(t, err)
}
