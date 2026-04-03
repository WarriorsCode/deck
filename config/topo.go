package config

// TopoSort returns service names in dependency order (dependencies first).
// Assumes no cycles (validated at parse time).
func (c *Config) TopoSort() []string {
	visited := make(map[string]bool)
	var order []string

	var visit func(name string)
	visit = func(name string) {
		if visited[name] {
			return
		}
		visited[name] = true
		svc, _ := c.Services.Get(name)
		for _, dep := range svc.DependsOn {
			visit(dep)
		}
		order = append(order, name)
	}

	for _, name := range c.Services.Keys() {
		visit(name)
	}
	return order
}

// ExpandDeps expands a set of service names to include all transitive dependencies.
// Returns names in topological order.
func (c *Config) ExpandDeps(names []string) []string {
	if len(names) == 0 {
		return nil
	}
	wanted := make(map[string]bool)
	var collect func(name string)
	collect = func(name string) {
		if wanted[name] {
			return
		}
		wanted[name] = true
		svc, _ := c.Services.Get(name)
		for _, dep := range svc.DependsOn {
			collect(dep)
		}
	}
	for _, n := range names {
		collect(n)
	}

	// Return in topo order, filtered to wanted set.
	all := c.TopoSort()
	var result []string
	for _, name := range all {
		if wanted[name] {
			result = append(result, name)
		}
	}
	return result
}
