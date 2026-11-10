package pkg

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ThemeDir returns the directory matheme reads theme definitions from.
func ThemeDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return filepath.Join(homeDir, ".config", "matheme", "themes"), nil
}

// ThemePath resolves the definition file of a single theme.
func ThemePath(name string) (string, error) {
	dir, err := ThemeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, name+".toml"), nil
}

// ListThemeNames returns every theme name found in the theme directory.
func ListThemeNames() ([]string, error) {
	dir, err := ThemeDir()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read theme directory %s: %w", dir, err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".toml" {
			continue
		}
		names = append(names, strings.TrimSuffix(entry.Name(), ".toml"))
	}
	return names, nil
}

// CurrentThemePath returns the file recording the last fully applied theme.
func CurrentThemePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return filepath.Join(homeDir, ".config", "matheme", "current_theme"), nil
}

// CurrentTheme returns the recorded theme, or an empty string when nothing is recorded yet.
func CurrentTheme() (string, error) {
	path, err := CurrentThemePath()
	if err != nil {
		return "", err
	}
	content, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to read state file %s: %w", path, err)
	}
	return strings.TrimSpace(string(content)), nil
}

// WriteCurrentTheme records the theme as fully applied.
func WriteCurrentTheme(name string) error {
	path, err := CurrentThemePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}
	if err := os.WriteFile(path, []byte(name), 0o644); err != nil {
		return fmt.Errorf("failed to write state file %s: %w", path, err)
	}
	return nil
}
