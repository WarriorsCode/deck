package engine

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/warriorscode/deck/config"
)

const shutdownTimeout = 30 * time.Second

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

	baseEnv, err := BuildEnv(e.cfg.Env, "", nil)
	if err != nil {
		return err
	}
	if err := EnsureDeps(ctx, e.dir, e.cfg.Deps, baseEnv); err != nil {
		return err
	}
	if err := RunBootstrap(ctx, e.dir, e.cfg.Bootstrap, baseEnv); err != nil {
		return err
	}
	return RunHooks(ctx, e.dir, e.cfg.Hooks.PreStart, false, e.cfg.Env)
}

// Start launches all services in config-defined order.
// On failure, already-started services are rolled back.
func (e *Engine) Start() error {
	var started []string
	err := e.cfg.Services.EachErr(func(name string, svc config.Service) error {
		env, err := BuildEnv(e.cfg.Env, svc.EnvFile, svc.Env)
		if err != nil {
			return fmt.Errorf("service %q: %w", name, err)
		}
		if err := e.pm.Start(name, svc, env); err != nil {
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
// Hooks are bounded by ctx to prevent wedging on a hung hook.
func (e *Engine) Shutdown(ctx context.Context) {
	RunHooks(ctx, e.dir, e.cfg.Hooks.PostStop, true, e.cfg.Env) //nolint:errcheck
	e.pm.StopAll()
}

// Stop kills all services, then runs post-stop hooks (deck stop ordering).
func (e *Engine) Stop() {
	e.pm.StopAll()
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	RunHooks(ctx, e.dir, e.cfg.Hooks.PostStop, true, e.cfg.Env) //nolint:errcheck
}

// Status returns status of all configured services, merging live PID data.
func (e *Engine) Status() []ServiceStatus {
	live := e.pm.Status()
	liveByName := make(map[string]ServiceStatus, len(live))
	for _, s := range live {
		liveByName[s.Name] = s
	}

	var statuses []ServiceStatus
	e.cfg.Services.Each(func(name string, svc config.Service) {
		s := ServiceStatus{
			Name:    name,
			Port:    svc.Port,
			Status:  "stopped",
			Type:    "service",
			LogPath: filepath.Join(e.pm.logDir, name+".log"),
		}
		if ls, ok := liveByName[name]; ok {
			s.PID = ls.PID
			s.Status = ls.Status
		}
		statuses = append(statuses, s)
	})
	return statuses
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
