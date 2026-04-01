package config

import (
	"fmt"
	"maps"

	"gopkg.in/yaml.v3"
)

// ParseWithOverride parses a base config and deep-merges a local override into it.
// Maps merge by key, lists replace entirely. If local is nil, behaves like Parse.
func ParseWithOverride(base, local []byte) (*Config, error) {
	if local == nil {
		return Parse(base)
	}

	var baseRaw, localRaw map[string]any
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

type mergePair struct {
	dst, src map[string]any
}

// deepMerge merges src into dst iteratively. Maps merge by key, everything else replaces.
func deepMerge(dst, src map[string]any) map[string]any {
	root := make(map[string]any, len(dst)+len(src))
	maps.Copy(root, dst)

	stack := []mergePair{{dst: root, src: src}}
	for len(stack) > 0 {
		pair := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		for k, v := range pair.src {
			dstVal, exists := pair.dst[k]
			if !exists {
				pair.dst[k] = v
				continue
			}
			srcMap, srcIsMap := v.(map[string]any)
			dstMap, dstIsMap := dstVal.(map[string]any)
			if srcIsMap && dstIsMap {
				stack = append(stack, mergePair{dst: dstMap, src: srcMap})
				continue
			}
			pair.dst[k] = v
		}
	}
	return root
}
