package common

import (
	"os"
	"path/filepath"
	"strings"
)

func ListThemes() []string {
	homeDir := os.Getenv("HOME")
	themeDir := filepath.Join(homeDir, ".config/matheme/themes")
	entries, err := os.ReadDir(themeDir)
	if err != nil {
		panic(err)
	}

	themes := make([]string, 0)
	for _, entry := range entries {
		ext := filepath.Ext(entry.Name())
		themes = append(themes, strings.TrimSuffix(entry.Name(), ext))
	}
	return themes
}
