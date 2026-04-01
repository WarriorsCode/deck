package engine

import (
	"context"
	"fmt"

	"github.com/warriorscode/deck/config"
)

// RunHooks executes hooks in order.
// If bestEffort is true, errors are logged but execution continues.
// If bestEffort is false, first error stops execution.
func RunHooks(ctx context.Context, dir string, hooks []config.Hook, bestEffort bool) error {
	for _, hook := range hooks {
		if err := RunShell(ctx, dir, hook.Run); err != nil {
			if bestEffort {
				continue
			}
			return fmt.Errorf("hook %q: %w", hook.Name, err)
		}
	}
	return nil
}
