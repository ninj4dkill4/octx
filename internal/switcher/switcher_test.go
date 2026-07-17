package switcher

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ninj4dkill4/octx/internal/config"
)

func TestSwitchWritesStateAndSSHCurrent(t *testing.T) {
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
    codex_profile: core
    aliyun_profile: core-devops
    kubeconfig: ~/.kube/core
    ssh_config: `+sshConfig+`
`), 0o600); err != nil {
		t.Fatal(err)
	}

	stateFile := filepath.Join(dir, "state.yaml")
	sshCurrent := filepath.Join(dir, "ssh-current")
	result, err := Switch("core", Options{
		ConfigFile: configFile,
		StateFile:  stateFile,
		SSHCurrent: sshCurrent,
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.Project.Code != "core" {
		t.Fatalf("project code = %q, want core", result.Project.Code)
	}
	if _, err := os.Stat(stateFile); err != nil {
		t.Fatalf("state file not written: %v", err)
	}
	target, err := os.Readlink(sshCurrent)
	if err != nil {
		t.Fatalf("ssh current is not a symlink: %v", err)
	}
	if target != sshConfig {
		t.Fatalf("ssh current target = %q, want %q", target, sshConfig)
	}
}

func TestShellExports(t *testing.T) {
	output := ShellExports(config.Project{
		Code:          "core",
		AWSProfile:    "core-devops",
		CodexProfile:  "core",
		AliyunProfile: "core-aliyun",
		Kubeconfig:    "~/.kube/core",
	})
	if !strings.Contains(output, "export AWS_PROFILE='core-devops'") {
		t.Fatalf("missing AWS_PROFILE export: %s", output)
	}
	if !strings.Contains(output, "export ALIBABA_CLOUD_PROFILE='core-aliyun'") {
		t.Fatalf("missing ALIBABA_CLOUD_PROFILE export: %s", output)
	}
	if !strings.Contains(output, "export KUBECONFIG='~/.kube/core'") {
		t.Fatalf("missing KUBECONFIG export: %s", output)
	}
}

func TestSwitchClearsOptionalSSHConfig(t *testing.T) {
	dir := t.TempDir()
	oldSSHConfig := filepath.Join(dir, "old-ssh")
	if err := os.WriteFile(oldSSHConfig, []byte("Host old\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	configFile := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(configFile, []byte(`
projects:
  - code: no-ssh
    name: No SSH
`), 0o600); err != nil {
		t.Fatal(err)
	}

	stateFile := filepath.Join(dir, "state.yaml")
	sshCurrent := filepath.Join(dir, "ssh-current")
	if err := os.Symlink(oldSSHConfig, sshCurrent); err != nil {
		t.Fatal(err)
	}

	if _, err := Switch("no-ssh", Options{
		ConfigFile: configFile,
		StateFile:  stateFile,
		SSHCurrent: sshCurrent,
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Lstat(sshCurrent); !os.IsNotExist(err) {
		t.Fatalf("ssh current still exists after switching to project without ssh_config: %v", err)
	}
}

func TestShellExportsUnsetOptionalProfiles(t *testing.T) {
	output := ShellExports(config.Project{
		Code: "no-profiles",
	})
	for _, want := range []string{
		"export OPSCTX_PROJECT='no-profiles'",
		"unset AWS_PROFILE",
		"unset CODEX_PROFILE",
		"unset ALIBABA_CLOUD_PROFILE",
		"unset KUBECONFIG",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("missing %q in shell exports: %s", want, output)
		}
	}
}

func TestClearSavesUnsetStateAndRemovesSSHCurrent(t *testing.T) {
	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.yaml")
	sshCurrent := filepath.Join(dir, "ssh-current")
	sshTarget := filepath.Join(dir, "ssh-target")
	if err := os.WriteFile(stateFile, []byte("current_project: core\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sshTarget, []byte("Host core\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(sshTarget, sshCurrent); err != nil {
		t.Fatal(err)
	}

	if _, err := Clear(Options{
		StateFile:  stateFile,
		SSHCurrent: sshCurrent,
	}); err != nil {
		t.Fatal(err)
	}

	state, err := config.LoadState(stateFile)
	if err != nil {
		t.Fatalf("state file not readable after clear: %v", err)
	}
	if state.CurrentProject != config.UnsetProjectCode {
		t.Fatalf("current project = %q, want %q", state.CurrentProject, config.UnsetProjectCode)
	}
	if _, err := os.Lstat(sshCurrent); !os.IsNotExist(err) {
		t.Fatalf("ssh current still exists after clear: %v", err)
	}
}

func TestShellUnsetAll(t *testing.T) {
	output := ShellUnsetAll()
	for _, want := range []string{
		"unset OPSCTX_PROJECT",
		"unset AWS_PROFILE",
		"unset CODEX_PROFILE",
		"unset ALIBABA_CLOUD_PROFILE",
		"unset KUBECONFIG",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("missing %q in shell unset output: %s", want, output)
		}
	}
}
