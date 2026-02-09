package config

import (
	"os"
	"path/filepath"
)

const (
	DefaultServerPort = 51234
	DefaultPortMin    = 1
	DefaultPortMax    = 65535
)

func DefaultDBPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".port-registry", "ports.db")
}

func DefaultPIDPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".port-registry", "port-registry.pid")
}

func DefaultLogPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".port-registry", "port-registry.log")
}
