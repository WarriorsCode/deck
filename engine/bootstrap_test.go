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

func TestBootstrapStepEnvLiteral(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "result")
	steps := []config.BootstrapStep{
		{Name: "Env test", Check: "false", Run: "echo $MY_VAR > " + marker, Env: map[string]string{"MY_VAR": "hello"}},
	}
	err := RunBootstrap(context.Background(), ".", steps, nil)
	require.NoError(t, err)
	data, err := os.ReadFile(marker)
	require.NoError(t, err)
	assert.Contains(t, string(data), "hello")
}

func TestBootstrapStepEnvInterpolation(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "result")
	steps := []config.BootstrapStep{
		{Name: "Interpolate", Check: "false", Run: "echo $GREETING > " + marker, Env: map[string]string{"GREETING": "$(echo world)"}},
	}
	err := RunBootstrap(context.Background(), ".", steps, nil)
	require.NoError(t, err)
	data, err := os.ReadFile(marker)
	require.NoError(t, err)
	assert.Contains(t, string(data), "world")
}

func TestBootstrapStepEnvAvailableInCheck(t *testing.T) {
	steps := []config.BootstrapStep{
		{Name: "Check sees env", Check: "test $MY_FLAG = yes", Run: "false", Env: map[string]string{"MY_FLAG": "yes"}},
	}
	// Check should pass thanks to env, so run (which would fail) is never called.
	err := RunBootstrap(context.Background(), ".", steps, nil)
	require.NoError(t, err)
}

func TestBootstrapStepEnvOverridesGlobal(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "result")
	globalEnv := []string{"MY_VAR=global"}
	steps := []config.BootstrapStep{
		{Name: "Override", Check: "false", Run: "echo $MY_VAR > " + marker, Env: map[string]string{"MY_VAR": "step"}},
	}
	err := RunBootstrap(context.Background(), ".", steps, globalEnv)
	require.NoError(t, err)
	data, err := os.ReadFile(marker)
	require.NoError(t, err)
	assert.Contains(t, string(data), "step")
}

func TestBootstrapStepEnvFailedInterpolation(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "result")
	steps := []config.BootstrapStep{
		{Name: "Bad cmd", Check: "false", Run: "echo [$MISSING] > " + marker, Env: map[string]string{"MISSING": "$(cat /nonexistent/xxx)"}},
	}
	err := RunBootstrap(context.Background(), ".", steps, nil)
	require.NoError(t, err)
	data, err := os.ReadFile(marker)
	require.NoError(t, err)
	assert.Contains(t, string(data), "[]")
}

func TestBootstrapStepEnvNotVisibleToNextStep(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "result")
	steps := []config.BootstrapStep{
		{Name: "Step 1", Check: "false", Run: "true", Env: map[string]string{"STEP1_VAR": "secret"}},
		{Name: "Step 2", Check: "false", Run: "echo [$STEP1_VAR] > " + marker},
	}
	err := RunBootstrap(context.Background(), ".", steps, nil)
	require.NoError(t, err)
	data, err := os.ReadFile(marker)
	require.NoError(t, err)
	assert.Contains(t, string(data), "[]")
}

func TestBootstrapWithEnvFile(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, "test.env")
	marker := filepath.Join(dir, "result")
	require.NoError(t, os.WriteFile(envFile, []byte("BOOT_VAR=fromfile\n"), 0644))

	steps := []config.BootstrapStep{
		{Name: "Env file", Check: "false", Run: "echo $BOOT_VAR > " + marker, EnvFile: envFile},
	}
	err := RunBootstrap(context.Background(), ".", steps, nil)
	require.NoError(t, err)

	data, err := os.ReadFile(marker)
	require.NoError(t, err)
	assert.Contains(t, string(data), "fromfile")
}

func TestBootstrapEnvFileOverriddenByStepEnv(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, "test.env")
	marker := filepath.Join(dir, "result")
	require.NoError(t, os.WriteFile(envFile, []byte("MY_VAR=file\n"), 0644))

	steps := []config.BootstrapStep{
		{Name: "Override", Check: "false", Run: "echo $MY_VAR > " + marker, EnvFile: envFile, Env: map[string]string{"MY_VAR": "step"}},
	}
	err := RunBootstrap(context.Background(), ".", steps, nil)
	require.NoError(t, err)

	data, err := os.ReadFile(marker)
	require.NoError(t, err)
	assert.Contains(t, string(data), "step")
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
