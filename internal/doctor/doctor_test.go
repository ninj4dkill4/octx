package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ninj4dkill4/octx/internal/config"
)

func TestRunReportsUnsetShellContext(t *testing.T) {
	dir := t.TempDir()
	configFile := writeConfig(t, dir, `
projects:
  - code: core
`)

	report := Run(Options{
		Paths: testPaths(dir, configFile),
		Env:   testEnv(dir),
	})

	assertContains(t, report, OK, "shell", "context unset")
	if report.HasErrors() {
		t.Fatalf("doctor should not fail on unset shell context: %#v", report.Results)
	}
}

func TestRunWarnsWhenShellProjectMissingFromConfig(t *testing.T) {
	dir := t.TempDir()
	configFile := writeConfig(t, dir, `
projects:
  - code: core
`)
	env := testEnv(dir)
	env["OPSCTX_PROJECT"] = "missing"

	report := Run(Options{
		Paths: testPaths(dir, configFile),
		Env:   env,
	})

	assertContains(t, report, Warn, "shell", `OPSCTX_PROJECT="missing" is not in config`)
	if report.HasErrors() {
		t.Fatalf("doctor should not fail when shell points to an unknown project: %#v", report.Results)
	}
}

func TestRunSkipsDependentChecksWhenConfigMissing(t *testing.T) {
	dir := t.TempDir()

	report := Run(Options{
		Paths: testPaths(dir, filepath.Join(dir, "missing-config.yaml")),
		Env:   testEnv(dir),
	})

	assertContains(t, report, Error, "config", "config not found")
	for _, result := range report.Results {
		if result.Check == "shell" || result.Check == "ssh" || result.Check == "env" {
			t.Fatalf("dependent check %s should be skipped when config is missing: %#v", result.Check, report.Results)
		}
	}
}

func TestRunWarnsOnMissingSSHConfig(t *testing.T) {
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

	assertProjectContains(t, report, Warn, "core", "ssh", "ssh_config")
}

func TestRunWarnsOnLegacySSHCurrentInclude(t *testing.T) {
	dir := t.TempDir()
	sshConfig := filepath.Join(dir, "core-ssh")
	if err := os.WriteFile(sshConfig, []byte("Host core\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	configFile := writeConfig(t, dir, `
projects:
  - code: core
    ssh_config: `+sshConfig+`
`)
	writeSSHConfig(t, dir, "Include "+filepath.Join(dir, "ssh-current")+"\n")

	report := Run(Options{
		Paths: testPaths(dir, configFile),
		Env:   testEnv(dir),
	})

	assertContains(t, report, Warn, "ssh", "legacy")
	if report.HasErrors() {
		t.Fatalf("legacy ssh-current include should warn, not fail: %#v", report.Results)
	}
}

func TestRunAllowsMissingSSHInclude(t *testing.T) {
	dir := t.TempDir()
	sshConfig := filepath.Join(dir, "core-ssh")
	if err := os.WriteFile(sshConfig, []byte("Host core\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	configFile := writeConfig(t, dir, `
projects:
  - code: core
    ssh_config: `+sshConfig+`
`)
	writeSSHConfig(t, dir, "Host *\n  ServerAliveInterval 30\n")

	report := Run(Options{
		Paths: testPaths(dir, configFile),
		Env:   testEnv(dir),
	})

	assertProjectContains(t, report, OK, "core", "ssh", "ssh_config exists")
	if report.HasErrors() {
		t.Fatalf("missing legacy include should not fail anymore: %#v", report.Results)
	}
}

func TestRunDoesNotRequireSSHIncludeWithoutSSHConfigs(t *testing.T) {
	dir := t.TempDir()
	configFile := writeConfig(t, dir, `
projects:
  - code: core
`)

	report := Run(Options{
		Paths: testPaths(dir, configFile),
		Env:   testEnv(dir),
	})

	for _, result := range report.Results {
		if result.Check == "ssh" && result.Level == Error {
			t.Fatalf("ssh include should not be required without project ssh_config: %#v", report.Results)
		}
	}
}

func TestRunValidatesKubeconfig(t *testing.T) {
	dir := t.TempDir()
	kubeconfig := filepath.Join(dir, "kubeconfig")
	if err := os.WriteFile(kubeconfig, []byte("apiVersion: v1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	configFile := writeConfig(t, dir, `
projects:
  - code: core
    kubeconfig: `+kubeconfig+`
`)

	report := Run(Options{
		Paths: testPaths(dir, configFile),
		Env:   testEnv(dir),
	})

	assertProjectContains(t, report, OK, "core", "kube", "kubeconfig exists")
}

func TestRunWarnsOnMissingKubeconfig(t *testing.T) {
	dir := t.TempDir()
	configFile := writeConfig(t, dir, `
projects:
  - code: core
    kubeconfig: `+filepath.Join(dir, "missing-kubeconfig")+`
`)

	report := Run(Options{
		Paths: testPaths(dir, configFile),
		Env:   testEnv(dir),
	})

	assertLevel(t, report, Warn, "kube")
	if report.HasErrors() {
		t.Fatalf("doctor should not fail on missing optional kubeconfig: %#v", report.Results)
	}
}

func TestRunWarnsOnEnvMismatchWhenShellContextIsSet(t *testing.T) {
	dir := t.TempDir()
	configFile := writeConfig(t, dir, `
projects:
  - code: core
    aws_profile: core-devops
    gcloud_config: core-gcp
    azure_config_dir: `+filepath.Join(dir, "azure")+`
    kubeconfig: `+filepath.Join(dir, "kubeconfig")+`
`)
	env := testEnv(dir)
	env["OPSCTX_PROJECT"] = "core"
	env["AWS_PROFILE"] = "wrong"
	env["CLOUDSDK_ACTIVE_CONFIG_NAME"] = "wrong"
	env["AZURE_CONFIG_DIR"] = "wrong"
	env["KUBECONFIG"] = "wrong"

	report := Run(Options{
		Paths: testPaths(dir, configFile),
		Env:   env,
	})

	assertContains(t, report, Warn, "env", `AWS_PROFILE="wrong", want "core-devops"`)
	assertContains(t, report, Warn, "env", `CLOUDSDK_ACTIVE_CONFIG_NAME="wrong", want "core-gcp"`)
	assertContains(t, report, Warn, "env", `AZURE_CONFIG_DIR="wrong"`)
	assertContains(t, report, Warn, "env", `KUBECONFIG="wrong"`)
}

func TestRunOmitsUnsetOptionalEnvMatches(t *testing.T) {
	dir := t.TempDir()
	configFile := writeConfig(t, dir, `
projects:
  - code: core
    aws_profile: core-devops
`)
	env := testEnv(dir)
	env["OPSCTX_PROJECT"] = "core"
	env["AWS_PROFILE"] = "core-devops"

	report := Run(Options{
		Paths: testPaths(dir, configFile),
		Env:   env,
		LookPath: func(name string) (string, error) {
			return "", os.ErrNotExist
		},
	})

	for _, result := range report.Results {
		if result.Check == "env" && strings.Contains(result.Message, "is unset") {
			t.Fatalf("unset optional env should be omitted, got %#v", report.Results)
		}
	}
	assertContains(t, report, OK, "env", "OPSCTX_PROJECT matches")
	assertContains(t, report, OK, "env", "AWS_PROFILE matches")
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
	azureDir := filepath.Join(dir, "azure", "core")
	if err := os.MkdirAll(azureDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(azureDir, "config"), []byte("[cloud]\nname = AzureCloud\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	configFile := writeConfig(t, dir, `
projects:
  - code: core
    aws_profile: core-devops
    aliyun_profile: core-aliyun
    codex_profile: core
    gcloud_config: core-gcp
    azure_config_dir: `+azureDir+`
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
			case "gcloud config configurations list --format=value(name)":
				return "core-gcp\nother\n", nil
			default:
				return "", fmt.Errorf("unexpected command %s", command)
			}
		},
	})

	assertContains(t, report, OK, "aws", `CLI found at /usr/bin/aws`)
	assertProjectContains(t, report, OK, "core", "aws", `profile "core-devops" exists`)
	assertContains(t, report, OK, "aliyun", `CLI found at /usr/bin/aliyun`)
	assertProjectContains(t, report, OK, "core", "aliyun", `profile "core-aliyun" exists`)
	assertProjectContains(t, report, OK, "core", "codex", `profile "core" exists`)
	assertContains(t, report, OK, "gcloud", `CLI found at /usr/bin/gcloud`)
	assertProjectContains(t, report, OK, "core", "gcloud", `configuration "core-gcp" exists`)
	assertContains(t, report, OK, "azure", `CLI found at /usr/bin/az`)
	assertProjectContains(t, report, OK, "core", "azure", "config dir")
	if report.HasErrors() {
		t.Fatalf("doctor should pass: %#v", report.Results)
	}
}

