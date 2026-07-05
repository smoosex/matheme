package apply

import (
	"fmt"
	"os/exec"

	"github.com/matheme/pkg"
)

func ApplySystemAppearance(theme *pkg.Theme) error {
	isDarkMode := "false"
	if theme.Type == "dark" {
		isDarkMode = "true"
	}

	script := fmt.Sprintf(`tell application "System Events" to tell appearance preferences to set dark mode to %s`, isDarkMode)
	cmd := exec.Command("osascript", "-e", script)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to switch system appearance: %w", err)
	}

	return nil
}

func ApplyWallpaper(wallpaperPath string) error {
	script := fmt.Sprintf(`tell application "Finder" to set desktop picture to POSIX file "%s"`, wallpaperPath)
	cmd := exec.Command("osascript", "-e", script)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set wallpaper: %w", err)
	}

	return nil
}
