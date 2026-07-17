package switcher

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ninj4dkill4/octx/internal/config"
)

type Options struct {
	ConfigFile string
	SSHDir     string
}

type Result struct {
	Project   config.Project
	SSHConfig string
}

type ClearResult struct{}

func Switch(projectCode string, opts Options) (Result, error) {
	paths, err := resolvePaths(opts)
	if err != nil {
		return Result{}, err
	}

	cfg, err := config.LoadConfig(paths.ConfigFile)
	if err != nil {
		return Result{}, err
	}

	project, ok := cfg.FindProject(projectCode)
	if !ok {
		return Result{}, fmt.Errorf("project %q not found", projectCode)
	}

	sshConfig, err := prepareSSHConfig(project, paths.SSHDir)
	if err != nil {
		return Result{}, err
	}

	return Result{
		Project:   project,
		SSHConfig: sshConfig,
	}, nil
}

func Clear(opts Options) (ClearResult, error) {
	return ClearResult{}, nil
}

func ShellExports(project config.Project, sshConfig string) string {
	var b strings.Builder
	writeExport(&b, "OPSCTX_PROJECT", project.Code)
	writeExport(&b, "AWS_PROFILE", project.AWSProfile)
	writeExport(&b, "CODEX_PROFILE", project.CodexProfile)
	writeExport(&b, "ALIBABA_CLOUD_PROFILE", project.AliyunProfile)
	writeExport(&b, "CLOUDSDK_ACTIVE_CONFIG_NAME", project.GCloudConfig)
	writePathExport(&b, "AZURE_CONFIG_DIR", project.AzureConfigDir)
	writePathExport(&b, "KUBECONFIG", project.Kubeconfig)
	writeExport(&b, "OCTX_SSH_CONFIG", sshConfig)
	return b.String()
}

func ShellUnsetAll() string {
	var b strings.Builder
	for _, key := range []string{
		"OPSCTX_PROJECT",
		"AWS_PROFILE",
		"CODEX_PROFILE",
		"ALIBABA_CLOUD_PROFILE",
		"CLOUDSDK_ACTIVE_CONFIG_NAME",
		"AZURE_CONFIG_DIR",
		"KUBECONFIG",
		"OCTX_SSH_CONFIG",
	} {
		writeExport(&b, key, "")
	}
	return b.String()
}

func resolvePaths(opts Options) (config.Paths, error) {
	paths, err := config.DefaultPaths()
	if err != nil {
		return config.Paths{}, err
	}
	if opts.ConfigFile != "" {
		paths.ConfigFile = opts.ConfigFile
	}
	if opts.SSHDir != "" {
		paths.SSHDir = opts.SSHDir
	}
	return paths, nil
}

func prepareSSHConfig(project config.Project, sshDir string) (string, error) {
	if project.SSHConfig == "" {
		return "", nil
	}

	source := config.ExpandPath(project.SSHConfig)
	if _, err := os.Stat(source); err != nil {
		return "", fmt.Errorf("ssh config %q: %w", source, err)
	}

	sshDir = config.ExpandPath(sshDir)
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		return "", err
	}

	path := filepath.Join(sshDir, safeFileName(project.Code)+".config")
	content := fmt.Sprintf("Include ~/.ssh/config\nInclude %s\n", source)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func writeExport(b *strings.Builder, key, value string) {
	if value == "" {
		fmt.Fprintf(b, "unset %s\n", key)
		return
	}
	fmt.Fprintf(b, "export %s=%s\n", key, shellQuote(value))
}

func writePathExport(b *strings.Builder, key, value string) {
	if value != "" {
		value = config.ExpandPath(value)
	}
	writeExport(b, key, value)
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

var unsafeFileNameChars = regexp.MustCompile(`[^A-Za-z0-9_.-]+`)

func safeFileName(value string) string {
	sum := sha1.Sum([]byte(value))
	suffix := hex.EncodeToString(sum[:4])
	value = unsafeFileNameChars.ReplaceAllString(value, "_")
	value = strings.Trim(value, "._-")
	if value == "" {
		value = "project"
	}
	return value + "-" + suffix
}
