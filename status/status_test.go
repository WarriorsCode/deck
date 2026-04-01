package status

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/warriorscode/deck/engine"
)

var testEntries = []engine.ServiceStatus{
	{Name: "api", PID: 12345, Port: 4000, Status: "running", Type: "service"},
	{Name: "webapp", PID: 12346, Port: 5173, Status: "running", Type: "service"},
	{Name: "postgres", PID: 0, Port: 5432, Status: "running", Type: "dep"},
}

func TestFormatTable(t *testing.T) {
	out, err := Format(testEntries, "")
	require.NoError(t, err)
	assert.Contains(t, out, "SERVICE")
	assert.Contains(t, out, "api")
	assert.Contains(t, out, "12345")
	assert.Contains(t, out, "running")
	assert.Contains(t, out, "postgres")
	assert.Contains(t, out, "running (dep)")
}

func TestFormatJSON(t *testing.T) {
	out, err := Format(testEntries, "json")
	require.NoError(t, err)
	assert.Contains(t, out, `"name":"api"`)
	assert.Contains(t, out, `"pid":12345`)
}

func TestFormatGoTemplate(t *testing.T) {
	out, err := Format(testEntries, "{{.Name}} {{.Status}}")
	require.NoError(t, err)
	assert.Contains(t, out, "api running")
	assert.Contains(t, out, "webapp running")
}

func TestFormatInvalidTemplate(t *testing.T) {
	_, err := Format(testEntries, "{{.Invalid")
	require.Error(t, err)
}
