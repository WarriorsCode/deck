package engine

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHasTimestamp(t *testing.T) {
	tests := []struct {
		line string
		want bool
	}{
		{"2026-04-01T18:30:00Z something happened", true},
		{"2026-04-01 18:30:00 something happened", true},
		{"time=2026-04-01T18:30:00Z level=INFO msg=hello", true},
		{"just a plain log line", false},
		{"ERROR: something broke", false},
		{"18:30:00 short time", true},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, HasTimestamp(tt.line), "line: %s", tt.line)
	}
}

func TestFormatLogLine(t *testing.T) {
	line := "server started"
	result := FormatLogLine("api", line, true)
	assert.Contains(t, result, "[api]")
	assert.Contains(t, result, "server started")
	assert.Contains(t, result, time.Now().Format("2006-01-02"))
}

func TestFormatLogLineNoTimestamp(t *testing.T) {
	line := "server started"
	result := FormatLogLine("api", line, false)
	assert.Contains(t, result, "[api]")
	assert.Contains(t, result, "server started")
	assert.NotContains(t, result, time.Now().Format("2006-01-02 15:04"))
}

func TestFormatLogLineExistingTimestamp(t *testing.T) {
	line := "2026-04-01T18:30:00Z server started"
	result := FormatLogLine("api", line, true)
	assert.Contains(t, result, "[api]")
	assert.Contains(t, result, "2026-04-01T18:30:00Z server started")
}

func TestStripANSI(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"\033[32mhello\033[0m", "hello"},
		{"\033[42m\033[30m INFO \033[0m \033[32mconnected\033[0m", " INFO  connected"},
		{"\033[2m[02:40:26]\033[0m starting", "[02:40:26] starting"},
		{"no ansi here", "no ansi here"},
		{"", ""},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, StripANSI(tt.input), "input: %q", tt.input)
	}
}

func TestFormatLogLineStripsANSI(t *testing.T) {
	line := "\033[32mserver started\033[0m"
	result := FormatLogLine("api", line, false)
	assert.Contains(t, result, "server started")
	assert.NotContains(t, result, "\033[32m")
}

func TestTailLogs(t *testing.T) {
	dir := t.TempDir()
	logFile := filepath.Join(dir, "api.log")
	require.NoError(t, os.WriteFile(logFile, []byte("line1\n"), 0644))

	var buf bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	services := map[string]LogConfig{
		"api": {Path: logFile, Color: "cyan", Timestamp: true},
	}

	go TailLogs(ctx, services, &buf)

	time.Sleep(200 * time.Millisecond)
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_WRONLY, 0644)
	require.NoError(t, err)
	_, err = f.WriteString("line2\n")
	require.NoError(t, err)
	f.Close()

	<-ctx.Done()
	time.Sleep(100 * time.Millisecond)

	output := buf.String()
	assert.Contains(t, output, "[api]")
	assert.Contains(t, output, "line2")
}
