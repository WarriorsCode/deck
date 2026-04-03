package engine

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/warriorscode/deck/config"
)

// BuildEnv builds a combined environment from the OS env, global config env,
// an optional env file, and per-step env overrides.
// Precedence (highest wins): step env > env file > global env > OS env.
func BuildEnv(globalEnv config.Env, envFile string, stepEnv config.Env) ([]string, error) {
	env := make(config.Env, len(globalEnv)+len(stepEnv))

	for _, e := range os.Environ() {
		k, v, _ := strings.Cut(e, "=")
		env[k] = config.EnvVar{Value: v}
	}
	env.Merge(globalEnv)

	if envFile != "" {
		fileEnv, err := ParseEnvFile(envFile)
		if err != nil {
			return nil, err
		}
		env.Merge(fileEnv)
	}

	env.Merge(stepEnv)
	return env.ToSlice(), nil
}

// MergeSlice overlays resolved env vars onto a base env slice.
// Overlay values win on conflict.
func MergeSlice(base []string, overlay config.Env) []string {
	if len(overlay) == 0 {
		return base
	}
	merged := make(config.Env, len(base)+len(overlay))
	for _, e := range base {
		k, v, _ := strings.Cut(e, "=")
		merged[k] = config.EnvVar{Value: v}
	}
	merged.Merge(overlay)
	return merged.ToSlice()
}

// ResolveEnv evaluates an env map, handling three modes:
//   - value with $(…): shell interpolation via sh -c
//   - script: explicit shell command evaluation
//   - file: read value from structured file (JSON, YAML, TOML, INI)
//   - plain value: used as-is
//
// If a command/read fails, the value is set to empty string and a warning is logged.
func ResolveEnv(ctx context.Context, dir string, raw config.Env, baseEnv []string) config.Env {
	if len(raw) == 0 {
		return nil
	}
	resolved := make(config.Env, len(raw))
	for k, v := range raw {
		switch {
		case v.File != "":
			val, err := ReadFileValue(v.File)
			if err != nil {
				slog.Warn("env file read failed", "key", k, "error", err)
				resolved[k] = config.EnvVar{}
				continue
			}
			resolved[k] = config.EnvVar{Value: val}

		case v.Script != "":
			cmd := exec.CommandContext(ctx, "sh", "-c", v.Script)
			cmd.Dir = dir
			cmd.Env = baseEnv
			out, err := cmd.Output()
			if err != nil {
				slog.Warn("env script failed", "key", k, "error", err)
				resolved[k] = config.EnvVar{}
				continue
			}
			resolved[k] = config.EnvVar{Value: strings.TrimRight(string(out), "\n")}

		case strings.Contains(v.Value, "$("):
			cmd := exec.CommandContext(ctx, "sh", "-c", `printf '%s' `+v.Value)
			cmd.Dir = dir
			cmd.Env = baseEnv
			out, err := cmd.Output()
			if err != nil {
				slog.Warn("env interpolation failed", "key", k, "error", err)
				resolved[k] = config.EnvVar{}
				continue
			}
			resolved[k] = config.EnvVar{Value: string(out)}

		default:
			resolved[k] = v
		}
	}
	return resolved
}

// ParseEnvFile reads a simple KEY=VALUE env file.
// Supports comments (#), empty lines, and single/double quoted values.
func ParseEnvFile(path string) (config.Env, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening env file %s: %w", path, err)
	}
	defer f.Close()

	env := make(config.Env)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if len(v) >= 2 && ((v[0] == '\'' && v[len(v)-1] == '\'') || (v[0] == '"' && v[len(v)-1] == '"')) {
			v = v[1 : len(v)-1]
		}
		env[k] = config.EnvVar{Value: v}
	}
	return env, scanner.Err()
}
