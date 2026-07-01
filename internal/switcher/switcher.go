package switcher

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ninj4dkill4/octx/internal/config"
)

type Options struct {
	ConfigFile string
	StateFile  string
	SSHCurrent string
}

type Result struct {
	Project    config.Project
	StateFile  string
	SSHCurrent string
}

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

	if err := applySSH(project, paths.SSHCurrent); err != nil {
		return Result{}, err
	}

	state := config.State{
		CurrentProject: project.Code,
		SwitchedAt:     time.Now(),
	}
	if err := config.SaveState(paths.StateFile, state); err != nil {
		return Result{}, err
	}

	return Result{
		Project:    project,
		StateFile:  paths.StateFile,
		SSHCurrent: paths.SSHCurrent,
	}, nil
}

func ShellExports(project config.Project) string {
	var b strings.Builder
	writeExport(&b, "OPSCTX_PROJECT", project.Code)
	writeExport(&b, "AWS_PROFILE", project.AWSProfile)
	writeExport(&b, "CODEX_PROFILE", project.CodexProfile)
	writeExport(&b, "ALIBABA_CLOUD_PROFILE", project.AliyunProfile)
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
	if opts.StateFile != "" {
		paths.StateFile = opts.StateFile
	}
	if opts.SSHCurrent != "" {
		paths.SSHCurrent = opts.SSHCurrent
	}
	return paths, nil
}

func applySSH(project config.Project, currentPath string) error {
	currentPath = config.ExpandPath(currentPath)
	if project.SSHConfig == "" {
		if err := os.Remove(currentPath); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}

	source := config.ExpandPath(project.SSHConfig)
	if _, err := os.Stat(source); err != nil {
		return fmt.Errorf("ssh config %q: %w", source, err)
	}

	if err := os.MkdirAll(filepath.Dir(currentPath), 0o700); err != nil {
		return err
	}

	tmpPath := currentPath + ".tmp"
	_ = os.Remove(tmpPath)
	if err := os.Symlink(source, tmpPath); err != nil {
		return err
	}
	return os.Rename(tmpPath, currentPath)
}

func writeExport(b *strings.Builder, key, value string) {
	if value == "" {
		fmt.Fprintf(b, "unset %s\n", key)
		return
	}
	fmt.Fprintf(b, "export %s=%s\n", key, shellQuote(value))
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
