package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const appName = "opsctx"

var ErrNotFound = errors.New("config not found")

type Config struct {
	Projects []Project `yaml:"projects"`
}

type Project struct {
	Code         string `yaml:"code"`
	Name         string `yaml:"name"`
	AWSProfile   string `yaml:"aws_profile"`
	CodexProfile string `yaml:"codex_profile"`
	SSHConfig    string `yaml:"ssh_config"`
}

type State struct {
	CurrentProject string    `yaml:"current_project"`
	SwitchedAt     time.Time `yaml:"switched_at"`
}

type Paths struct {
	ConfigFile string
	StateFile  string
	SSHCurrent string
}

func DefaultPaths() (Paths, error) {
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		return Paths{}, err
	}

	return Paths{
		ConfigFile: filepath.Join(cfgDir, appName, "config.yaml"),
		StateFile:  filepath.Join(cfgDir, appName, "state.yaml"),
		SSHCurrent: filepath.Join(cfgDir, appName, "ssh-current"),
	}, nil
}

func LoadConfig(path string) (Config, error) {
	data, err := os.ReadFile(expandHome(path))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, ErrNotFound
		}
		return Config{}, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	seen := make(map[string]struct{}, len(c.Projects))
	for i, project := range c.Projects {
		if project.Code == "" {
			return fmt.Errorf("projects[%d].code is required", i)
		}
		if _, ok := seen[project.Code]; ok {
			return fmt.Errorf("duplicate project code %q", project.Code)
		}
		seen[project.Code] = struct{}{}
	}
	return nil
}

func (c Config) FindProject(code string) (Project, bool) {
	for _, project := range c.Projects {
		if project.Code == code {
			return project, true
		}
	}
	return Project{}, false
}

func LoadState(path string) (State, error) {
	data, err := os.ReadFile(expandHome(path))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return State{}, ErrNotFound
		}
		return State{}, err
	}

	var state State
	if err := yaml.Unmarshal(data, &state); err != nil {
		return State{}, err
	}
	return state, nil
}

func SaveState(path string, state State) error {
	return writeYAML(path, state, 0o600)
}

func WriteSampleConfig(path string) error {
	cfg := Config{
		Projects: []Project{
			{
				Code:         "core",
				Name:         "Core Platform",
				AWSProfile:   "core-devops",
				CodexProfile: "core",
				SSHConfig:    "~/.ssh/config.d/core",
			},
			{
				Code:         "pay",
				Name:         "Payment",
				AWSProfile:   "payment-devops",
				CodexProfile: "payment",
				SSHConfig:    "~/.ssh/config.d/payment",
			},
		},
	}
	return writeYAML(path, cfg, 0o600)
}

func ExpandPath(path string) string {
	return expandHome(path)
}

func writeYAML(path string, value any, perm os.FileMode) error {
	path = expandHome(path)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	data, err := yaml.Marshal(value)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, perm)
}

func expandHome(path string) string {
	if path == "" || path[0] != '~' {
		return path
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if path == "~" {
		return home
	}
	if len(path) > 1 && os.IsPathSeparator(path[1]) {
		return filepath.Join(home, path[2:])
	}
	return path
}
