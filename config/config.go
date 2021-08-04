// Package config handles configuration for wodman
package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"strconv"
	"strings"

	"github.com/mcuadros/go-defaults"
	"github.com/mook/podman-wsl/winapi"
	toml "github.com/pelletier/go-toml/v2"
)

// We are rolling our own configuration for better Windows support:
// github.com/spf13/viper, github.com/rakyll/globalconf don't support Windows
// github.com/tucnak/store doesn't support environment variables.

// ConfigType is the external configuration interface.
type ConfigType struct {
	Port         int    `default:"1234"`                   // The TCP port that podman will listen on.
	Distro       string `default:"podman"`                 // The name of the distribution in WSL.
	RegisterOnly bool   `default:"false"`                  // If set, only register without doing anything else.
	DistroPath   string `default:"${LOCALAPPDATA}/podman"` // Where to register the distribution.
}

type ConfigWrapper struct {
	WSL ConfigType
}

var Config ConfigType

func init() {
	defaults.SetDefaults(&Config)
	readConfig()
	readEnvironment()
}

func readConfig() {
	appData, err := winapi.SHGetKnownFolderPath(winapi.FOLDERID_RoamingAppData, winapi.KF_FLAG_DEFAULT)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting %%APPDATA%% folder: %s", err)
		var ok bool
		appData, ok = os.LookupEnv("APPDATA")
		if !ok {
			return
		}
	}
	// We store the configs in containers.conf, under a "wsl" key.
	configPath := path.Join(appData, "containers", "containers.conf")
	configBytes, err := ioutil.ReadFile(configPath)
	if err == nil {
		var wrapper ConfigWrapper
		err = toml.Unmarshal(configBytes, &wrapper)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing configuration file: %s\n", err)
		}
		Config = wrapper.WSL
		Config.DistroPath = os.ExpandEnv(Config.DistroPath)
	} else if !errors.Is(err, os.ErrNotExist) {
		fmt.Fprintf(os.Stderr, "Error reading configuration file: %s\n", err)
	}
}

func readEnvironment() {
	prefix := "PODMAN_"
	val := reflect.ValueOf(&Config)
	for i := 0; i < val.Elem().NumField(); i++ {
		field := val.Elem().Type().Field(i)
		name := prefix + strings.ToUpper(field.Name)
		value, ok := os.LookupEnv(name)
		if !ok {
			continue
		}
		switch field.Type.Kind() {
		case reflect.Bool:
			switch strings.ToLower(strings.TrimSpace(value)) {
			case "", "f", "false", "n", "no", "0":
				val.Elem().Field(i).SetBool(false)
			default:
				val.Elem().Field(i).SetBool(true)
			}
		case reflect.Int:
			num, err := strconv.ParseInt(strings.TrimSpace(value), 0, 0)
			if err == nil {
				val.Elem().Field(i).SetInt(num)
			} else {
				fmt.Fprintf(os.Stderr, "Error parsing %s: %s\n", name, err)
			}
		case reflect.String:
			val.Elem().Field(i).SetString(value)
		default:
			fmt.Fprintf(os.Stderr, "Ignoring %s, don't know how to handle\n", name)
		}
	}
}
