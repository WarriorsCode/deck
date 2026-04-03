package engine

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/warriorscode/deck/config"
)

const shutdownTimeout = 30 * time.Second

func toSet(names []string) map[string]bool {
	if len(names) == 0 {
		return nil
	}
	s := make(map[string]bool, len(names))
	for _, n := range names {
		s[n] = true
	}
	return s
}

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

// ServiceEnv builds the resolved environment for a service.
func (e *Engine) ServiceEnv(svc config.Service) ([]string, error) {
	baseEnv, err := BuildEnv(e.cfg.Env, svc.EnvFile, nil)
	if err != nil {
		return nil, err
	}
	svcDir := stepDir(e.dir, svc.Dir)
	resolved := ResolveEnv(context.Background(), svcDir, svc.Env, baseEnv)
	return MergeSlice(baseEnv, resolved), nil
}

// Start launches services in config-defined order.
// If filter is non-empty, only matching services are started.
// On failure, already-started services are rolled back.
func (e *Engine) Start(filter []string) error {
	allowed := toSet(filter)
	var started []string
	err := e.cfg.Services.EachErr(func(name string, svc config.Service) error {
		if len(allowed) > 0 && !allowed[name] {
			return nil
		}
		env, err := e.ServiceEnv(svc)
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

// StopServices stops only the named services. No hooks are run.
func (e *Engine) StopServices(names []string) {
	for _, name := range names {
		e.pm.Stop(name) //nolint:errcheck
	}
}

// Status returns status of configured services, merging live PID data.
// If filter is non-empty, only matching services are returned.
func (e *Engine) Status(filter []string) []ServiceStatus {
	allowed := toSet(filter)
	live := e.pm.Status()
	liveByName := make(map[string]ServiceStatus, len(live))
	for _, s := range live {
		liveByName[s.Name] = s
	}

	var statuses []ServiceStatus
	e.cfg.Services.Each(func(name string, svc config.Service) {
		if len(allowed) > 0 && !allowed[name] {
			return
		}
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

// LogConfigs returns log configurations for services in config-defined order.
// If filter is non-empty, only matching services are returned.
func (e *Engine) LogConfigs(filter []string) map[string]LogConfig {
	allowed := toSet(filter)
	configs := make(map[string]LogConfig, e.cfg.Services.Len())
	i := 0
	e.cfg.Services.Each(func(name string, svc config.Service) {
		color := svc.Color
		if color == "" && i < len(defaultPalette) {
			color = defaultPalette[i]
		}
		if len(allowed) == 0 || allowed[name] {
			configs[name] = LogConfig{
				Path:      filepath.Join(e.pm.logDir, name+".log"),
				Color:     color,
				Timestamp: svc.TimestampEnabled(),
			}
		}
		i++
	})
	return configs
}
