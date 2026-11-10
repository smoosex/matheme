package pkg

import (
	"os"

	"github.com/BurntSushi/toml"
)

type Theme struct {
	Type   string            `toml:"type"`
	Accent string            `toml:"accent"`
	Base16 map[string]string `toml:"base_16"`
}

func ParseTheme(name string) (*Theme, error) {
	themePath, err := ThemePath(name)
	if err != nil {
		return nil, err
	}

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
