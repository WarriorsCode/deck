package engine

import (
	"context"
	"fmt"
	"os"

	"github.com/warriorscode/deck/config"
)

// DiagEntry represents a single diagnostic item from deck doctor.
type DiagEntry struct {
	Section  string   `json:"section"`
	Name     string   `json:"name"`
	Status   string   `json:"status"` // "ok", "fail", "warn"
	Dir      string   `json:"dir,omitempty"`
	Command  string   `json:"command,omitempty"`
	Env      []string `json:"env,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// Doctor runs diagnostics on the full config without starting anything.
func (e *Engine) Doctor(ctx context.Context) []DiagEntry {
	baseEnv, _ := BuildEnv(e.cfg.Env, "", nil)
	var entries []DiagEntry

	// Deps
	e.cfg.Deps.Each(func(name string, dep config.Dep) {
		d := DiagEntry{Section: "dep", Name: name, Command: dep.Check}
		if CheckShell(ctx, e.dir, dep.Check, baseEnv) {
			d.Status = "ok"
		} else {
			d.Status = "fail"
		}
		entries = append(entries, d)
	})

	// Bootstrap
	for _, step := range e.cfg.Bootstrap {
		d := DiagEntry{Section: "bootstrap", Name: step.Name, Command: step.Run}
		d.Dir = stepDir(e.dir, step.Dir)

		stepEnv := baseEnv
		if step.EnvFile != "" {
			if _, err := os.Stat(step.EnvFile); err != nil {
				d.Warnings = append(d.Warnings, "env_file not found: "+step.EnvFile)
			} else {
				fileEnv, err := ParseEnvFile(step.EnvFile)
				if err != nil {
					d.Warnings = append(d.Warnings, "env_file error: "+err.Error())
				} else {
					stepEnv = MergeSlice(stepEnv, fileEnv)
				}
			}
		}
		resolved := resolveEnvWithWarnings(ctx, d.Dir, step.Env, stepEnv, &d.Warnings)
		stepEnv = MergeSlice(stepEnv, resolved)
		d.Env = userEnvOnly(e.cfg.Env, step.Env, step.EnvFile)

		if CheckShell(ctx, d.Dir, step.Check, stepEnv) {
			d.Status = "ok"
		} else {
			d.Status = "fail"
		}
		entries = append(entries, d)
	}

	// Hooks
	for _, phase := range []struct {
		name  string
		hooks []config.Hook
	}{
		{"pre-start", e.cfg.Hooks.PreStart},
		{"post-stop", e.cfg.Hooks.PostStop},
	} {
		for _, hook := range phase.hooks {
			d := DiagEntry{Section: "hook:" + phase.name, Name: hook.Name, Command: hook.Run}
			d.Dir = stepDir(e.dir, hook.Dir)
			d.Status = "ok"

			if hook.EnvFile != "" {
				if _, err := os.Stat(hook.EnvFile); err != nil {
					d.Warnings = append(d.Warnings, "env_file not found: "+hook.EnvFile)
					d.Status = "warn"
				}
			}
			d.Env = userEnvOnly(e.cfg.Env, hook.Env, hook.EnvFile)
			entries = append(entries, d)
		}
	}

	// Services
	e.cfg.Services.Each(func(name string, svc config.Service) {
		d := DiagEntry{Section: "service", Name: name, Command: svc.Run}
		d.Dir = stepDir(e.dir, svc.Dir)
		d.Status = "ok"

		if svc.EnvFile != "" {
			if _, err := os.Stat(svc.EnvFile); err != nil {
				d.Warnings = append(d.Warnings, "env_file not found: "+svc.EnvFile)
				d.Status = "warn"
			}
		}

		svcEnv := baseEnv
		if svc.EnvFile != "" {
			if fileEnv, err := ParseEnvFile(svc.EnvFile); err == nil {
				svcEnv = MergeSlice(svcEnv, fileEnv)
			}
		}
		resolved := resolveEnvWithWarnings(ctx, d.Dir, svc.Env, svcEnv, &d.Warnings)
		if len(resolved) > 0 {
			_ = resolved
		}
		if len(d.Warnings) > 0 && d.Status == "ok" {
			d.Status = "warn"
		}
		d.Env = userEnvOnly(e.cfg.Env, svc.Env, svc.EnvFile)
		entries = append(entries, d)
	})

	return entries
}

// resolveEnvWithWarnings is like ResolveEnv but collects interpolation warnings.
func resolveEnvWithWarnings(ctx context.Context, dir string, raw config.Env, baseEnv []string, warnings *[]string) config.Env {
	if len(raw) == 0 {
		return nil
	}
	resolved := make(config.Env, len(raw))
	for k, v := range raw {
		if v.IsStatic() {
			resolved[k] = v
			continue
		}
		out := ResolveEnv(ctx, dir, config.Env{k: v}, baseEnv)
		if val, ok := out[k]; ok && val.Value == "" {
			*warnings = append(*warnings, fmt.Sprintf("env resolution returned empty: %s", k))
		}
		if val, ok := out[k]; ok {
			resolved[k] = val
		}
	}
	return resolved
}

// userEnvOnly returns a display-friendly list of user-configured env (global + step + file).
func userEnvOnly(global config.Env, step config.Env, envFile string) []string {
	var out []string
	for k, v := range global {
		out = append(out, k+"="+v.Value)
	}
	if envFile != "" {
		out = append(out, "(env_file: "+envFile+")")
	}
	for k, v := range step {
		out = append(out, k+"="+v.Value)
	}
	return out
}
