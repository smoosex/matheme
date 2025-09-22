package apply

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"

	"github.com/matheme/pkg"
)

const alacrittyTemplate = `
[colors.primary]
background = "{{ .Base16.base00 }}"
foreground = "{{ .Base16.base05 }}"
dim_foreground = "{{ .Base16.base04 }}"
bright_foreground = "{{ .Base16.base06 }}"

[colors.cursor]
text = "{{ .Base16.base00 }}"
cursor = "{{ .Base16.base05 }}"

[colors.vi_mode_cursor]
text = "{{ .Base16.base00 }}"
cursor = "{{ .Base16.base05 }}"

[colors.search.matches]
foreground = "{{ .Base16.base00 }}"
background = "{{ .Base16.base0B }}"

[colors.search.focused_match]
foreground = "{{ .Base16.base00 }}"
background = "{{ .Base16.base0D }}"

[colors.footer_bar]
foreground = "{{ .Base16.base00 }}"
background = "{{ .Base16.base0B }}"

[colors.hints.start]
foreground = "{{ .Base16.base00 }}"
background = "{{ .Base16.base0A }}"

[colors.hints.end]
foreground = "{{ .Base16.base00 }}"
background = "{{ .Base16.base09 }}"

[colors.selection]
text = "{{ .Base16.base00 }}"
background = "{{ .Base16.base05 }}"

[colors.normal]
black = "{{ .Base16.base01 }}"
red = "{{ .Base16.base08 }}"
green = "{{ .Base16.base0B }}"
yellow = "{{ .Base16.base0A }}"
blue = "{{ .Base16.base0D }}"
magenta = "{{ .Base16.base0E }}"
cyan = "{{ .Base16.base0C }}"
white = "{{ .Base16.base05 }}"

[colors.bright]
black = "{{ .Base16.base03 }}"
red = "{{ .Base16.base08 }}"
green = "{{ .Base16.base0B }}"
yellow = "{{ .Base16.base0A }}"
blue = "{{ .Base16.base0D }}"
magenta = "{{ .Base16.base0E }}"
cyan = "{{ .Base16.base0C }}"
white = "{{ .Base16.base07 }}"

{{ if .Base16.base0F }}
[[colors.indexed_colors]]
index = 16
color = "{{ .Base16.base0F }}"
{{ end }}
`

func ApplyAlacrittyTheme(theme *pkg.Theme) error {
	tmpl, err := template.New("alacritty").Parse(alacrittyTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse alacritty template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, theme); err != nil {
		return fmt.Errorf("failed to execute alacritty template: %w", err)
	}

	tmpDir := "/tmp/matheme"
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}

	filePath := filepath.Join(tmpDir, "alacritty_theme.toml")
	if err := os.WriteFile(filePath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write alacritty theme file: %w", err)
	}

	return nil
}
