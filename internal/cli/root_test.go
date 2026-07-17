package cli

import (
	"bytes"
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
	currentProject, err := currentProjectForPicker(config.Paths{
		StateFile: filepath.Join(dir, "missing-state.yaml"),
	}, config.Config{Projects: []config.Project{{Code: "core"}}})
	if err != nil {
		t.Fatal(err)
	}
	if currentProject != config.UnsetProjectCode {
		t.Fatalf("current project = %q, want %q", currentProject, config.UnsetProjectCode)
	}
}

func TestCurrentProjectForPickerAcceptsUnsetState(t *testing.T) {
	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.yaml")
	if err := config.SaveState(stateFile, config.State{CurrentProject: config.UnsetProjectCode}); err != nil {
		t.Fatal(err)
	}

	currentProject, err := currentProjectForPicker(config.Paths{StateFile: stateFile}, config.Config{
		Projects: []config.Project{{Code: "core"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if currentProject != config.UnsetProjectCode {
		t.Fatalf("current project = %q, want %q", currentProject, config.UnsetProjectCode)
	}
}

func TestCurrentProjectForPickerRejectsUnknownState(t *testing.T) {
	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.yaml")
	if err := config.SaveState(stateFile, config.State{CurrentProject: "missing"}); err != nil {
		t.Fatal(err)
	}

	_, err := currentProjectForPicker(config.Paths{StateFile: stateFile}, config.Config{
		Projects: []config.Project{{Code: "core"}},
	})
	if err == nil || !strings.Contains(err.Error(), `current project "missing" is not in config`) {
		t.Fatalf("expected unknown state error, got %v", err)
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
