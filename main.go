package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/containers/podman/v3/cmd/podman/registry"
	"github.com/containers/podman/v3/libpod/define"
	"github.com/containers/podman/v3/pkg/domain/entities"
	"github.com/containers/podman/v3/pkg/terminal"
	"github.com/mook/podman-wsl/config"
	"github.com/mook/podman-wsl/wsl"
	"github.com/sirupsen/logrus"

	_ "embed"

	_ "github.com/containers/podman/v3/cmd/podman/containers"
	_ "github.com/containers/podman/v3/cmd/podman/images"
	_ "github.com/containers/podman/v3/cmd/podman/machine"
	_ "github.com/containers/podman/v3/cmd/podman/networks"
	_ "github.com/containers/podman/v3/cmd/podman/pods"
	_ "github.com/containers/podman/v3/cmd/podman/system"
	_ "github.com/containers/podman/v3/cmd/podman/system/connection"
	_ "github.com/containers/podman/v3/cmd/podman/volumes"

	_ "github.com/mook/podman-wsl/cmd"
)

//go:embed image/podman-wsl-distro.tar.gz
var distroArchive []byte

func main() {
	parseCommands()
	execute()
}

func ensurePodman() error {
	err := wsl.RegisterWSL(config.Config.Distro, config.Config.DistroPath, distroArchive)
	if err != nil {
		return err
	}
	err = wsl.EnsurePodman(config.Config.Distro, config.Config.Port)
	if err != nil {
		return err
	}
	return nil
}

func ensurePodmanHook() {
	err := ensurePodman()
	if err != nil {
		logrus.Fatalf("Error launching podman in WSL: %s\n", err)
	}
}

func parseCommands() {
	cfg := registry.PodmanConfig()
	cfg.EngineMode = entities.TunnelMode
	cfg.URI = "tcp://127.0.0.1:1234"

	for _, c := range registry.Commands {
		found := false
		for _, mode := range c.Mode {
			if mode == cfg.EngineMode {
				found = true
				break
			}
		}
		if !found {
			continue
		}

		parent := rootCmd
		if c.Parent != nil {
			parent = c.Parent
		}
		parent.AddCommand(c.Command)
	}

	if err := terminal.SetConsole(); err != nil {
		logrus.Error(err)
		os.Exit(1)
	}
}

func execute() {
	if err := rootCmd.ExecuteContext(registry.GetContextWithOptions()); err != nil {
		fmt.Fprintln(os.Stderr, formatError(err))
	} else if registry.GetExitCode() == registry.ExecErrorCodeGeneric {
		// The exitCode modified from registry.ExecErrorCodeGeneric,
		// indicates an application
		// running inside of a container failed, as opposed to the
		// podman command failed.  Must exit with that exit code
		// otherwise command exited correctly.
		registry.SetExitCode(0)
	}
	os.Exit(registry.GetExitCode())
}

func formatError(err error) string {
	if errors.Is(err, define.ErrOCIRuntime) {
		// libpod.getOCIRuntimeError() wraps the generic error with the error
		// string via github.com/pkg/errors.Wrapf(), which puts the underlying
		// error string at the end; however, it's actually the prefix that
		// contains the more specific details.
		message := err.Error()
		suffix := ": " + define.ErrOCIRuntime.Error()
		if strings.HasSuffix(message, suffix) {
			return fmt.Sprintf(
				"Error: %s: %s",
				define.ErrOCIRuntime,
				strings.TrimSuffix(message, suffix),
			)
		}
		return message
	}
	if logrus.IsLevelEnabled(logrus.TraceLevel) {
		return fmt.Sprintf("Error: %+v", err)
	}
	return fmt.Sprintf("Error: %v", err)
}
