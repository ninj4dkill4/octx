package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ninj4dkill4/octx/internal/config"
	"github.com/ninj4dkill4/octx/internal/doctor"
)

func TestVersionCommand(t *testing.T) {
	cmd := NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"version"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if out.String() == "" {
		t.Fatal("version output is empty")
	}
}

func TestCurrentProjectForPickerDefaultsMissingStateToUnset(t *testing.T) {
	dir := t.TempDir()
	pickerState, err := currentProjectForPicker(config.Paths{
		StateFile: filepath.Join(dir, "missing-state.yaml"),
	}, config.Config{Projects: []config.Project{{Code: "core"}}})
	if err != nil {
		t.Fatal(err)
	}
	if pickerState.currentProject != config.UnsetProjectCode {
		t.Fatalf("current project = %q, want %q", pickerState.currentProject, config.UnsetProjectCode)
	}
}

func TestCurrentProjectForPickerAcceptsUnsetState(t *testing.T) {
	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.yaml")
	if err := config.SaveState(stateFile, config.State{CurrentProject: config.UnsetProjectCode}); err != nil {
		t.Fatal(err)
	}

	pickerState, err := currentProjectForPicker(config.Paths{StateFile: stateFile}, config.Config{
		Projects: []config.Project{{Code: "core"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if pickerState.currentProject != config.UnsetProjectCode {
		t.Fatalf("current project = %q, want %q", pickerState.currentProject, config.UnsetProjectCode)
	}
}

func TestCurrentProjectForPickerResetsUnknownStateToUnset(t *testing.T) {
	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.yaml")
	sshCurrent := filepath.Join(dir, "ssh-current")
	sshTarget := filepath.Join(dir, "ssh-target")
	if err := config.SaveState(stateFile, config.State{CurrentProject: "missing"}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sshTarget, []byte("Host core\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(sshTarget, sshCurrent); err != nil {
		t.Fatal(err)
	}

	pickerState, err := currentProjectForPicker(config.Paths{StateFile: stateFile, SSHCurrent: sshCurrent}, config.Config{
		Projects: []config.Project{{Code: "core"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if pickerState.currentProject != config.UnsetProjectCode {
		t.Fatalf("current project = %q, want %q", pickerState.currentProject, config.UnsetProjectCode)
	}
	if !pickerState.reset {
		t.Fatal("unknown state should mark picker state as reset")
	}
	state, err := config.LoadState(stateFile)
	if err != nil {
		t.Fatal(err)
	}
	if state.CurrentProject != config.UnsetProjectCode {
		t.Fatalf("state current project = %q, want %q", state.CurrentProject, config.UnsetProjectCode)
	}
	if _, err := os.Lstat(sshCurrent); !os.IsNotExist(err) {
		t.Fatalf("ssh current still exists after resetting unknown state: %v", err)
	}
}

func TestRootShellResetsUnknownStateAndPrintsUnsetExports(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.yaml")
	stateFile := filepath.Join(dir, "state.yaml")
	sshCurrent := filepath.Join(dir, "ssh-current")
	sshTarget := filepath.Join(dir, "ssh-target")

	if err := os.WriteFile(configFile, []byte("projects:\n  - code: core\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := config.SaveState(stateFile, config.State{CurrentProject: "missing"}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sshTarget, []byte("Host core\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(sshTarget, sshCurrent); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{
		"--config", configFile,
		"--state", stateFile,
		"--ssh-current", sshCurrent,
		"--shell",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	output := out.String()
	for _, want := range []string{
		"unset OPSCTX_PROJECT",
		"unset AWS_PROFILE",
		"unset CODEX_PROFILE",
		"unset ALIBABA_CLOUD_PROFILE",
		"unset CLOUDSDK_ACTIVE_CONFIG_NAME",
		"unset AZURE_CONFIG_DIR",
		"unset KUBECONFIG",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("missing %q in shell output:\n%s", want, output)
		}
	}
	state, err := config.LoadState(stateFile)
	if err != nil {
		t.Fatal(err)
	}
	if state.CurrentProject != config.UnsetProjectCode {
		t.Fatalf("state current project = %q, want %q", state.CurrentProject, config.UnsetProjectCode)
	}
	if _, err := os.Lstat(sshCurrent); !os.IsNotExist(err) {
		t.Fatalf("ssh current still exists after reset: %v", err)
	}
}

func TestWriteDoctorReportGroupsByProject(t *testing.T) {
	report := doctor.Report{Results: []doctor.Result{
		{Level: doctor.OK, Check: "config", Message: "loaded config"},
		{Level: doctor.OK, Project: "core", Check: "aws", Message: `profile "core" exists`},
		{Level: doctor.Warn, Project: "pay", Check: "kube", Message: "kubeconfig missing"},
		{Level: doctor.OK, Check: "binary", Message: "running octx"},
		{Level: doctor.OK, Project: "core", Check: "ssh", Message: "ssh_config exists"},
	}}

	var out bytes.Buffer
	writeDoctorReport(&out, report)
	output := out.String()

	for _, want := range []string{
		"[global]\n",
		"OK    config   loaded config\n",
		"OK    binary   running octx\n",
		"[core]\n",
		"OK    aws      profile \"core\" exists\n",
		"OK    ssh      ssh_config exists\n",
		"[pay]\n",
		"WARN  kube     kubeconfig missing\n",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("missing %q in output:\n%s", want, output)
		}
	}

	if strings.Index(output, "[global]") > strings.Index(output, "[core]") {
		t.Fatalf("global section should be first:\n%s", output)
	}
	if strings.Index(output, "[core]") > strings.Index(output, "[pay]") {
		t.Fatalf("project sections should keep first-seen order:\n%s", output)
	}
}
