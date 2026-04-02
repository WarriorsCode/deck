package engine

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// BuildEnv builds a combined environment from the OS env, global config env,
// an optional env file, and per-step env overrides.
// Precedence (highest wins): step env > env file > global env > OS env.
func BuildEnv(globalEnv map[string]string, envFile string, stepEnv map[string]string) ([]string, error) {
	env := make(map[string]string, len(globalEnv)+len(stepEnv))

	// Start with OS environment.
	for _, e := range os.Environ() {
		k, v, _ := strings.Cut(e, "=")
		env[k] = v
	}

	// Layer global config env.
	for k, v := range globalEnv {
		env[k] = v
	}

	// Layer env file if specified.
	if envFile != "" {
		fileEnv, err := ParseEnvFile(envFile)
		if err != nil {
			return nil, err
		}
		for k, v := range fileEnv {
			env[k] = v
		}
	}

	// Layer step-level env (highest priority).
	for k, v := range stepEnv {
		env[k] = v
	}

	result := make([]string, 0, len(env))
	for k, v := range env {
		result = append(result, k+"="+v)
	}
	return result, nil
}

// ParseEnvFile reads a simple KEY=VALUE env file.
// Supports comments (#), empty lines, and single/double quoted values.
func ParseEnvFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening env file %s: %w", path, err)
	}
	defer f.Close()

	env := make(map[string]string)
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
		// Strip surrounding quotes.
		if len(v) >= 2 && ((v[0] == '\'' && v[len(v)-1] == '\'') || (v[0] == '"' && v[len(v)-1] == '"')) {
			v = v[1 : len(v)-1]
		}
		env[k] = v
	}
	return env, scanner.Err()
}
