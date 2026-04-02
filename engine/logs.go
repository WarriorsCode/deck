package engine

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"
)

var colorCodes = map[string]string{
	"cyan":    "\033[36m",
	"magenta": "\033[35m",
	"yellow":  "\033[33m",
	"green":   "\033[32m",
	"blue":    "\033[34m",
	"red":     "\033[31m",
}

const colorReset = "\033[0m"

var defaultPalette = []string{"cyan", "magenta", "yellow", "green", "blue", "red"}

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)

var timestampPatterns = []*regexp.Regexp{
	regexp.MustCompile(`^\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}`),
	regexp.MustCompile(`^\d{2}:\d{2}:\d{2}`),
	regexp.MustCompile(`time=\d{4}-\d{2}-\d{2}`),
}

type LogConfig struct {
	Path      string
	Color     string
	Timestamp bool
}

func HasTimestamp(line string) bool {
	for _, p := range timestampPatterns {
		if p.MatchString(line) {
			return true
		}
	}
	return false
}

func FormatLogLine(name, line string, injectTimestamp bool) string {
	return FormatLogLineWithColor(name, line, "cyan", injectTimestamp)
}

func StripANSI(s string) string {
	return ansiPattern.ReplaceAllString(s, "")
}

func FormatLogLineWithColor(name, line, colorName string, injectTimestamp bool) string {
	line = StripANSI(line)
	var sb strings.Builder
	code, ok := colorCodes[colorName]
	if !ok {
		code = colorCodes["cyan"]
	}
	sb.WriteString(code)
	sb.WriteString("[" + name + "]")
	sb.WriteString(colorReset)
	sb.WriteString("  ")
	if injectTimestamp && !HasTimestamp(line) {
		sb.WriteString(time.Now().Format("2006-01-02 15:04:05"))
		sb.WriteString(" | ")
	}
	sb.WriteString(line)
	return sb.String()
}

// TailLogs tails all log files and writes formatted output to w.
func TailLogs(ctx context.Context, services map[string]LogConfig, w io.Writer) {
	for name, cfg := range services {
		go tailFile(ctx, name, cfg, w)
	}
	<-ctx.Done()
}

const tailLines = 20

// lastLines reads the last n lines from the current position of f,
// then leaves the file offset at EOF for subsequent tailing.
func lastLines(f *os.File, n int) []string {
	scanner := bufio.NewScanner(f)
	ring := make([]string, 0, n)
	for scanner.Scan() {
		if len(ring) >= n {
			ring = append(ring[:0], ring[1:]...)
		}
		ring = append(ring, scanner.Text())
	}
	return ring
}

func tailFile(ctx context.Context, name string, cfg LogConfig, w io.Writer) {
	for {
		f, err := os.Open(cfg.Path)
		if err != nil {
			select {
			case <-ctx.Done():
				return
			case <-time.After(500 * time.Millisecond):
				continue
			}
		}

		// Print last N lines before tailing.
		for _, line := range lastLines(f, tailLines) {
			formatted := FormatLogLineWithColor(name, line, cfg.Color, cfg.Timestamp)
			fmt.Fprintln(w, formatted)
		}

		scanner := bufio.NewScanner(f)
		for {
			select {
			case <-ctx.Done():
				f.Close()
				return
			default:
			}
			if scanner.Scan() {
				line := scanner.Text()
				formatted := FormatLogLineWithColor(name, line, cfg.Color, cfg.Timestamp)
				fmt.Fprintln(w, formatted)
				continue
			}
			time.Sleep(100 * time.Millisecond)
			scanner = bufio.NewScanner(f)
		}
	}
}
