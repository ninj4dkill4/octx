package cli

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/ninj4dkill4/octx/internal/config"
	"github.com/ninj4dkill4/octx/internal/doctor"
	"github.com/ninj4dkill4/octx/internal/switcher"
	opsTUI "github.com/ninj4dkill4/octx/internal/tui"
	"github.com/ninj4dkill4/octx/internal/version"
	"github.com/spf13/cobra"
)

type rootOptions struct {
	configFile string
	sshDir     string
	shell      bool
}

func NewRootCommand() *cobra.Command {
	opts := &rootOptions{}

	cmd := &cobra.Command{
		Use:           "octx",
		Short:         "Switch ops profiles for Codex workflows",
		Long:          "octx switches ops profiles by project code for Codex, SSH, cloud CLIs, and Kubernetes.",
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRoot(cmd, opts)
		},
	}

	cmd.PersistentFlags().StringVar(&opts.configFile, "config", "", "config file path")
	cmd.PersistentFlags().StringVar(&opts.sshDir, "ssh-dir", "", "generated per-project SSH config directory")
	cmd.Flags().BoolVar(&opts.shell, "shell", false, "switch selected project and print shell exports for eval")
	_ = cmd.Flags().MarkHidden("shell")

	cmd.AddCommand(
		newInitCommand(opts),
		newCurrentCommand(opts),
		newDoctorCommand(opts),
		newVersionCommand(),
	)

	return cmd
}

func newInitCommand(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Write a sample config",
		RunE: func(cmd *cobra.Command, args []string) error {
			paths, err := pathsFromOptions(opts)
			if err != nil {
				return err
			}
			if _, err := os.Stat(config.ExpandPath(paths.ConfigFile)); err == nil {
				return fmt.Errorf("config already exists: %s", paths.ConfigFile)
			}
			if err := config.WriteSampleConfig(paths.ConfigFile); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Wrote %s\n", paths.ConfigFile)
			return nil
		},
	}
}

func newDoctorCommand(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:          "doctor",
		Short:        "Diagnose local octx setup",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			paths, err := pathsFromOptions(opts)
			if err != nil {
				return err
			}
			report := doctor.Run(doctor.Options{Paths: paths})
			writeDoctorReport(cmd.OutOrStdout(), report)
			if report.HasErrors() {
				return fmt.Errorf("doctor found %d error(s)", report.ErrorCount())
			}
			return nil
		},
	}
}

func writeDoctorReport(out io.Writer, report doctor.Report) {
	var global []doctor.Result
	projectOrder := make([]string, 0)
	projectResults := make(map[string][]doctor.Result)
	for _, result := range report.Results {
		if result.Project == "" {
			global = append(global, result)
			continue
		}
		if _, ok := projectResults[result.Project]; !ok {
			projectOrder = append(projectOrder, result.Project)
		}
		projectResults[result.Project] = append(projectResults[result.Project], result)
	}

	wroteSection := false
	writeSection := func(name string, results []doctor.Result) {
		if len(results) == 0 {
			return
		}
		if wroteSection {
			fmt.Fprintln(out)
		}
		fmt.Fprintf(out, "[%s]\n", name)
		for _, result := range results {
			fmt.Fprintf(out, "%-5s %-8s %s\n", result.Level, result.Check, result.Message)
		}
		wroteSection = true
	}

	writeSection("global", global)
	for _, project := range projectOrder {
		writeSection(project, projectResults[project])
	}
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print octx version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(cmd.OutOrStdout(), version.Version)
		},
	}
}

func newCurrentCommand(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "current",
		Short: "Show current project context",
		RunE: func(cmd *cobra.Command, args []string) error {
			current := os.Getenv("OPSCTX_PROJECT")
			if current == "" {
				fmt.Fprintln(cmd.OutOrStdout(), "No current project")
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s\n", current)
			return nil
		},
	}
}

func pathsFromOptions(opts *rootOptions) (config.Paths, error) {
	paths, err := config.DefaultPaths()
	if err != nil {
		return config.Paths{}, err
	}
	if opts.configFile != "" {
		paths.ConfigFile = opts.configFile
	}
	if opts.sshDir != "" {
		paths.SSHDir = opts.sshDir
	}
	return paths, nil
}

func runRoot(cmd *cobra.Command, opts *rootOptions) error {
	paths, err := pathsFromOptions(opts)
	if err != nil {
		return err
	}
	cfg, err := config.LoadConfig(paths.ConfigFile)
	if err != nil {
		return friendlyConfigError(err, paths.ConfigFile)
	}

	output := cmd.OutOrStdout()
	if opts.shell {
		output = os.Stderr
	}
	currentProject := currentProjectForPicker(cfg, os.Getenv("OPSCTX_PROJECT"))
	selection, err := opsTUI.Pick(cfg, currentProject, output)
	if err != nil {
		return err
	}
	if selection == nil {
		return nil
	}
	if selection.Clear {
		if opts.shell {
			fmt.Fprint(cmd.OutOrStdout(), switcher.ShellUnsetAll())
			return nil
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Unset")
		return nil
	}
	if selection.Project == nil {
		return nil
	}

	result, err := switcher.Switch(selection.Project.Code, switcher.Options{
		ConfigFile: paths.ConfigFile,
		SSHDir:     paths.SSHDir,
	})
	if err != nil {
		return err
	}
	if opts.shell {
		fmt.Fprint(cmd.OutOrStdout(), switcher.ShellExports(result.Project, result.SSHConfig))
		return nil
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Switched to %s\n", result.Project.Code)
	return nil
}

func currentProjectForPicker(cfg config.Config, shellProject string) string {
	if shellProject == "" || shellProject == config.UnsetProjectCode {
		return config.UnsetProjectCode
	}
	if _, ok := cfg.FindProject(shellProject); !ok {
		return config.UnsetProjectCode
	}
	return shellProject
}

func friendlyConfigError(err error, path string) error {
	if errors.Is(err, config.ErrNotFound) {
		return fmt.Errorf("config not found at %s; run `octx init` first", path)
	}
	return err
}
