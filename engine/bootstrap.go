package engine

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"golang.org/x/term"

	"github.com/warriorscode/deck/config"
)

// RunBootstrap runs each bootstrap step in order. Skips if check passes. Fails fast on error.
func RunBootstrap(ctx context.Context, dir string, steps []config.BootstrapStep, env []string) error {
	for _, step := range steps {
		d := stepDir(dir, step.Dir)
		resolved := ResolveEnv(ctx, d, step.Env, env)
		stepEnv := MergeSlice(env, resolved)
		if CheckShell(ctx, d, step.Check, stepEnv) {
			continue
		}
		if step.Prompt != "" {
			extra, err := handlePrompt(step)
			if err != nil {
				return fmt.Errorf("bootstrap %q: %w", step.Name, err)
			}
			stepEnv = append(stepEnv, extra...)
		}
		if err := RunShell(ctx, d, step.Run, stepEnv); err != nil {
			return fmt.Errorf("bootstrap %q: %w", step.Name, err)
		}
	}
	return nil
}

// handlePrompt displays the prompt, reads multi-line input, writes it to a temp file,
// and returns env vars pointing to the input.
func handlePrompt(step config.BootstrapStep) ([]string, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		slog.Warn("bootstrap prompt skipped (not a terminal)", "step", step.Name)
		return nil, fmt.Errorf("prompt requires an interactive terminal")
	}

	fmt.Fprintf(os.Stderr, "\n%s\n", step.Prompt)

	input, err := readMultiLine(os.Stdin)
	if err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "deck-prompt-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp file: %w", err)
	}
	if _, err := tmpFile.WriteString(input); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return nil, fmt.Errorf("writing temp file: %w", err)
	}
	tmpFile.Close()

	absPath, _ := filepath.Abs(tmpFile.Name())
	return []string{
		"DECK_INPUT=" + input,
		"DECK_INPUT_FILE=" + absPath,
	}, nil
}

// readMultiLine reads lines until an empty line or EOF.
func readMultiLine(r io.Reader) (string, error) {
	scanner := bufio.NewScanner(r)
	var result string
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			break
		}
		if result != "" {
			result += "\n"
		}
		result += line
	}
	return result, scanner.Err()
}

func stepDir(base, override string) string {
	if override != "" {
		return override
	}
	return base
}
