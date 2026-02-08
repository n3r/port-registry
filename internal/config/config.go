package config

import (
	"os"
	"path/filepath"
)

const (
	DefaultServerPort = 51234
	DefaultPortMin    = 3000
	DefaultPortMax    = 9999
)

func DefaultDBPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".port_server", "ports.db")
}
