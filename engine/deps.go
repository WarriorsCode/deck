package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/warriorscode/deck/config"
)

const strategyTimeout = 10 * time.Second

// EnsureDeps checks each dependency and starts it if needed.
// Tries each start strategy in order until the check passes.
func EnsureDeps(ctx context.Context, dir string, deps config.Map[config.Dep], env []string) error {
	return deps.EachErr(func(name string, dep config.Dep) error {
		if CheckShell(ctx, dir, dep.Check, env) {
			return nil
		}
		return startDep(ctx, dir, name, dep, env)
	})
}

func startDep(ctx context.Context, dir, name string, dep config.Dep, env []string) error {
	for i, strategy := range dep.Start {
		RunShell(ctx, dir, strategy, env) //nolint:errcheck

		timeout := strategyTimeout
		if deadline, ok := ctx.Deadline(); ok {
			remaining := time.Until(deadline)
			if remaining < timeout {
				timeout = remaining
			}
		}

		deadline := time.After(timeout)
		for {
			if CheckShell(ctx, dir, dep.Check, env) {
				return nil
			}
			select {
			case <-ctx.Done():
				return fmt.Errorf("dep %q: context cancelled", name)
			case <-deadline:
				if i < len(dep.Start)-1 {
					goto nextStrategy
				}
				return fmt.Errorf("dep %q: not reachable after trying all %d strategies", name, len(dep.Start))
			case <-time.After(1 * time.Second):
				continue
			}
		}
	nextStrategy:
	}
	return fmt.Errorf("dep %q: no start strategies defined", name)
}
