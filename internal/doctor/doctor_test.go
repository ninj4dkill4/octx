package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ninj4dkill4/octx/internal/config"
)

func TestRunWarnsWhenStateMissing(t *testing.T) {
	dir := t.TempDir()
	configFile := writeConfig(t, dir, `
projects:
  - code: core
`)

	report := Run(Options{
		Paths: testPaths(dir, configFile),
		Env:   testEnv(dir),
	})

	assertLevel(t, report, Warn, "state")
	if report.HasErrors() {
		t.Fatalf("doctor should not fail on missing state: %#v", report.Results)
	}
}

func TestRunSkipsDependentChecksWhenConfigMissing(t *testing.T) {
	dir := t.TempDir()
	if err := config.SaveState(filepath.Join(dir, "state.yaml"), config.State{CurrentProject: "old"}); err != nil {
		t.Fatal(err)
	}

	report := Run(Options{
		Paths: testPaths(dir, filepath.Join(dir, "missing-config.yaml")),
		Env:   testEnv(dir),
	})

	assertContains(t, report, Error, "config", "config not found")
	for _, result := range report.Results {
		if result.Check == "state" || result.Check == "ssh" || result.Check == "env" {
			t.Fatalf("dependent check %s should be skipped when config is missing: %#v", result.Check, report.Results)
		}
	}
}

func TestRunErrorsOnMissingSSHConfig(t *testing.T) {
	dir := t.TempDir()
	configFile := writeConfig(t, dir, `
projects:
  - code: core
    ssh_config: `+filepath.Join(dir, "missing-ssh")+`
`)

	report := Run(Options{
		Paths: testPaths(dir, configFile),
		Env:   testEnv(dir),
	})

	assertLevel(t, report, Error, "ssh")
}

func TestRunErrorsWhenStateProjectMissingFromConfig(t *testing.T) {
	dir := t.TempDir()
	configFile := writeConfig(t, dir, `
projects:
  - code: core
`)
	if err := config.SaveState(filepath.Join(dir, "state.yaml"), config.State{CurrentProject: "missing"}); err != nil {
		t.Fatal(err)
	}

	report := Run(Options{
		Paths: testPaths(dir, configFile),
		Env:   testEnv(dir),
	})

	assertContains(t, report, Error, "state", `current project "missing" is not in config`)
}

func TestRunWarnsOnEnvMismatch(t *testing.T) {
	dir := t.TempDir()
	configFile := writeConfig(t, dir, `
projects:
  - code: core
    aws_profile: core-devops
`)
	if err := config.SaveState(filepath.Join(dir, "state.yaml"), config.State{CurrentProject: "core"}); err != nil {
		t.Fatal(err)
	}
	env := testEnv(dir)
	env["OPSCTX_PROJECT"] = "core"
	env["AWS_PROFILE"] = "wrong"

	report := Run(Options{
		Paths: testPaths(dir, configFile),
		Env:   env,
	})

	assertContains(t, report, Warn, "env", `AWS_PROFILE="wrong", want "core-devops"`)
}

func TestRunValidatesExternalProfiles(t *testing.T) {
	dir := t.TempDir()
	codexHome := filepath.Join(dir, "codex")
	if err := os.MkdirAll(codexHome, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(codexHome, "core.config.toml"), []byte("model = \"test\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	configFile := writeConfig(t, dir, `
projects:
  - code: core
    aws_profile: core-devops
    aliyun_profile: core-aliyun
    codex_profile: core
`)
	env := testEnv(dir)
	env["CODEX_HOME"] = codexHome

	report := Run(Options{
		Paths: testPaths(dir, configFile),
		Env:   env,
		LookPath: func(name string) (string, error) {
			return "/usr/bin/" + name, nil
		},
		RunCommand: func(name string, args ...string) (string, error) {
			command := name + " " + strings.Join(args, " ")
			switch command {
			case "aws configure list-profiles":
				return "core-devops\nother\n", nil
			case "aliyun configure list":
				return "Profile      | Credential | Valid | Region | Language\n---------    | ---------- | ----- | ------ | --------\ncore-aliyun * | AK:***     | Valid | cn     | en\n", nil
			default:
				return "", fmt.Errorf("unexpected command %s", command)
			}
		},
	})

	assertContains(t, report, OK, "aws", `profile "core-devops" exists`)
	assertContains(t, report, OK, "aliyun", `profile "core-aliyun" exists`)
	assertContains(t, report, OK, "codex", `profile "core" exists`)
	if report.HasErrors() {
		t.Fatalf("doctor should pass: %#v", report.Results)
	}
}

func TestRunErrorsOnMissingExternalProfiles(t *testing.T) {
	dir := t.TempDir()
	configFile := writeConfig(t, dir, `
projects:
  - code: core
    aws_profile: core-devops
    aliyun_profile: core-aliyun
    codex_profile: core
`)

	report := Run(Options{
		Paths: testPaths(dir, configFile),
		Env:   testEnv(dir),
		LookPath: func(name string) (string, error) {
			return "/usr/bin/" + name, nil
		},
		RunCommand: func(name string, args ...string) (string, error) {
			return "other\n", nil
		},
	})

	assertContains(t, report, Error, "aws", `profile "core-devops" not found`)
	assertContains(t, report, Error, "aliyun", `profile "core-aliyun" not found`)
	assertContains(t, report, Error, "codex", `profile "core" file not found`)
}

func TestParseAliyunProfiles(t *testing.T) {
	profiles := parseAliyunProfiles(`Profile      | Credential            | Valid   | Region           | Language
---------    | ------------------    | ------- | ---------------- | --------
default      | AK:***                | Invalid | eu-central-1     | en
scio-cloud * | OAuth:***             | Valid   | eu-central-1     | en
`)
	for _, profile := range []string{"default", "scio-cloud"} {
		if !profiles[profile] {
			t.Fatalf("missing profile %q in %#v", profile, profiles)
		}
	}
}

func TestNPMWrapperMatchesNativeBinary(t *testing.T) {
	wrapper := "/tmp/prefix/lib/node_modules/@ninj4dkill4/octx/bin/octx.js"
	binary := "/tmp/prefix/lib/node_modules/@ninj4dkill4/octx/node_modules/@ninj4dkill4/octx-linux-x64/bin/octx"
	if !isNPMWrapperForBinary(wrapper, binary) {
		t.Fatalf("expected npm wrapper to match native binary")
	}
}

func testPaths(dir, configFile string) config.Paths {
	return config.Paths{
		ConfigFile: configFile,
		StateFile:  filepath.Join(dir, "state.yaml"),
		SSHCurrent: filepath.Join(dir, "ssh-current"),
	}
}

func testEnv(dir string) map[string]string {
	return map[string]string{
		"HOME": dir,
		"PATH": filepath.Join(dir, "bin"),
	}
}

func writeConfig(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func assertLevel(t *testing.T, report Report, level Level, check string) {
	t.Helper()
	for _, result := range report.Results {
		if result.Level == level && result.Check == check {
			return
		}
	}
	t.Fatalf("missing %s %s in %#v", level, check, report.Results)
}

func assertContains(t *testing.T, report Report, level Level, check, message string) {
	t.Helper()
	for _, result := range report.Results {
		if result.Level == level && result.Check == check && strings.Contains(result.Message, message) {
			return
		}
	}
	t.Fatalf("missing %s %s %q in %#v", level, check, message, report.Results)
}
