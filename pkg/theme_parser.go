package pkg

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Theme struct {
	Type   string            `toml:"type"`
	Base16 map[string]string `toml:"base_16"`
}

func ParseTheme(name string) (*Theme, error) {
	homePath := os.Getenv("HOME")
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
