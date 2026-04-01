package status

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"
	"text/template"

	"github.com/warriorscode/deck/engine"
)

// Format formats service statuses in the requested format.
// Empty string or "table" = table format.
// "json" = JSON array.
// Anything else = Go template applied per entry.
func Format(entries []engine.ServiceStatus, format string) (string, error) {
	switch format {
	case "", "table":
		return formatTable(entries)
	case "json":
		return formatJSON(entries)
	default:
		return formatTemplate(entries, format)
	}
}

func formatTable(entries []engine.ServiceStatus) (string, error) {
	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "SERVICE\tPID\tPORT\tSTATUS")
	for _, e := range entries {
		pid := fmt.Sprintf("%d", e.PID)
		if e.PID == 0 {
			pid = "-"
		}
		port := fmt.Sprintf("%d", e.Port)
		if e.Port == 0 {
			port = "-"
		}
		status := e.Status
		if e.Type == "dep" {
			status += " (dep)"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", e.Name, pid, port, status)
	}
	w.Flush()
	return strings.TrimRight(buf.String(), "\n"), nil
}

func formatJSON(entries []engine.ServiceStatus) (string, error) {
	data, err := json.Marshal(entries)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func formatTemplate(entries []engine.ServiceStatus, tmplStr string) (string, error) {
	tmpl, err := template.New("status").Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("invalid format template: %w", err)
	}
	var buf bytes.Buffer
	for _, e := range entries {
		if err := tmpl.Execute(&buf, e); err != nil {
			return "", err
		}
		buf.WriteString("\n")
	}
	return strings.TrimRight(buf.String(), "\n"), nil
}
