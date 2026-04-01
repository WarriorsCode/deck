package engine

import (
	"context"
	"fmt"

	"github.com/warriorscode/deck/config"
)

// RunBootstrap runs each bootstrap step in order. Skips if check passes. Fails fast on error.
func RunBootstrap(ctx context.Context, dir string, steps []config.BootstrapStep) error {
	for _, step := range steps {
		if CheckShell(ctx, dir, step.Check) {
			continue
		}
		if err := RunShell(ctx, dir, step.Run); err != nil {
			return fmt.Errorf("bootstrap %q: %w", step.Name, err)
		}
	}
	return nil
}
