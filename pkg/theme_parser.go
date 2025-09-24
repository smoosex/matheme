package pkg

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Theme struct {
	Type   string            `toml:"type"`
	Base16 map[string]string `toml:"base_16"`
}

func ParseTheme(name string) (*Theme, error) {
	homePath, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get user home directory: %v\n", err)
	}
	themePath := filepath.Join(homePath, ".config/matheme/themes", name+".toml")

	data, err := os.ReadFile(themePath)
	if err != nil {
		return nil, err
	}

	var theme Theme
	if _, err := toml.Decode(string(data), &theme); err != nil {
		return nil, err
	}

	return &theme, nil
}
