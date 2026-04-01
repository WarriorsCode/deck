package engine

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunShellSuccess(t *testing.T) {
	err := RunShell(context.Background(), ".", "true")
	require.NoError(t, err)
}

func TestRunShellFailure(t *testing.T) {
	err := RunShell(context.Background(), ".", "false")
	require.Error(t, err)
}

func TestRunShellDir(t *testing.T) {
	dir := t.TempDir()
	err := RunShell(context.Background(), dir, "test $(pwd) = "+dir)
	require.NoError(t, err)
}

func TestCheckShellPass(t *testing.T) {
	ok := CheckShell(context.Background(), ".", "true")
	assert.True(t, ok)
}

func TestCheckShellFail(t *testing.T) {
	ok := CheckShell(context.Background(), ".", "false")
	assert.False(t, ok)
}
