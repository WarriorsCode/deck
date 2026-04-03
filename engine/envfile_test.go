package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadFileValueJSON(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "config.json")
	require.NoError(t, os.WriteFile(f, []byte(`{"db": {"host": "localhost", "port": 5432}}`), 0644))

	val, err := ReadFileValue(f + " | db.host")
	require.NoError(t, err)
	assert.Equal(t, "localhost", val)

	val, err = ReadFileValue(f + " | db.port")
	require.NoError(t, err)
	assert.Equal(t, "5432", val)
}

func TestReadFileValueYAML(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(f, []byte("db:\n  host: 127.0.0.1\n  user: admin\n"), 0644))

	val, err := ReadFileValue(f + " | db.host")
	require.NoError(t, err)
	assert.Equal(t, "127.0.0.1", val)

	val, err = ReadFileValue(f + " | db.user")
	require.NoError(t, err)
	assert.Equal(t, "admin", val)
}

func TestReadFileValueTOML(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(f, []byte("[db]\nhost = \"pg.local\"\nport = 5432\n"), 0644))

	val, err := ReadFileValue(f + " | db.host")
	require.NoError(t, err)
	assert.Equal(t, "pg.local", val)
}

func TestReadFileValueINI(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "app.conf")
	require.NoError(t, os.WriteFile(f, []byte("[db]\nhost = 127.0.0.1\nusername = postgres\npassword = secret\n"), 0644))

	val, err := ReadFileValue(f + " | db.host")
	require.NoError(t, err)
	assert.Equal(t, "127.0.0.1", val)

	val, err = ReadFileValue(f + " | db.username")
	require.NoError(t, err)
	assert.Equal(t, "postgres", val)
}

func TestReadFileValueINIQuoted(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "app.conf")
	require.NoError(t, os.WriteFile(f, []byte("[db]\nhost = \"localhost\"\n"), 0644))

	val, err := ReadFileValue(f + " | db.host")
	require.NoError(t, err)
	assert.Equal(t, "localhost", val)
}

func TestReadFileValueMissingKey(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "config.json")
	require.NoError(t, os.WriteFile(f, []byte(`{"db": {"host": "localhost"}}`), 0644))

	_, err := ReadFileValue(f + " | db.port")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestReadFileValueMissingFile(t *testing.T) {
	_, err := ReadFileValue("/nonexistent/file.json | db.host")
	require.Error(t, err)
}

func TestReadFileValueNoSeparator(t *testing.T) {
	_, err := ReadFileValue("/some/file.json")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "path | key.path")
}
