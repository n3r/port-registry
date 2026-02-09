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
	return filepath.Join(home, ".port_server", "ports.db")
}

func DefaultPIDPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".port_server", "port-server.pid")
}

func DefaultLogPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".port_server", "port-server.log")
}
