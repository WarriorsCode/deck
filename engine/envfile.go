package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

// ReadFileValue reads a value from a structured file using a dot-separated key path.
// The spec format is "filepath | key.path" where the file format is detected from extension.
// Supported: .json, .yaml, .yml, .toml, .conf, .ini
func ReadFileValue(spec string) (string, error) {
	parts := strings.SplitN(spec, "|", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("env file spec must be 'path | key.path', got %q", spec)
	}
	path := strings.TrimSpace(parts[0])
	keyPath := strings.TrimSpace(parts[1])

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", path, err)
	}

	ext := strings.ToLower(filepath.Ext(path))
	var obj map[string]any

	switch ext {
	case ".json":
		if err := json.Unmarshal(data, &obj); err != nil {
			return "", fmt.Errorf("parsing %s as JSON: %w", path, err)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &obj); err != nil {
			return "", fmt.Errorf("parsing %s as YAML: %w", path, err)
		}
	case ".toml":
		if err := toml.Unmarshal(data, &obj); err != nil {
			return "", fmt.Errorf("parsing %s as TOML: %w", path, err)
		}
	case ".conf", ".ini":
		obj = parseINI(data)
	default:
		return "", fmt.Errorf("unsupported file extension %q (supported: .json, .yaml, .yml, .toml, .conf, .ini)", ext)
	}

	return lookupPath(obj, keyPath)
}

// parseINI parses a simple INI/conf file into a nested map.
// [section] headers create nested maps; key = value pairs are stored as strings.
func parseINI(data []byte) map[string]any {
	result := make(map[string]any)
	var section map[string]any

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			name := strings.TrimSpace(line[1 : len(line)-1])
			section = make(map[string]any)
			result[name] = section
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		// Strip quotes
		if len(v) >= 2 && ((v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'')) {
			v = v[1 : len(v)-1]
		}
		if section != nil {
			section[k] = v
		} else {
			result[k] = v
		}
	}
	return result
}

// lookupPath traverses a nested map using a dot-separated path.
func lookupPath(obj map[string]any, path string) (string, error) {
	keys := strings.Split(path, ".")
	var current any = obj

	for _, key := range keys {
		m, ok := current.(map[string]any)
		if !ok {
			return "", fmt.Errorf("key %q: not a map", key)
		}
		current, ok = m[key]
		if !ok {
			return "", fmt.Errorf("key %q not found", key)
		}
	}

	return fmt.Sprintf("%v", current), nil
}
