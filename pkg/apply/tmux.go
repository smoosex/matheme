package apply

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func ApplyTmuxTheme(scriptPath string, themeName string) error {
	if scriptPath == "" {
		return fmt.Errorf("tmux switch script path is empty")
	}

	if strings.HasPrefix(scriptPath, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %w", err)
		}
		scriptPath = filepath.Join(homeDir, scriptPath[2:])
	}

	output, err := exec.Command(scriptPath, "switch", themeName).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to switch tmux theme: %w: %s", err, strings.TrimSpace(string(output)))
	}

	return nil
}
