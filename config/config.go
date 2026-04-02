package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Name      string            `yaml:"name"`
	Env       map[string]string `yaml:"env"`
	Bootstrap []BootstrapStep   `yaml:"bootstrap"`
	Deps      Map[Dep]          `yaml:"deps"`
	Hooks     Hooks             `yaml:"hooks"`
	Services  Map[Service]      `yaml:"services"`
}

type BootstrapStep struct {
	Name  string `yaml:"name"`
	Check string `yaml:"check"`
	Run   string `yaml:"run"`
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
	Run     string `yaml:"run"`
	EnvFile string `yaml:"env_file"`
}

type Service struct {
	Dir       string            `yaml:"dir"`
	Run       string            `yaml:"run"`
	Port      int               `yaml:"port"`
	Color     string            `yaml:"color"`
	Timestamp *bool             `yaml:"timestamp"`
	Env       map[string]string `yaml:"env"`
	EnvFile   string            `yaml:"env_file"`
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
	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}
