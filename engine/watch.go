package engine

import (
	"context"
	"log/slog"
	"time"

	"github.com/warriorscode/deck/config"
)

// Watch monitors running services and restarts those with a restart policy.
// Blocks until ctx is cancelled.
func (e *Engine) Watch(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.checkAndRestart(ctx)
		}
	}
}

func (e *Engine) checkAndRestart(ctx context.Context) {
	e.cfg.Services.Each(func(name string, svc config.Service) {
		if svc.Restart == "" {
			return
		}
		if !e.pm.hasPidFile(name) {
			return // never started or already stopped by user
		}
		if e.pm.isRunning(name) {
			return
		}

		// Process died unexpectedly.
		exitCode := e.pm.lastExitCode(name)
		if svc.Restart == "on-failure" && exitCode == 0 {
			slog.Info("service exited cleanly, not restarting", "service", name)
			return
		}

		slog.Warn("service crashed, restarting", "service", name, "exit_code", exitCode)
		env, err := e.ServiceEnv(svc)
		if err != nil {
			slog.Error("failed to build env for restart", "service", name, "error", err)
			return
		}
		if err := e.pm.Start(name, svc, env); err != nil {
			slog.Error("failed to restart service", "service", name, "error", err)
		}
	})
}
