package switcher

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ninj4dkill4/octx/internal/config"
)

func TestSwitchGeneratesProjectSSHConfigWithoutState(t *testing.T) {
	dir := t.TempDir()
	sshConfig := filepath.Join(dir, "project-ssh")
	if err := os.WriteFile(sshConfig, []byte("Host core\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	configFile := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(configFile, []byte(`
projects:
  - code: core
    name: Core Platform
    aws_profile: core-devops
    ssh_config: `+sshConfig+`
`), 0o600); err != nil {
		t.Fatal(err)
	}

	stateFile := filepath.Join(dir, "state.yaml")
	result, err := Switch("core", Options{
		ConfigFile: configFile,
		SSHDir:     filepath.Join(dir, "ssh"),
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.Project.Code != "core" {
		t.Fatalf("project code = %q, want core", result.Project.Code)
	}
	if result.SSHConfig == "" {
		t.Fatal("missing generated ssh config path")
	}
	data, err := os.ReadFile(result.SSHConfig)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Include ~/.ssh/config", "Include " + sshConfig} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("generated ssh config missing %q:\n%s", want, data)
		}
	}
	if _, err := os.Stat(stateFile); !os.IsNotExist(err) {
		t.Fatalf("switch should not write state file: %v", err)
	}
}

func TestSwitchKeepsProjectSSHConfigsIsolated(t *testing.T) {
	dir := t.TempDir()
	coreSSH := filepath.Join(dir, "core-ssh")
	paySSH := filepath.Join(dir, "pay-ssh")
	if err := os.WriteFile(coreSSH, []byte("Host core\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paySSH, []byte("Host pay\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	configFile := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(configFile, []byte(`
projects:
  - code: core
    ssh_config: `+coreSSH+`
  - code: pay
    ssh_config: `+paySSH+`
`), 0o600); err != nil {
		t.Fatal(err)
	}

	sshDir := filepath.Join(dir, "ssh")
	core, err := Switch("core", Options{ConfigFile: configFile, SSHDir: sshDir})
	if err != nil {
		t.Fatal(err)
	}
	pay, err := Switch("pay", Options{ConfigFile: configFile, SSHDir: sshDir})
	if err != nil {
		t.Fatal(err)
	}

	if core.SSHConfig == pay.SSHConfig {
		t.Fatalf("generated ssh configs should be isolated, got %s", core.SSHConfig)
	}
	coreData, err := os.ReadFile(core.SSHConfig)
	if err != nil {
		t.Fatal(err)
	}
	payData, err := os.ReadFile(pay.SSHConfig)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(coreData), coreSSH) || !strings.Contains(string(payData), paySSH) {
		t.Fatalf("generated configs point to wrong files:\ncore=%s\npay=%s", coreData, payData)
	}
}

func TestShellExports(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	output := ShellExports(config.Project{
		Code:           "core",
		AWSProfile:     "core-devops",
		CodexProfile:   "core",
		AliyunProfile:  "core-aliyun",
		GCloudConfig:   "core-gcp",
		AzureConfigDir: "~/.azure/core",
		Kubeconfig:     "~/.kube/core",
	}, filepath.Join(home, ".config", "opsctx", "ssh", "core.config"))
	for _, want := range []string{
		"export OPSCTX_PROJECT='core'",
		"export AWS_PROFILE='core-devops'",
		"export ALIBABA_CLOUD_PROFILE='core-aliyun'",
		"export KUBECONFIG='" + filepath.Join(home, ".kube", "core") + "'",
		"export CLOUDSDK_ACTIVE_CONFIG_NAME='core-gcp'",
		"export AZURE_CONFIG_DIR='" + filepath.Join(home, ".azure", "core") + "'",
		"export OCTX_SSH_CONFIG='" + filepath.Join(home, ".config", "opsctx", "ssh", "core.config") + "'",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("missing %q in shell exports: %s", want, output)
		}
	}
}

func TestShellExportsUnsetOptionalProfiles(t *testing.T) {
	output := ShellExports(config.Project{
		Code: "no-profiles",
	}, "")
	for _, want := range []string{
		"export OPSCTX_PROJECT='no-profiles'",
		"unset AWS_PROFILE",
		"unset CODEX_PROFILE",
		"unset ALIBABA_CLOUD_PROFILE",
		"unset CLOUDSDK_ACTIVE_CONFIG_NAME",
		"unset AZURE_CONFIG_DIR",
		"unset KUBECONFIG",
		"unset OCTX_SSH_CONFIG",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("missing %q in shell exports: %s", want, output)
		}
	}
}

func TestShellUnsetAll(t *testing.T) {
	output := ShellUnsetAll()
	for _, want := range []string{
		"unset OPSCTX_PROJECT",
		"unset AWS_PROFILE",
		"unset CODEX_PROFILE",
		"unset ALIBABA_CLOUD_PROFILE",
		"unset CLOUDSDK_ACTIVE_CONFIG_NAME",
		"unset AZURE_CONFIG_DIR",
		"unset KUBECONFIG",
		"unset OCTX_SSH_CONFIG",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("missing %q in shell unset output: %s", want, output)
		}
	}
}
