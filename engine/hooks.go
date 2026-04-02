package engine

import (
	"context"
	"fmt"

	"github.com/warriorscode/deck/config"
)

// RunHooks executes hooks in order.
// If bestEffort is true, errors are logged but execution continues.
// If bestEffort is false, first error stops execution.
// globalEnv is the base env; each hook's EnvFile is layered on top.
func RunHooks(ctx context.Context, dir string, hooks []config.Hook, bestEffort bool, globalEnv map[string]string) error {
	for _, hook := range hooks {
		env, err := BuildEnv(globalEnv, hook.EnvFile, nil)
		if err != nil {
			if bestEffort {
				continue
			}
			return fmt.Errorf("hook %q: %w", hook.Name, err)
		}
		d := stepDir(dir, hook.Dir)
		if err := RunShell(ctx, d, hook.Run, env); err != nil {
			if bestEffort {
				continue
			}
			return fmt.Errorf("hook %q: %w", hook.Name, err)
		}
	}
	return nil
}
