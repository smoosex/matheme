package apply

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

var piThemeMap = map[string]string{
	"one_light":        "onedark-light",
	"gruvchad":         "gruvbox-dark",
	"rosepine":         "rosepine-dark",
	"everforest":       "everforest-dark",
	"everforest_light": "everforest-light",
	"tundra":           "tundra-dark",
	"bearded-arc":      "bearded-arc-dark",
}

type piThemeControl struct {
	Theme string `json:"theme"`
}

func ApplyPiTheme(themeName string, controlFilePath string) error {
	piTheme, ok := piThemeMap[themeName]
	if !ok {
		return nil
	}

	if controlFilePath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %w", err)
		}
		controlFilePath = filepath.Join(homeDir, ".pi", "agent", "pi-theme.json")
	}

	if err := os.MkdirAll(filepath.Dir(controlFilePath), 0755); err != nil {
		return fmt.Errorf("failed to create pi theme config directory: %w", err)
	}

	data, err := json.MarshalIndent(piThemeControl{Theme: piTheme}, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal pi theme config: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(controlFilePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write pi theme config: %w", err)
	}

	return nil
}
