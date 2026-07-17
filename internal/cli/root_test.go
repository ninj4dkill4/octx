package cli

import (
	"bytes"
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

func TestCurrentProjectForPickerDefaultsToUnsetWithoutShellEnv(t *testing.T) {
	currentProject := currentProjectForPicker(config.Config{Projects: []config.Project{{Code: "core"}}}, "")
	if currentProject != config.UnsetProjectCode {
		t.Fatalf("current project = %q, want %q", currentProject, config.UnsetProjectCode)
	}
}

func TestCurrentProjectForPickerPrefersShellEnv(t *testing.T) {
	currentProject := currentProjectForPicker(config.Config{
		Projects: []config.Project{{Code: "core"}, {Code: "pay"}},
	}, "pay")
	if currentProject != "pay" {
		t.Fatalf("current project = %q, want pay", currentProject)
	}
}

func TestCurrentProjectForPickerDefaultsUnknownShellEnvToUnset(t *testing.T) {
	currentProject := currentProjectForPicker(config.Config{
		Projects: []config.Project{{Code: "core"}},
	}, "missing")
	if currentProject != config.UnsetProjectCode {
		t.Fatalf("current project = %q, want %q", currentProject, config.UnsetProjectCode)
	}
}

func TestCurrentCommandReadsShellEnv(t *testing.T) {
	t.Setenv("OPSCTX_PROJECT", "core")

	cmd := NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"current"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out.String()) != "core" {
		t.Fatalf("current output = %q, want core", out.String())
	}
}

func TestCurrentCommandWithoutShellEnv(t *testing.T) {
	t.Setenv("OPSCTX_PROJECT", "")

	cmd := NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"current"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "No current project") {
		t.Fatalf("current output = %q, want no current project", out.String())
	}
}

func TestWriteDoctorReportGroupsByProject(t *testing.T) {
	report := doctor.Report{Results: []doctor.Result{
		{Level: doctor.OK, Check: "config", Message: "loaded config"},
		{Level: doctor.OK, Project: "core", Color: "#22c55e", Check: "aws", Message: `profile "core" exists`},
		{Level: doctor.Warn, Project: "pay", Check: "kube", Message: "kubeconfig missing"},
		{Level: doctor.OK, Check: "binary", Message: "running octx"},
		{Level: doctor.OK, Project: "core", Color: "#22c55e", Check: "ssh", Message: "ssh_config exists"},
	}}

	var out bytes.Buffer
	writeDoctorReport(&out, report)
	output := out.String()

	for _, want := range []string{
		"[global]\n",
		"OK    config   loaded config\n",
		"OK    binary   running octx\n",
		"[\x1b[38;2;34;197;94mcore\x1b[0m]\n",
		"OK    aws      profile \"core\" exists\n",
		"OK    ssh      ssh_config exists\n",
		"[pay]\n",
		"WARN  kube     kubeconfig missing\n",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("missing %q in output:\n%s", want, output)
		}
	}

	if strings.Index(output, "[global]") > strings.Index(output, "core") {
		t.Fatalf("global section should be first:\n%s", output)
	}
	if strings.Index(output, "core") > strings.Index(output, "[pay]") {
		t.Fatalf("project sections should keep first-seen order:\n%s", output)
	}
}
