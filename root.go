package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/containers/podman/v3/cmd/podman/registry"
	"github.com/containers/podman/v3/cmd/podman/validate"
	"github.com/containers/podman/v3/version"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	rootCmd = &cobra.Command{
		TraverseChildren:   true,
		PersistentPreRunE:  persistentPreRunE,
		RunE:               validate.SubCommandExists,
		PersistentPostRunE: persistentPostRunE,
		Version:            version.Version.String(),
	}
)

func init() {
	flags := rootCmd.Flags()
	flags.String("context", "default", "Name of the context to use to connect to the daemon (This flag is a NOOP and provided solely for scripting compatibility.)")
	flags.MarkHidden("context")
}

func persistentPreRunE(cmd *cobra.Command, args []string) error {
	logrus.Debugf("Called %s.PersistentPreRunE(%s)", cmd.Name(), strings.Join(os.Args, " "))

	// Help, completion and commands with subcommands are special cases, no need for more setup
	// Completion cmd is used to generate the shell scripts
	if cmd.Name() == "help" || cmd.Name() == "completion" || cmd.HasSubCommands() {
		return nil
	}

	// Special case if command is hidden completion command ("__complete","__completeNoDesc")
	// Since __completeNoDesc is an alias the cm.Name is always __complete
	if cmd.Name() == cobra.ShellCompRequestCmd {
		// Parse the cli arguments after the the completion cmd (always called as second argument)
		// This ensures that the --url, --identity and --connection flags are properly set
		compCmd, _, err := cmd.Root().Traverse(os.Args[2:])
		if err != nil {
			return err
		}
		// If we don't complete the root cmd hide all root flags
		// so they won't show up in the completions on subcommands.
		if compCmd != compCmd.Root() {
			compCmd.Root().Flags().VisitAll(func(flag *pflag.Flag) {
				flag.Hidden = true
			})
		}
		// No need for further setup the completion logic setups the engines as needed.
		return nil
	}

	cfg := registry.PodmanConfig()

	// Prep the engines
	if _, err := registry.NewImageEngine(cmd, args); err != nil {
		return err
	}
	if _, err := registry.NewContainerEngine(cmd, args); err != nil {
		return err
	}

	for _, env := range cfg.Engine.Env {
		splitEnv := strings.SplitN(env, "=", 2)
		if len(splitEnv) != 2 {
			return fmt.Errorf("invalid environment variable for engine %s, valid configuration is KEY=value pair", env)
		}
		// skip if the env is already defined
		if _, ok := os.LookupEnv(splitEnv[0]); ok {
			logrus.Debugf("environment variable %s is already defined, skip the settings from containers.conf", splitEnv[0])
			continue
		}
		if err := os.Setenv(splitEnv[0], splitEnv[1]); err != nil {
			return err
		}
	}

	context := cmd.Root().LocalFlags().Lookup("context")
	if context.Value.String() != "default" {
		return errors.New("Podman does not support swarm, the only --context value allowed is \"default\"")
	}
	return nil
}

func persistentPostRunE(cmd *cobra.Command, args []string) error {
	logrus.Debugf("Called %s.PersistentPostRunE(%s)", cmd.Name(), strings.Join(os.Args, " "))

	ctx := registry.Context()
	if ctx == nil {
		return fmt.Errorf("no registry context")
	}
	if imageEngine := registry.ImageEngine(); imageEngine != nil {
		imageEngine.Shutdown(registry.Context())
	}
	if containerEngine := registry.ContainerEngine(); containerEngine != nil {
		containerEngine.Shutdown(registry.Context())
	}
	return nil
}
