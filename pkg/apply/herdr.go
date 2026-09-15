package apply

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/matheme/pkg"
)

// herdrThemeTokens is the token order used when writing [theme.custom].
var herdrThemeTokens = []string{
	"accent", "panel_bg", "sidebar_bg", "active_row_bg", "selection_bg",
	"surface0", "surface1", "surface_dim", "overlay0", "overlay1",
	"text", "subtext0", "mauve", "green", "yellow", "red", "blue", "teal", "peach",
}

type herdrPalette struct {
	base   string
	tokens map[string]string
}

// herdrDim keeps the palette's own dim color when it stays readable on bg, otherwise blends
// fg into bg until minRatio contrast is reached.
func herdrDim(bg, fg, preferred rgb, minRatio float64) rgb {
	if contrast(bg, preferred) >= minRatio {
		return preferred
	}
	quantize := func(c rgb) rgb {
		q, err := parseHex(c.hex())
		if err != nil {
			return c
		}
		return q
	}
	low, high := 0.0, 1.0
	best := quantize(fg)
	for i := 0; i < 16; i++ {
		mid := (low + high) / 2
		candidate := quantize(mix(bg, fg, mid))
		if contrast(bg, candidate) >= minRatio {
			high = mid
			best = candidate
		} else {
			low = mid
		}
	}
	return best
}

func herdrThemeColor(theme *pkg.Theme, key string) (rgb, error) {
	color, err := parseHex(theme.Base16[key])
	if err != nil {
		return rgb{}, fmt.Errorf("failed to read %s: %w", key, err)
	}
	return color, nil
}

// parseHerdrPalette maps the base16 palette onto herdr's theme tokens. Backgrounds come from
// the low base16 slots, foregrounds from the high ones, and dim text falls back to a blend of
// background and foreground when the theme's own comment colors would be unreadable.
func parseHerdrPalette(theme *pkg.Theme) (*herdrPalette, error) {
	if theme.Type != "dark" && theme.Type != "light" {
		return nil, fmt.Errorf("theme type must be dark or light, got %q", theme.Type)
	}

	colors := make(map[string]rgb)
	for _, key := range []string{
		"base00", "base01", "base02", "base03", "base04", "base05",
		"base08", "base09", "base0A", "base0B", "base0C", "base0D", "base0E",
	} {
		color, err := herdrThemeColor(theme, key)
		if err != nil {
			return nil, err
		}
		colors[key] = color
	}

	accent, err := parseHex(theme.Accent)
	if err != nil {
		accent = colors["base0D"]
	}

	base := "catppuccin"
	if theme.Type == "light" {
		base = "catppuccin-latte"
	}

	bg, fg := colors["base00"], colors["base05"]

	return &herdrPalette{
		base: base,
		tokens: map[string]string{
			"accent":        accent.hex(),
			"panel_bg":      bg.hex(),
			"sidebar_bg":    colors["base01"].hex(),
			"active_row_bg": colors["base02"].hex(),
			"selection_bg":  colors["base02"].hex(),
			"surface0":      colors["base01"].hex(),
			"surface1":      colors["base02"].hex(),
			"surface_dim":   mix(bg, colors["base01"], 0.5).hex(),
			"overlay0":      herdrDim(bg, fg, colors["base03"], 3).hex(),
			"overlay1":      herdrDim(bg, fg, colors["base04"], 4.5).hex(),
			"text":          fg.hex(),
			"subtext0":      herdrDim(bg, fg, colors["base04"], 7).hex(),
			"mauve":         colors["base0E"].hex(),
			"green":         colors["base0B"].hex(),
			"yellow":        colors["base0A"].hex(),
			"red":           colors["base08"].hex(),
			"blue":          colors["base0D"].hex(),
			"teal":          colors["base0C"].hex(),
			"peach":         colors["base09"].hex(),
		},
	}, nil
}

func herdrThemeBlock(theme *pkg.Theme) (string, error) {
	palette, err := parseHerdrPalette(theme)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteString("[theme]\n")
	fmt.Fprintf(&b, "name = %q\n", palette.base)
	b.WriteString("auto_switch = false\n\n")
	b.WriteString("[theme.custom]\n")
	for _, token := range herdrThemeTokens {
		fmt.Fprintf(&b, "%s = %q\n", token, palette.tokens[token])
	}
	return b.String(), nil
}

func herdrTableName(line string) string {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "[") {
		return ""
	}
	trimmed = strings.TrimLeft(trimmed, "[")
	if i := strings.IndexAny(trimmed, "]. \t"); i >= 0 {
		trimmed = trimmed[:i]
	}
	return trimmed
}

// replaceHerdrThemeSection swaps the [theme] table and its subtables for block, leaving every
// other table in the file untouched.
func replaceHerdrThemeSection(content, block string) string {
	start, end := -1, len(content)
	offset := 0
	for _, line := range strings.SplitAfter(content, "\n") {
		if name := herdrTableName(line); name != "" {
			if name == "theme" {
				if start < 0 {
					start = offset
				}
			} else if start >= 0 {
				end = offset
				break
			}
		}
		offset += len(line)
	}

	if start < 0 {
		trimmed := strings.TrimRight(content, "\n")
		if trimmed == "" {
			return block
		}
		return trimmed + "\n\n" + block
	}

	result := content[:start] + block
	if tail := strings.TrimLeft(content[end:], "\n"); tail != "" {
		result += "\n" + tail
	}
	return result
}

// HerdrConfigPath resolves the configured herdr config path, defaulting to the global herdr
// config file.
func HerdrConfigPath(configured string) (string, error) {
	if configured != "" {
		return configured, nil
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return filepath.Join(homeDir, ".config", "herdr", "config.toml"), nil
}

// ApplyHerdrTheme rewrites the [theme] section of the herdr config with a [theme.custom] block
// generated from the base16 palette.
func ApplyHerdrTheme(theme *pkg.Theme, configPath string) error {
	block, err := herdrThemeBlock(theme)
	if err != nil {
		return err
	}

	configPath, err = HerdrConfigPath(configPath)
	if err != nil {
		return err
	}

	var content string
	if data, err := os.ReadFile(configPath); err == nil {
		content = string(data)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to read herdr config: %w", err)
	}

	updated := replaceHerdrThemeSection(content, block)

	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create herdr config directory: %w", err)
	}
	if err := os.WriteFile(configPath, []byte(updated), 0644); err != nil {
		return fmt.Errorf("failed to write herdr config: %w", err)
	}

	return nil
}

// ReloadHerdr asks a running herdr server to re-read config.toml so attached clients pick up
// the new theme. When no server is running the theme is already on disk for the next start.
func ReloadHerdr() error {
	if _, err := exec.LookPath("herdr"); err != nil {
		return fmt.Errorf("herdr not found in PATH: %w", err)
	}

	out, err := exec.Command("herdr", "server", "reload-config").CombinedOutput()

	var response struct {
		Error *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(out, &response) == nil {
		if response.Error == nil || response.Error.Code == "server_not_running" {
			return nil
		}
		return fmt.Errorf("failed to reload herdr: %s", response.Error.Message)
	}
	if err != nil {
		return fmt.Errorf("failed to reload herdr: %w (%s)", err, strings.TrimSpace(string(out)))
	}

	return nil
}
