package scaffold

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectGoProject(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644))

	stacks := Detect(dir)
	require.Len(t, stacks, 1)
	assert.Equal(t, "go", stacks[0].Name)
	assert.Equal(t, ".", stacks[0].Dir)
}

func TestDetectNodeSubdir(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "webapp")
	require.NoError(t, os.Mkdir(sub, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(sub, "package.json"), []byte("{}"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(sub, "pnpm-lock.yaml"), []byte(""), 0644))

	stacks := Detect(dir)
	require.Len(t, stacks, 1)
	assert.Equal(t, "node", stacks[0].Name)
	assert.Equal(t, "webapp", stacks[0].Dir)
}

func TestDetectMultipleStacks(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644))

	sub := filepath.Join(dir, "frontend")
	require.NoError(t, os.Mkdir(sub, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(sub, "package.json"), []byte("{}"), 0644))

	stacks := Detect(dir)
	require.Len(t, stacks, 2)
	assert.Equal(t, "go", stacks[0].Name)
	assert.Equal(t, "node", stacks[1].Name)
}

func TestDetectEmpty(t *testing.T) {
	dir := t.TempDir()
	stacks := Detect(dir)
	assert.Empty(t, stacks)
}

func TestGenerateGo(t *testing.T) {
	stacks := []Stack{{Name: "go", Dir: "."}}
	out := Generate(stacks, "myapp")
	assert.Contains(t, out, "name: myapp")
	assert.Contains(t, out, "go run .")
	assert.NotContains(t, out, "dir:")
}

func TestGenerateNodePnpm(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "webapp")
	require.NoError(t, os.Mkdir(sub, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(sub, "package.json"), []byte("{}"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(sub, "pnpm-lock.yaml"), []byte(""), 0644))

	stacks := Detect(dir)
	out := Generate(stacks, "myapp")
	assert.Contains(t, out, "pnpm dev")
	assert.Contains(t, out, "pnpm install")
	assert.Contains(t, out, "dir: ./webapp")
}

func TestGenerateDefault(t *testing.T) {
	out := Generate(nil, "empty")
	assert.Contains(t, out, "name: empty")
	assert.Contains(t, out, "replace with your start command")
}

func TestNodePkgManagerDetection(t *testing.T) {
	dir := t.TempDir()
	assert.Equal(t, "npm", detectNodePkgManager(dir))

	require.NoError(t, os.WriteFile(filepath.Join(dir, "yarn.lock"), []byte(""), 0644))
	assert.Equal(t, "yarn", detectNodePkgManager(dir))
}
