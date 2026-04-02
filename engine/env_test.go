package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/warriorscode/deck/config"
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

	globalEnv := config.Env{"A": "from_global", "B": "from_global", "C": "from_global"}
	stepEnv := config.Env{"A": "from_step"}

	result, err := BuildEnv(globalEnv, envFile, stepEnv)
	require.NoError(t, err)

	envMap := toMap(result)
	assert.Equal(t, "from_step", envMap["A"])
	assert.Equal(t, "from_file", envMap["B"])
	assert.Equal(t, "from_global", envMap["C"])
}

func toMap(envSlice []string) map[string]string {
	m := make(map[string]string, len(envSlice))
	for _, e := range envSlice {
		if k, v, ok := cutString(e, "="); ok {
			m[k] = v
		}
	}
	return m
}

func cutString(s, sep string) (string, string, bool) {
	for i := range len(s) {
		if s[i:i+len(sep)] == sep {
			return s[:i], s[i+len(sep):], true
		}
	}
	return s, "", false
}

func TestBuildEnvNoFile(t *testing.T) {
	globalEnv := config.Env{"FOO": "bar"}
	result, err := BuildEnv(globalEnv, "", nil)
	require.NoError(t, err)
	assert.Equal(t, "bar", toMap(result)["FOO"])
}

func TestBuildEnvNilEverything(t *testing.T) {
	result, err := BuildEnv(nil, "", nil)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestResolveEnvLiteral(t *testing.T) {
	resolved := ResolveEnv(config.Env{"FOO": "bar", "BAZ": "qux"}, nil)
	assert.Equal(t, "bar", resolved["FOO"])
	assert.Equal(t, "qux", resolved["BAZ"])
}

func TestResolveEnvInterpolation(t *testing.T) {
	resolved := ResolveEnv(config.Env{"GREETING": "$(echo hello)"}, nil)
	assert.Equal(t, "hello", resolved["GREETING"])
}

func TestResolveEnvFailedCommand(t *testing.T) {
	resolved := ResolveEnv(config.Env{"MISSING": "$(cat /nonexistent/file/xxx)"}, nil)
	assert.Equal(t, "", resolved["MISSING"])
}

func TestResolveEnvNil(t *testing.T) {
	assert.Nil(t, ResolveEnv(nil, nil))
}

func TestMergeSlice(t *testing.T) {
	base := []string{"A=1", "B=2"}
	step := config.Env{"B": "override", "C": "3"}
	merged := MergeSlice(base, step)

	m := toMap(merged)
	assert.Equal(t, "1", m["A"])
	assert.Equal(t, "override", m["B"])
	assert.Equal(t, "3", m["C"])
}

func TestMergeSliceEmpty(t *testing.T) {
	base := []string{"A=1"}
	assert.Equal(t, base, MergeSlice(base, nil))
}
