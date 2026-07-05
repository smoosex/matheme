package apply

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/matheme/pkg"
)

func ApplyBordersTheme(theme *pkg.Theme, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open borders file %s: %w", filePath, err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	found := false
	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimmedLine, "active_color") {
			found = true
			indent := ""
			if idx := strings.Index(line, "active_color"); idx > 0 {
				indent = line[:idx]
			}
			newColor := strings.TrimPrefix(theme.Base16["base08"], "#")
			line = fmt.Sprintf("%sactive_color=0xff%s", indent, newColor)
		}
		lines = append(lines, line)
	}

	if !found {
		return nil
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading borders file: %w", err)
	}

	output := strings.Join(lines, "\n")
	if err := os.WriteFile(filePath, []byte(output), 0644); err != nil {
		return fmt.Errorf("failed to write updated borders file: %w", err)
	}

	return nil
}
