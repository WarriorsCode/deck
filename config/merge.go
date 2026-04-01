package config

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// ParseWithOverride parses a base config and deep-merges a local override into it.
// Maps merge by key, lists replace entirely. If local is nil, behaves like Parse.
func ParseWithOverride(base, local []byte) (*Config, error) {
	if local == nil {
		return Parse(base)
	}

	var baseRaw, localRaw map[string]interface{}
	if err := yaml.Unmarshal(base, &baseRaw); err != nil {
		return nil, fmt.Errorf("parsing base config: %w", err)
	}
	if err := yaml.Unmarshal(local, &localRaw); err != nil {
		return nil, fmt.Errorf("parsing local config: %w", err)
	}

	merged := deepMerge(baseRaw, localRaw)

	out, err := yaml.Marshal(merged)
	if err != nil {
		return nil, fmt.Errorf("re-marshaling merged config: %w", err)
	}
	return Parse(out)
}

// deepMerge merges src into dst. Maps merge recursively by key.
// Everything else (lists, scalars) in src replaces dst.
func deepMerge(dst, src map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{})
	for k, v := range dst {
		out[k] = v
	}
	for k, v := range src {
		dstVal, exists := out[k]
		if !exists {
			out[k] = v
			continue
		}
		srcMap, srcIsMap := v.(map[string]interface{})
		dstMap, dstIsMap := dstVal.(map[string]interface{})
		if srcIsMap && dstIsMap {
			out[k] = deepMerge(dstMap, srcMap)
		} else {
			out[k] = v
		}
	}
	return out
}
