package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"
)

const appName = "opsctx"

const UnsetProjectCode = "unset"

var ErrNotFound = errors.New("config not found")

type Config struct {
	Projects []Project `yaml:"projects"`
}

type Project struct {
	Code           string `yaml:"code"`
	Name           string `yaml:"name"`
	Color          string `yaml:"color"`
	AWSProfile     string `yaml:"aws_profile"`
	CodexProfile   string `yaml:"codex_profile"`
	AliyunProfile  string `yaml:"aliyun_profile"`
	GCloudConfig   string `yaml:"gcloud_config"`
	AzureConfigDir string `yaml:"azure_config_dir"`
	Kubeconfig     string `yaml:"kubeconfig"`
	SSHConfig      string `yaml:"ssh_config"`
}

type Paths struct {
	ConfigFile string
	SSHDir     string
	SSHCurrent string
}

func DefaultPaths() (Paths, error) {
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		return Paths{}, err
	}

	return Paths{
		ConfigFile: filepath.Join(cfgDir, appName, "config.yaml"),
		SSHDir:     filepath.Join(cfgDir, appName, "ssh"),
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
		if project.Color != "" {
			if _, _, _, ok := ParseHexColor(project.Color); !ok {
				return fmt.Errorf("projects[%d].color must be #RGB or #RRGGBB", i)
			}
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

func WriteSampleConfig(path string) error {
	return writeFile(path, []byte(`projects:
- code: core
  name: Core Platform
  color: "#22c55e"
  aws_profile: core-devops
  aliyun_profile: core-devops
  codex_profile: core
  gcloud_config: core-devops
  azure_config_dir: ~/.azure/core
  kubeconfig: ~/.kube/core
  ssh_config: ~/.ssh/config.d/core

- code: pay
  name: Payment
  color: "#3b82f6"
  aws_profile: payment-devops
  aliyun_profile: payment-devops
  codex_profile: payment
  gcloud_config: payment-devops
  azure_config_dir: ~/.azure/payment
  kubeconfig: ~/.kube/payment
  ssh_config: ~/.ssh/config.d/payment
`), 0o600)
}

func ExpandPath(path string) string {
	return expandHome(path)
}

func ParseHexColor(value string) (uint8, uint8, uint8, bool) {
	if len(value) == 4 && value[0] == '#' {
		r, ok := parseHexByte(string([]byte{value[1], value[1]}))
		if !ok {
			return 0, 0, 0, false
		}
		g, ok := parseHexByte(string([]byte{value[2], value[2]}))
		if !ok {
			return 0, 0, 0, false
		}
		b, ok := parseHexByte(string([]byte{value[3], value[3]}))
		if !ok {
			return 0, 0, 0, false
		}
		return r, g, b, true
	}
	if len(value) == 7 && value[0] == '#' {
		r, ok := parseHexByte(value[1:3])
		if !ok {
			return 0, 0, 0, false
		}
		g, ok := parseHexByte(value[3:5])
		if !ok {
			return 0, 0, 0, false
		}
		b, ok := parseHexByte(value[5:7])
		if !ok {
			return 0, 0, 0, false
		}
		return r, g, b, true
	}
	return 0, 0, 0, false
}

func parseHexByte(value string) (uint8, bool) {
	parsed, err := strconv.ParseUint(value, 16, 8)
	if err != nil {
		return 0, false
	}
	return uint8(parsed), true
}

func writeFile(path string, data []byte, perm os.FileMode) error {
	path = expandHome(path)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
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