func TestRunWarnsOnMissingExternalProfiles(t *testing.T) {
	dir := t.TempDir()
	configFile := writeConfig(t, dir, `
projects:
  - code: core
    aws_profile: core-devops
    aliyun_profile: core-aliyun
    codex_profile: core
    gcloud_config: core-gcp
    azure_config_dir: `+filepath.Join(dir, "missing-azure")+`
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

	assertProjectContains(t, report, Warn, "core", "aws", `profile "core-devops" not found`)
	assertProjectContains(t, report, Warn, "core", "aliyun", `profile "core-aliyun" not found`)
	assertProjectContains(t, report, Warn, "core", "codex", `profile "core" file not found`)
	assertProjectContains(t, report, Warn, "core", "gcloud", `configuration "core-gcp" not found`)
	assertProjectContains(t, report, Warn, "core", "azure", "config dir")
	if report.HasErrors() {
		t.Fatalf("doctor should not fail on missing optional external profiles: %#v", report.Results)
	}
}

func TestRunWarnsWhenGCloudAndAzureCLIsMissing(t *testing.T) {
	dir := t.TempDir()
	configFile := writeConfig(t, dir, `
projects:
  - code: core
    gcloud_config: core-gcp
    azure_config_dir: `+filepath.Join(dir, "azure")+`
`)

	report := Run(Options{
		Paths: testPaths(dir, configFile),
		Env:   testEnv(dir),
		LookPath: func(name string) (string, error) {
			return "", os.ErrNotExist
		},
	})

	assertProjectContains(t, report, Warn, "core", "gcloud", "gcloud CLI not found")
	assertProjectContains(t, report, Warn, "core", "azure", "az CLI not found")
	if report.HasErrors() {
		t.Fatalf("doctor should not fail when optional cloud CLIs are missing: %#v", report.Results)
	}
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
		SSHDir:     filepath.Join(dir, "ssh"),
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

func writeSSHConfig(t *testing.T, dir, content string) {
	t.Helper()
	path := filepath.Join(dir, ".ssh", "config")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
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

func assertProjectContains(t *testing.T, report Report, level Level, project, check, message string) {
	t.Helper()
	for _, result := range report.Results {
		if result.Level == level && result.Project == project && result.Check == check && strings.Contains(result.Message, message) {
			return
		}
	}
	t.Fatalf("missing %s [%s] %s %q in %#v", level, project, check, message, report.Results)
}
