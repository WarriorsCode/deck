package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseEnvFile(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, "test.env")
	content := `# comment
DB_HOST=localhost
DB_PORT=5432

QUOTED_SINGLE='hello world'
QUOTED_DOUBLE="foo bar"
EMPTY=
`
	require.NoError(t, os.WriteFile(envFile, []byte(content), 0644))

	env, err := ParseEnvFile(envFile)
	require.NoError(t, err)
	assert.Equal(t, "localhost", env["DB_HOST"])
	assert.Equal(t, "5432", env["DB_PORT"])
	assert.Equal(t, "hello world", env["QUOTED_SINGLE"])
	assert.Equal(t, "foo bar", env["QUOTED_DOUBLE"])
	assert.Equal(t, "", env["EMPTY"])
}

func TestParseEnvFileNotFound(t *testing.T) {
	_, err := ParseEnvFile("/nonexistent/file.env")
	require.Error(t, err)
}

func TestBuildEnvPrecedence(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, "test.env")
	require.NoError(t, os.WriteFile(envFile, []byte("A=from_file\nB=from_file\n"), 0644))

	globalEnv := map[string]string{"A": "from_global", "B": "from_global", "C": "from_global"}
	stepEnv := map[string]string{"A": "from_step"}

	result, err := BuildEnv(globalEnv, envFile, stepEnv)
	require.NoError(t, err)

	envMap := make(map[string]string, len(result))
	for _, e := range result {
		k, v, _ := cutString(e, "=")
		envMap[k] = v
	}

	// Step env wins over everything.
	assert.Equal(t, "from_step", envMap["A"])
	// Env file wins over global.
	assert.Equal(t, "from_file", envMap["B"])
	// Global fills in the rest.
	assert.Equal(t, "from_global", envMap["C"])
}

func cutString(s, sep string) (string, string, bool) {
	i := 0
	for i < len(s) {
		if s[i:i+len(sep)] == sep {
			return s[:i], s[i+len(sep):], true
		}
		i++
	}
	return s, "", false
}

func TestBuildEnvNoFile(t *testing.T) {
	globalEnv := map[string]string{"FOO": "bar"}
	result, err := BuildEnv(globalEnv, "", nil)
	require.NoError(t, err)

	envMap := make(map[string]string, len(result))
	for _, e := range result {
		k, v, _ := cutString(e, "=")
		envMap[k] = v
	}
	assert.Equal(t, "bar", envMap["FOO"])
}

func TestBuildEnvNilEverything(t *testing.T) {
	result, err := BuildEnv(nil, "", nil)
	require.NoError(t, err)
	// Should at least have OS environment.
	assert.NotEmpty(t, result)
}
