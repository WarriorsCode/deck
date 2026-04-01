package config

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// ParseWithOverride parses a base config and deep-merges a local override into it.
// Maps merge by key (preserving base key order, appending new keys from local).
// Lists replace entirely. If local is nil, behaves like Parse.
func ParseWithOverride(base, local []byte) (*Config, error) {
	if local == nil {
		return Parse(base)
	}

	var baseDoc, localDoc yaml.Node
	if err := yaml.Unmarshal(base, &baseDoc); err != nil {
		return nil, fmt.Errorf("parsing base config: %w", err)
	}
	if err := yaml.Unmarshal(local, &localDoc); err != nil {
		return nil, fmt.Errorf("parsing local config: %w", err)
	}

	// yaml.Unmarshal wraps in a document node; unwrap to the mapping.
	baseRoot := unwrapDoc(&baseDoc)
	localRoot := unwrapDoc(&localDoc)
	if baseRoot == nil || localRoot == nil {
		return Parse(base)
	}

	mergeNodes(baseRoot, localRoot)

	out, err := yaml.Marshal(&baseDoc)
	if err != nil {
		return nil, fmt.Errorf("re-marshaling merged config: %w", err)
	}
	return Parse(out)
}

func unwrapDoc(n *yaml.Node) *yaml.Node {
	if n.Kind == yaml.DocumentNode && len(n.Content) > 0 {
		return n.Content[0]
	}
	return n
}

type nodePair struct {
	dst, src *yaml.Node
}

// mergeNodes merges src mapping into dst mapping iteratively.
// Maps merge by key (preserving dst order, appending new keys from src).
// Everything else (lists, scalars) in src replaces dst.
func mergeNodes(dst, src *yaml.Node) {
	if dst.Kind != yaml.MappingNode || src.Kind != yaml.MappingNode {
		return
	}

	stack := []nodePair{{dst: dst, src: src}}
	for len(stack) > 0 {
		pair := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		// Iterate src.Content directly to preserve local key order for appends.
		for i := 0; i < len(pair.src.Content); i += 2 {
			key := pair.src.Content[i].Value
			srcValNode := pair.src.Content[i+1]

			dstIdx := mappingKeyIndex(pair.dst, key)
			if dstIdx < 0 {
				pair.dst.Content = append(pair.dst.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Value: key},
					srcValNode,
				)
				continue
			}
			dstValNode := pair.dst.Content[dstIdx+1]

			if dstValNode.Kind == yaml.MappingNode && srcValNode.Kind == yaml.MappingNode {
				stack = append(stack, nodePair{dst: dstValNode, src: srcValNode})
				continue
			}
			pair.dst.Content[dstIdx+1] = srcValNode
		}
	}
}

// mappingKeyIndex returns the Content index of the key node, or -1 if not found.
func mappingKeyIndex(n *yaml.Node, key string) int {
	for i := 0; i < len(n.Content); i += 2 {
		if n.Content[i].Value == key {
			return i
		}
	}
	return -1
}
