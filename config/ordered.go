package config

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// Map preserves YAML key insertion order while providing O(1) lookup.
type Map[V any] struct {
	keys   []string
	values map[string]V
}

func NewMap[V any](cap int) Map[V] {
	return Map[V]{
		keys:   make([]string, 0, cap),
		values: make(map[string]V, cap),
	}
}

func (m *Map[V]) Set(key string, val V) {
	if _, exists := m.values[key]; !exists {
		m.keys = append(m.keys, key)
	}
	m.values[key] = val
}

func (m *Map[V]) Get(key string) (V, bool) {
	v, ok := m.values[key]
	return v, ok
}

func (m *Map[V]) Len() int {
	return len(m.keys)
}

// Keys returns keys in insertion order.
func (m *Map[V]) Keys() []string {
	return m.keys
}

// Each calls fn for each entry in insertion order.
func (m *Map[V]) Each(fn func(key string, val V)) {
	for _, k := range m.keys {
		fn(k, m.values[k])
	}
}

// EachErr calls fn for each entry in insertion order, stopping on the first error.
func (m *Map[V]) EachErr(fn func(key string, val V) error) error {
	for _, k := range m.keys {
		if err := fn(k, m.values[k]); err != nil {
			return err
		}
	}
	return nil
}

// MapOf builds an Map from key-value pairs for test convenience.
func MapOf[V any](pairs ...any) Map[V] {
	m := NewMap[V](len(pairs) / 2)
	for i := 0; i < len(pairs); i += 2 {
		m.Set(pairs[i].(string), pairs[i+1].(V))
	}
	return m
}

func (m *Map[V]) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("expected mapping, got %v", node.Kind)
	}
	m.keys = make([]string, 0, len(node.Content)/2)
	m.values = make(map[string]V, len(node.Content)/2)
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valNode := node.Content[i+1]
		var val V
		if err := valNode.Decode(&val); err != nil {
			return fmt.Errorf("decoding value for key %q: %w", keyNode.Value, err)
		}
		m.keys = append(m.keys, keyNode.Value)
		m.values[keyNode.Value] = val
	}
	return nil
}

func (m Map[V]) MarshalYAML() (any, error) {
	node := &yaml.Node{Kind: yaml.MappingNode}
	for _, k := range m.keys {
		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: k}
		valNode := &yaml.Node{}
		data, err := yaml.Marshal(m.values[k])
		if err != nil {
			return nil, err
		}
		if err := yaml.Unmarshal(data, valNode); err != nil {
			return nil, err
		}
		node.Content = append(node.Content, keyNode, valNode)
	}
	return node, nil
}
