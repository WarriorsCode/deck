package engine

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/warriorscode/deck/config"
)

func TestDoctorDepStatus(t *testing.T) {
	dir := t.TempDir()
	deckDir := filepath.Join(dir, ".deck")
	cfg := &config.Config{
		Name: "doctor-test",
		Deps: config.MapOf[config.Dep](
			"ok-dep", config.Dep{Check: "true", Start: config.StringOrList{"true"}},
			"bad-dep", config.Dep{Check: "false", Start: config.StringOrList{"true"}},
		),
		Services: config.MapOf[config.Service]("svc", config.Service{Run: "sleep 60"}),
	}
	eng := New(cfg, dir, deckDir)
	entries := eng.Doctor(context.Background())

	deps := filterSection(entries, "dep")
	require.Len(t, deps, 2)
	assert.Equal(t, "ok", deps[0].Status)
	assert.Equal(t, "fail", deps[1].Status)
}

func TestDoctorBootstrapStatus(t *testing.T) {
	dir := t.TempDir()
	deckDir := filepath.Join(dir, ".deck")
	cfg := &config.Config{
		Name: "doctor-test",
		Bootstrap: []config.BootstrapStep{
			{Name: "done", Check: "true", Run: "true"},
			{Name: "needed", Check: "false", Run: "echo setup"},
		},
		Services: config.MapOf[config.Service]("svc", config.Service{Run: "sleep 60"}),
	}
	eng := New(cfg, dir, deckDir)
	entries := eng.Doctor(context.Background())

	bs := filterSection(entries, "bootstrap")
	require.Len(t, bs, 2)
	assert.Equal(t, "ok", bs[0].Status)
	assert.Equal(t, "fail", bs[1].Status)
}

func TestDoctorMissingEnvFile(t *testing.T) {
	dir := t.TempDir()
	deckDir := filepath.Join(dir, ".deck")
	cfg := &config.Config{
		Name: "doctor-test",
		Services: config.MapOf[config.Service](
			"svc", config.Service{Run: "sleep 60", EnvFile: "/nonexistent/file.env"},
		),
	}
	eng := New(cfg, dir, deckDir)
	entries := eng.Doctor(context.Background())

	svcs := filterSection(entries, "service")
	require.Len(t, svcs, 1)
	assert.Equal(t, "warn", svcs[0].Status)
	require.Len(t, svcs[0].Warnings, 1)
	assert.Contains(t, svcs[0].Warnings[0], "env_file not found")
}

func filterSection(entries []DiagEntry, section string) []DiagEntry {
	var out []DiagEntry
	for _, e := range entries {
		if e.Section == section {
			out = append(out, e)
		}
	}
	return out
}
