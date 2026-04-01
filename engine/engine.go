package engine

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/warriorscode/deck/config"
)

type Engine struct {
	cfg *config.Config
	dir string
	pm  *ProcessManager
}

func New(cfg *config.Config, dir, deckDir string) *Engine {
	return &Engine{cfg: cfg, dir: dir, pm: NewProcessManager(deckDir)}
}

// Preflight runs deps, bootstrap, and pre-start hooks.
func (e *Engine) Preflight(ctx context.Context) error {
	_, running := e.pm.CheckStale()
	if len(running) > 0 {
		return fmt.Errorf("services already running: %v. Run 'deck stop' first", running)
	}
	e.pm.CleanStale()

	if err := EnsureDeps(ctx, e.dir, e.cfg.Deps); err != nil {
		return err
	}
	if err := RunBootstrap(ctx, e.dir, e.cfg.Bootstrap); err != nil {
		return err
	}
	return RunHooks(ctx, e.dir, e.cfg.Hooks.PreStart, false)
}

// Start launches all services in config-defined order.
// On failure, already-started services are rolled back.
func (e *Engine) Start() error {
	var started []string
	err := e.cfg.Services.EachErr(func(name string, svc config.Service) error {
		if err := e.pm.Start(name, svc); err != nil {
			return err
		}
		started = append(started, name)
		return nil
	})
	if err != nil {
		for _, name := range started {
			e.pm.Stop(name) //nolint:errcheck
		}
		return err
	}
	return nil
}

// Shutdown runs post-stop hooks first, then kills services (deck up ordering per spec).
func (e *Engine) Shutdown() {
	RunHooks(context.Background(), e.dir, e.cfg.Hooks.PostStop, true) //nolint:errcheck
	e.pm.StopAll()
}

// Stop kills all services, then runs post-stop hooks (deck stop ordering).
func (e *Engine) Stop() {
	e.pm.StopAll()
	RunHooks(context.Background(), e.dir, e.cfg.Hooks.PostStop, true) //nolint:errcheck
}

// Status returns status of all managed services.
func (e *Engine) Status() []ServiceStatus {
	return e.pm.Status()
}

// LogConfigs returns log configurations for all services in config-defined order.
func (e *Engine) LogConfigs() map[string]LogConfig {
	configs := make(map[string]LogConfig, e.cfg.Services.Len())
	i := 0
	e.cfg.Services.Each(func(name string, svc config.Service) {
		color := svc.Color
		if color == "" && i < len(defaultPalette) {
			color = defaultPalette[i]
		}
		configs[name] = LogConfig{
			Path:      filepath.Join(e.pm.logDir, name+".log"),
			Color:     color,
			Timestamp: svc.TimestampEnabled(),
		}
		i++
	})
	return configs
}
