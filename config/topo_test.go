package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTopoSortNoDeps(t *testing.T) {
	cfg := &Config{
		Services: MapOf[Service](
			"a", Service{Run: "true"},
			"b", Service{Run: "true"},
			"c", Service{Run: "true"},
		),
	}
	order := cfg.TopoSort()
	assert.Equal(t, []string{"a", "b", "c"}, order)
}

func TestTopoSortLinearChain(t *testing.T) {
	cfg := &Config{
		Services: MapOf[Service](
			"web", Service{Run: "true", DependsOn: []string{"api"}},
			"api", Service{Run: "true", DependsOn: []string{"db"}},
			"db", Service{Run: "true"},
		),
	}
	order := cfg.TopoSort()
	assert.Equal(t, []string{"db", "api", "web"}, order)
}

func TestTopoSortDiamond(t *testing.T) {
	cfg := &Config{
		Services: MapOf[Service](
			"app", Service{Run: "true", DependsOn: []string{"svc-a", "svc-b"}},
			"svc-a", Service{Run: "true", DependsOn: []string{"db"}},
			"svc-b", Service{Run: "true", DependsOn: []string{"db"}},
			"db", Service{Run: "true"},
		),
	}
	order := cfg.TopoSort()
	// db must come first, app must come last
	assert.Equal(t, "db", order[0])
	assert.Equal(t, "app", order[len(order)-1])
}

func TestExpandDepsTransitive(t *testing.T) {
	cfg := &Config{
		Services: MapOf[Service](
			"web", Service{Run: "true", DependsOn: []string{"api"}},
			"api", Service{Run: "true", DependsOn: []string{"db"}},
			"db", Service{Run: "true"},
			"worker", Service{Run: "true"},
		),
	}
	expanded := cfg.ExpandDeps([]string{"web"})
	assert.Equal(t, []string{"db", "api", "web"}, expanded)
}

func TestExpandDepsEmpty(t *testing.T) {
	cfg := &Config{
		Services: MapOf[Service]("a", Service{Run: "true"}),
	}
	assert.Nil(t, cfg.ExpandDeps(nil))
}

func TestExpandDepsNoDeps(t *testing.T) {
	cfg := &Config{
		Services: MapOf[Service](
			"a", Service{Run: "true"},
			"b", Service{Run: "true"},
		),
	}
	expanded := cfg.ExpandDeps([]string{"a"})
	assert.Equal(t, []string{"a"}, expanded)
}

func TestDetectCycle(t *testing.T) {
	data := []byte(`
services:
  a:
    run: "true"
    depends_on: [b]
  b:
    run: "true"
    depends_on: [a]
`)
	_, err := Parse(data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dependency cycle")
}

func TestDetectSelfCycle(t *testing.T) {
	data := []byte(`
services:
  a:
    run: "true"
    depends_on: [a]
`)
	_, err := Parse(data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dependency cycle")
}

func TestDependsOnUnknownService(t *testing.T) {
	data := []byte(`
services:
  a:
    run: "true"
    depends_on: [nonexistent]
`)
	_, err := Parse(data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown service")
}
