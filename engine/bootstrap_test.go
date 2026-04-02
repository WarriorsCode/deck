package engine

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestReadMultiLine(t *testing.T) {
	input := "line one\nline two\nline three\n\nignored"
	result, err := readMultiLine(strings.NewReader(input))
	require.NoError(t, err)
	assert.Equal(t, "line one\nline two\nline three", result)
}

func TestReadMultiLineEOF(t *testing.T) {
	input := "single line"
	result, err := readMultiLine(strings.NewReader(input))
	require.NoError(t, err)
	assert.Equal(t, "single line", result)
}

func TestReadMultiLineEmpty(t *testing.T) {
	result, err := readMultiLine(strings.NewReader("\n"))
	require.NoError(t, err)
	assert.Equal(t, "", result)
}

func TestBootstrapPromptSkipsInNonTTY(t *testing.T) {
	steps := []config.BootstrapStep{
		{Name: "Needs input", Check: "false", Prompt: "Paste key:", Run: "true"},
	}
	// CI/test stdin is not a TTY, so prompt should fail gracefully.
	err := RunBootstrap(context.Background(), ".", steps, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "interactive terminal")
}
