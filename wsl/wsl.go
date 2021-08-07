// package wsl contains the code interacting with the WSL environment.
package wsl

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mook/podman-wsl/winapi"
	"golang.org/x/text/encoding/unicode"
)

type IsDistro string

const (
	IS_DISTRO_RUNNING    = IsDistro("--running")
	IS_DISTRO_REGISTERED = IsDistro("--all")
)

func RegisterWSL(distro, distroPath string, archiveData []byte) error {
	ok, err := isDistro(distro, IS_DISTRO_REGISTERED)
	if err != nil {
		return err
	}
	if !ok {
		err = registerDistro(distro, distroPath, archiveData)
		if err != nil {
			return err
		}
	}
	return nil
}

// EnsurePodman makes sure that the podman server is running.
// The WSL distribution must already have been registered.
func EnsurePodman(distro string, port int) error {
	ok, err := isDistro(distro, IS_DISTRO_RUNNING)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}

	winapi.Debugf("Distro %s is not running; starting podman", distro)
	cmd, err := wslCommand("--distribution", distro, "--exec",
		"/usr/bin/podman", "system", "service", "--time=0",
		fmt.Sprintf("tcp:127.0.0.1:%d", port))
	if err != nil {
		return err
	}
	err = cmd.Start()
	if err != nil {
		return err
	}
	return nil
}

func getWSLExecutable() (string, error) {
	systemDir, err := winapi.SHGetKnownFolderPath(
		winapi.FOLDERID_System,
		winapi.KF_FLAG_DEFAULT)
	if err != nil {
		return "", err
	}
	return filepath.Join(systemDir, "wsl.exe"), nil
}

func wslCommand(args ...string) (*exec.Cmd, error) {
	exe, err := getWSLExecutable()
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(exe, args...)
	return cmd, nil
}

func wsl(args ...string) error {
	cmd, err := wslCommand(args...)
	if err != nil {
		return err
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

func wslWithCapture(args ...string) ([]byte, error) {
	cmd, err := wslCommand(args...)
	if err != nil {
		return nil, err
	}
	cmd.Stderr = os.Stderr
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return output, nil
}

func decodeUTF16(input []byte) (string, error) {
	decoder := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewDecoder()
	bytes, err := decoder.Bytes(input)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func isDistro(distro string, mode IsDistro) (bool, error) {
	output, err := wslWithCapture("--list", string(mode), "--quiet")
	if err != nil {
		return false, err
	}
	text, err := decodeUTF16(output)
	if err != nil {
		return false, err
	}
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == distro {
			return true, nil
		}
	}
	err = winapi.Debugf("distribution %s is not %s", distro, string(mode))
	if err != nil {
		return false, err
	}
	return false, nil
}

// registerDistro actually registers the distribution.
// This does not check that the distribution does not already exist.
func registerDistro(distroName, distroPath string, archiveData []byte) error {
	archive, err := ioutil.TempFile("", "wodman-distro-*.tar")
	if err != nil {
		return fmt.Errorf("could not create temporary archive: %w", err)
	}
	defer os.Remove(archive.Name())
	reader, err := gzip.NewReader(bytes.NewBuffer(archiveData))
	if err != nil {
		return fmt.Errorf("could not open distro archive: %w", err)
	}
	_, err = io.Copy(archive, reader)
	if err != nil {
		return fmt.Errorf("could not write temporary archive: %w", err)
	}
	err = archive.Close()
	if err != nil {
		return fmt.Errorf("could not close temporary archive file: %w", err)
	}

	expandedDistroPath := os.ExpandEnv(distroPath)
	_ = winapi.Debugf("Registering distribution %s at %s from archive %s", distroName, expandedDistroPath, archive.Name())

	err = wsl("--import", distroName, expandedDistroPath, archive.Name())
	if err != nil {
		return fmt.Errorf("failed to register distribution: %w", err)
	}

	return nil
}
