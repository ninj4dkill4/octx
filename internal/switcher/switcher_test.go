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
		Code:         "core",
		AWSProfile:   "core-devops",
		CodexProfile: "core",
	})
	if !strings.Contains(output, "export AWS_PROFILE='core-devops'") {
		t.Fatalf("missing AWS_PROFILE export: %s", output)
	}
}
