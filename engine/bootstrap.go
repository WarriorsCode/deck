package engine

import (
	"context"
	"fmt"

	"github.com/warriorscode/deck/config"
)

// RunBootstrap runs each bootstrap step in order. Skips if check passes. Fails fast on error.
func RunBootstrap(ctx context.Context, dir string, steps []config.BootstrapStep, env []string) error {
	for _, step := range steps {
		d := stepDir(dir, step.Dir)
		if CheckShell(ctx, d, step.Check, env) {
			continue
		}
		if err := RunShell(ctx, d, step.Run, env); err != nil {
			return fmt.Errorf("bootstrap %q: %w", step.Name, err)
		}
	}
	return nil
}

func stepDir(base, override string) string {
	if override != "" {
		return override
	}
	return base
}
