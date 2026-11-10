package apply

import (
	"fmt"
	"os/exec"
	"strings"
)

func ApplyTmuxTheme(scriptPath string, themeName string) error {
	if scriptPath == "" {
		return fmt.Errorf("tmux switch script path is empty")
	}

	output, err := exec.Command(scriptPath, "switch", themeName).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to switch tmux theme: %w: %s", err, strings.TrimSpace(string(output)))
	}

	return nil
}
