package config

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Env is a string-to-string map of environment variables.
type Env map[string]string

// Merge copies all entries from other into e. Existing keys are overwritten.
func (e Env) Merge(other Env) {
	maps.Copy(e, other)
}

// ToSlice converts to the []string format expected by exec.Cmd.Env.
func (e Env) ToSlice() []string {
	s := make([]string, 0, len(e))
	for k, v := range e {
		s = append(s, k+"="+v)
	}
	return s
}

type Config struct {
	Name      string          `yaml:"name"`
	Env       Env             `yaml:"env"`
	Bootstrap []BootstrapStep `yaml:"bootstrap"`
	Deps      Map[Dep]        `yaml:"deps"`
	Hooks     Hooks           `yaml:"hooks"`
	Services  Map[Service]    `yaml:"services"`
}

type BootstrapStep struct {
	Name    string `yaml:"name"`
	Dir     string `yaml:"dir"`
	Check   string `yaml:"check"`
	Run     string `yaml:"run"`
	Prompt  string `yaml:"prompt"`
	Env     Env    `yaml:"env"`
	EnvFile string `yaml:"env_file"`
}

type Dep struct {
	Check string       `yaml:"check"`
	Start StringOrList `yaml:"start"`
	Stop  StringOrList `yaml:"stop"`
}

type Hooks struct {
	PreStart []Hook `yaml:"pre-start"`
	PostStop []Hook `yaml:"post-stop"`
}

type Hook struct {
	Name    string `yaml:"name"`
	Dir     string `yaml:"dir"`
	Run     string `yaml:"run"`
	EnvFile string `yaml:"env_file"`
	Env     Env    `yaml:"env"`
}

type Service struct {
	Dir       string   `yaml:"dir"`
	Run       string   `yaml:"run"`
	Port      int      `yaml:"port"`
	Color     string   `yaml:"color"`
	Timestamp *bool    `yaml:"timestamp"`
	Env       Env      `yaml:"env"`
	EnvFile   string   `yaml:"env_file"`
	DependsOn []string `yaml:"depends_on"`
	Ready     string   `yaml:"ready"`
}

func (s Service) TimestampEnabled() bool {
	if s.Timestamp == nil {
		return true
	}
	return *s.Timestamp
}

// StringOrList allows YAML fields to be either a single string or a list of strings.
type StringOrList []string

func (s *StringOrList) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		*s = []string{value.Value}
		return nil
	}
	var list []string
	if err := value.Decode(&list); err != nil {
		return err
	}
	*s = list
	return nil
}

func Parse(data []byte) (*Config, error) {
	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	if err := validateHookKeys(data); err != nil {
		return nil, err
	}
	if err := validate(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// LoadFile loads deck.yaml from path, merging deck.local.yaml if it exists in the same directory.
func LoadFile(path string) (*Config, error) {
	base, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", filepath.Base(path), err)
	}
	dir := filepath.Dir(path)
	localPath := filepath.Join(dir, "deck.local.yaml")
	local, err := os.ReadFile(localPath)
	if err != nil {
		if os.IsNotExist(err) {
			return Parse(base)
		}
		return nil, fmt.Errorf("reading deck.local.yaml: %w", err)
	}
	return ParseWithOverride(base, local)
}

func validateHookKeys(data []byte) error {
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return err
	}
	hooksRaw, ok := raw["hooks"]
	if !ok {
		return nil
	}
	hooksMap, ok := hooksRaw.(map[string]any)
	if !ok {
		return nil
	}
	for key := range hooksMap {
		if key != "pre-start" && key != "post-stop" {
			return fmt.Errorf("config: unknown hook %q (valid: pre-start, post-stop)", key)
		}
	}
	return nil
}

// ValidateServiceNames checks that all names exist in the config.
func (c *Config) ValidateServiceNames(names []string) error {
	for _, name := range names {
		if _, ok := c.Services.Get(name); !ok {
			return fmt.Errorf("unknown service %q", name)
		}
	}
	return nil
}

func validate(cfg *Config) error {
	if cfg.Services.Len() == 0 {
		return fmt.Errorf("config: at least one service must be defined")
	}
	var errs []error
	cfg.Services.Each(func(name string, svc Service) {
		if svc.Run == "" {
			errs = append(errs, fmt.Errorf("config: service %q: run is required", name))
		}
	})
	cfg.Deps.Each(func(name string, dep Dep) {
		if dep.Check == "" {
			errs = append(errs, fmt.Errorf("config: dep %q: check is required", name))
		}
		if len(dep.Start) == 0 {
			errs = append(errs, fmt.Errorf("config: dep %q: start is required", name))
		}
	})
	for _, step := range cfg.Bootstrap {
		if step.Check == "" {
			errs = append(errs, fmt.Errorf("config: bootstrap step %q: check is required", step.Name))
		}
	}
	cfg.Services.Each(func(name string, svc Service) {
		for _, dep := range svc.DependsOn {
			if _, ok := cfg.Services.Get(dep); !ok {
				errs = append(errs, fmt.Errorf("config: service %q: depends_on %q: unknown service", name, dep))
			}
		}
	})
	if len(errs) > 0 {
		return errs[0]
	}
	if err := detectCycle(cfg); err != nil {
		return err
	}
	return nil
}

func detectCycle(cfg *Config) error {
	// DFS-based cycle detection on the depends_on graph.
	const (
		unvisited = 0
		visiting  = 1
		visited   = 2
	)
	state := make(map[string]int)
	var path []string

	var visit func(name string) error
	visit = func(name string) error {
		if state[name] == visited {
			return nil
		}
		if state[name] == visiting {
			// Find cycle start in path for a clear error message.
			for i, p := range path {
				if p == name {
					cycle := append(path[i:], name)
					return fmt.Errorf("config: dependency cycle: %s", strings.Join(cycle, " → "))
				}
			}
			return fmt.Errorf("config: dependency cycle involving %q", name)
		}
		state[name] = visiting
		path = append(path, name)
		svc, _ := cfg.Services.Get(name)
		for _, dep := range svc.DependsOn {
			if err := visit(dep); err != nil {
				return err
			}
		}
		path = path[:len(path)-1]
		state[name] = visited
		return nil
	}

	for _, name := range cfg.Services.Keys() {
		if err := visit(name); err != nil {
			return err
		}
	}
	return nil
}
