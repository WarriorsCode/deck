package config

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Name      string             `yaml:"name"`
	Bootstrap []BootstrapStep    `yaml:"bootstrap"`
	Deps      map[string]Dep     `yaml:"deps"`
	Hooks     Hooks              `yaml:"hooks"`
	Services  map[string]Service `yaml:"services"`
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
	Name string `yaml:"name"`
	Run  string `yaml:"run"`
}

type Service struct {
	Dir       string `yaml:"dir"`
	Run       string `yaml:"run"`
	Port      int    `yaml:"port"`
	Color     string `yaml:"color"`
	Timestamp *bool  `yaml:"timestamp"`
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

func validateHookKeys(data []byte) error {
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return err
	}
	hooksRaw, ok := raw["hooks"]
	if !ok {
		return nil
	}
	hooksMap, ok := hooksRaw.(map[string]interface{})
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
	if len(cfg.Services) == 0 {
		return fmt.Errorf("config: at least one service must be defined")
	}
	for name, svc := range cfg.Services {
		if svc.Run == "" {
			return fmt.Errorf("config: service %q: run is required", name)
		}
	}
	for name, dep := range cfg.Deps {
		if dep.Check == "" {
			return fmt.Errorf("config: dep %q: check is required", name)
		}
		if len(dep.Start) == 0 {
			return fmt.Errorf("config: dep %q: start is required", name)
		}
	}
	for _, step := range cfg.Bootstrap {
		if step.Check == "" {
			return fmt.Errorf("config: bootstrap step %q: check is required", step.Name)
		}
	}
	return nil
}
