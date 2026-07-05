package apply

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/matheme/pkg"
)

var kittyPaletteOrder = []string{
	"base01", "base08", "base0B", "base0A", "base0D", "base0E", "base0C", "base05",
	"base03", "base08", "base0B", "base0A", "base0D", "base0E", "base0C", "base06",
}

func ApplyKittyTheme(theme *pkg.Theme) error {
	var buf bytes.Buffer

	for i, key := range kittyPaletteOrder {
		if color, ok := theme.Base16[key]; ok {
			buf.WriteString(fmt.Sprintf("color%d %s\n", i, color))
		}
	}

	buf.WriteString(fmt.Sprintf("background %s\n", theme.Base16["base00"]))
	buf.WriteString(fmt.Sprintf("foreground %s\n", theme.Base16["base05"]))
	buf.WriteString(fmt.Sprintf("cursor %s\n", theme.Base16["base05"]))
	buf.WriteString(fmt.Sprintf("cursor_text_color %s\n", theme.Base16["base00"]))
	buf.WriteString(fmt.Sprintf("selection_background %s\n", theme.Base16["base03"]))
	buf.WriteString(fmt.Sprintf("selection_foreground %s\n", theme.Base16["base05"]))

	tmpDir := "/tmp/matheme"
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}

	filePath := filepath.Join(tmpDir, "kitty_theme")
	if err := os.WriteFile(filePath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write kitty theme file: %w", err)
	}

	return nil
}
