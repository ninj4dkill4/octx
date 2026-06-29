package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/vanle3/opsctx/internal/config"
	"github.com/vanle3/opsctx/internal/switcher"
	opsTUI "github.com/vanle3/opsctx/internal/tui"
)

type rootOptions struct {
	configFile string
	stateFile  string
	sshCurrent string
	shell      bool
}

func NewRootCommand() *cobra.Command {
	opts := &rootOptions{}

	cmd := &cobra.Command{
		Use:   "octx",
		Short: "Switch devops project context",
		Long:  "octx switches terminal context by project code for AWS, Codex, and SSH.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRoot(cmd, opts)
		},
	}

	cmd.PersistentFlags().StringVar(&opts.configFile, "config", "", "config file path")
	cmd.PersistentFlags().StringVar(&opts.stateFile, "state", "", "state file path")
	cmd.PersistentFlags().StringVar(&opts.sshCurrent, "ssh-current", "", "generated current SSH config path")
	cmd.Flags().BoolVar(&opts.shell, "shell", false, "switch selected project and print shell exports for eval")
	_ = cmd.Flags().MarkHidden("shell")

	cmd.AddCommand(
		newInitCommand(opts),
		newCurrentCommand(opts),
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

func newCurrentCommand(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "current",
		Short: "Show current project context",
		RunE: func(cmd *cobra.Command, args []string) error {
			paths, err := pathsFromOptions(opts)
			if err != nil {
				return err
			}
			state, err := config.LoadState(paths.StateFile)
			if err != nil {
				if errors.Is(err, config.ErrNotFound) {
					fmt.Fprintln(cmd.OutOrStdout(), "No current project")
					return nil
				}
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s\n", state.CurrentProject)
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
	if opts.stateFile != "" {
		paths.StateFile = opts.stateFile
	}
	if opts.sshCurrent != "" {
		paths.SSHCurrent = opts.sshCurrent
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
	project, err := opsTUI.Pick(cfg, output)
	if err != nil {
		return err
	}
	if project == nil {
		return nil
	}

	result, err := switcher.Switch(project.Code, switcher.Options{
		ConfigFile: paths.ConfigFile,
		StateFile:  paths.StateFile,
		SSHCurrent: paths.SSHCurrent,
	})
	if err != nil {
		return err
	}
	if opts.shell {
		fmt.Fprint(cmd.OutOrStdout(), switcher.ShellExports(result.Project))
		return nil
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Switched to %s\n", result.Project.Code)
	return nil
}

func friendlyConfigError(err error, path string) error {
	if errors.Is(err, config.ErrNotFound) {
		return fmt.Errorf("config not found at %s; run `octx init` first", path)
	}
	return err
}
