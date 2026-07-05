package apply

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/matheme/pkg"
)

var sketchybarPaletteOrder = []string{
	"base00", "base01", "base02", "base03", "base04", "base05", "base06", "base07",
	"base08", "base09", "base0A", "base0B", "base0C", "base0D", "base0E", "base0F",
}

func ApplySketchybarTheme(theme *pkg.Theme) error {
	var buf bytes.Buffer

	buf.WriteString("return {\n")

	for i, key := range sketchybarPaletteOrder {
		if color, ok := theme.Base16[key]; ok {
			color = strings.TrimPrefix(color, "#")
			buf.WriteString(fmt.Sprintf("  c%d = 0xff%s,\n", i, color))
		}
	}

	bg := strings.TrimPrefix(theme.Base16["base00"], "#")
	fg := strings.TrimPrefix(theme.Base16["base05"], "#")
	buf.WriteString(fmt.Sprintf("  bg = 0xff%s,\n", bg))
	buf.WriteString(fmt.Sprintf("  fg = 0xff%s,\n", fg))

	buf.WriteString("}\n")

	tmpDir := "/tmp/matheme"
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}

	filePath := filepath.Join(tmpDir, "sketchybar_theme.lua")
	if err := os.WriteFile(filePath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write sketchybar theme file: %w", err)
	}

	return nil
}
