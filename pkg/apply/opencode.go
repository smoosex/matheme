package apply

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/matheme/pkg"
)

var opencodeBase16Keys = []string{
	"base00", "base01", "base02", "base03", "base04", "base05", "base06", "base07",
	"base08", "base09", "base0A", "base0B", "base0C", "base0D", "base0E", "base0F",
}

type opencodePalette struct {
	base00, base01, base02, base03, base04, base05, base06, base07 rgb
	base08, base09, base0A, base0B, base0C, base0D, base0E, base0F rgb
	accent                                                         rgb
	light                                                          bool
}

func parseOpencodePalette(theme *pkg.Theme) (*opencodePalette, error) {
	if theme.Type != "dark" && theme.Type != "light" {
		return nil, fmt.Errorf("theme type must be dark or light, got %q", theme.Type)
	}

	colors := make([]rgb, len(opencodeBase16Keys))
	for i, key := range opencodeBase16Keys {
		color, err := parseHex(theme.Base16[key])
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", key, err)
		}
		colors[i] = color
	}

	// accent is curated per theme, so prefer it over the base0D slot.
	accent, err := parseHex(theme.Accent)
	if err != nil {
		accent = colors[13]
	}

	return &opencodePalette{
		base00: colors[0], base01: colors[1], base02: colors[2], base03: colors[3],
		base04: colors[4], base05: colors[5], base06: colors[6], base07: colors[7],
		base08: colors[8], base09: colors[9], base0A: colors[10], base0B: colors[11],
		base0C: colors[12], base0D: colors[13], base0E: colors[14], base0F: colors[15],
		accent: accent,
		light:  theme.Type == "light",
	}, nil
}

// tint lightens color towards white, shade darkens it towards black.
func tint(color rgb, amount float64) rgb {
	return mix(rgb{1, 1, 1}, color, amount)
}

func shade(color rgb, amount float64) rgb {
	return mix(rgb{0, 0, 0}, color, amount)
}

// pickReadable returns the first candidate that reaches minRatio against background, or the
// last one, so that dim palette entries such as base03 cannot make text unreadable.
func pickReadable(background rgb, minRatio float64, candidates ...rgb) rgb {
	picked := background
	for _, candidate := range candidates {
		picked = candidate
		if contrast(background, candidate) >= minRatio {
			return candidate
		}
	}
	return picked
}

// hueScale derives the 100-900 steps opencode uses for agent and status colors.
func hueScale(color rgb) map[string]string {
	scale := map[string]string{
		"100": tint(color, 0.15).hex(),
		"200": tint(color, 0.3).hex(),
		"300": tint(color, 0.5).hex(),
		"400": tint(color, 0.7).hex(),
		"500": color.hex(),
		"600": shade(color, 0.85).hex(),
		"700": shade(color, 0.7).hex(),
		"800": shade(color, 0.55).hex(),
		"900": shade(color, 0.4).hex(),
	}
	return scale
}

func (p *opencodePalette) textOnAccent() rgb {
	if p.light {
		return p.base07
	}
	return p.base00
}

func (p *opencodePalette) mode() map[string]any {
	muted := pickReadable(p.base00, 3, p.base03, p.base04, mix(p.base00, p.base05, 0.55))
	subdued := pickReadable(p.base00, 3, p.base04, mix(p.base00, p.base05, 0.72))
	accentText := p.textOnAccent()

	return map[string]any{
		"hue": map[string]any{
			"gray":   hueScale(p.base05),
			"red":    hueScale(p.base08),
			"orange": hueScale(p.base09),
			"yellow": hueScale(p.base0A),
			"green":  hueScale(p.base0B),
			"cyan":   hueScale(p.base0C),
			"blue":   hueScale(p.base0D),
			"purple": hueScale(p.base0E),
		},
		"text": map[string]any{
			"default": p.base05.hex(),
			"subdued": subdued.hex(),
			"action": map[string]any{
				"primary":     map[string]string{"default": accentText.hex(), "$disabled": subdued.hex()},
				"secondary":   map[string]string{"default": subdued.hex(), "$hovered": p.base05.hex()},
				"destructive": map[string]string{"default": accentText.hex(), "$disabled": subdued.hex()},
			},
			"formfield": map[string]string{"default": p.base05.hex(), "$disabled": subdued.hex()},
			"status": map[string]string{
				"running":    p.accent.hex(),
				"question":   p.base0A.hex(),
				"permission": p.base0A.hex(),
				"unread":     p.base0C.hex(),
			},
			"feedback": map[string]any{
				"error":   map[string]string{"default": p.base08.hex(), "subdued": mix(p.base00, p.base08, 0.75).hex()},
				"warning": map[string]string{"default": p.base0A.hex(), "subdued": mix(p.base00, p.base0A, 0.75).hex()},
				"success": map[string]string{"default": p.base0B.hex(), "subdued": mix(p.base00, p.base0B, 0.75).hex()},
				"info":    map[string]string{"default": p.base0C.hex(), "subdued": mix(p.base00, p.base0C, 0.75).hex()},
			},
		},
		"background": map[string]any{
			"default": p.base00.hex(),
			"surface": map[string]string{"offset": p.base01.hex(), "overlay": p.base02.hex()},
			"action": map[string]any{
				"primary": map[string]string{
					"default":   p.accent.hex(),
					"$hovered":  shade(p.accent, 0.85).hex(),
					"$pressed":  shade(p.accent, 0.7).hex(),
					"$disabled": p.base02.hex(),
				},
				"secondary": map[string]string{"default": "transparent"},
				"destructive": map[string]string{
					"default":   p.base08.hex(),
					"$hovered":  shade(p.base08, 0.85).hex(),
					"$pressed":  shade(p.base08, 0.7).hex(),
					"$disabled": p.base02.hex(),
				},
			},
			"formfield": map[string]string{
				"default":   p.base00.hex(),
				"$hovered":  p.base01.hex(),
				"$focused":  p.accent.hex(),
				"$pressed":  p.accent.hex(),
				"$selected": p.base01.hex(),
				"$disabled": p.base01.hex(),
			},
			"feedback": map[string]any{
				"error":   map[string]string{"default": p.base00.hex()},
				"warning": map[string]string{"default": p.base00.hex()},
				"success": map[string]string{"default": p.base00.hex()},
				"info":    map[string]string{"default": p.base00.hex()},
			},
		},
		"border":    map[string]string{"default": p.base02.hex()},
		"scrollbar": map[string]string{"default": mix(p.base00, p.base05, 0.35).hex()},
		"diff": map[string]any{
			"text": map[string]string{
				"added":      p.base0B.hex(),
				"removed":    p.base08.hex(),
				"context":    muted.hex(),
				"hunkHeader": p.base0E.hex(),
			},
			"background": map[string]string{
				"added":   mix(p.base00, p.base0B, 0.18).hex(),
				"removed": mix(p.base00, p.base08, 0.18).hex(),
				"context": p.base01.hex(),
			},
			"highlight": map[string]string{"added": p.base0B.hex(), "removed": p.base08.hex()},
			"lineNumber": map[string]any{
				"text": muted.hex(),
				"background": map[string]string{
					"added":   mix(p.base00, p.base0B, 0.3).hex(),
					"removed": mix(p.base00, p.base08, 0.3).hex(),
				},
			},
		},
		"syntax": map[string]string{
			"comment":     muted.hex(),
			"keyword":     p.base0E.hex(),
			"function":    p.accent.hex(),
			"variable":    p.base05.hex(),
			"string":      p.base0B.hex(),
			"number":      p.base09.hex(),
			"type":        p.base0A.hex(),
			"operator":    p.base0C.hex(),
			"punctuation": p.base05.hex(),
		},
		"markdown": map[string]string{
			"text":            p.base05.hex(),
			"heading":         p.base0E.hex(),
			"link":            p.accent.hex(),
			"linkText":        p.base0C.hex(),
			"code":            p.base0B.hex(),
			"blockQuote":      muted.hex(),
			"emphasis":        p.base0A.hex(),
			"strong":          p.base05.hex(),
			"horizontalRule":  p.base02.hex(),
			"listItem":        p.accent.hex(),
			"listEnumeration": p.base0C.hex(),
			"image":           p.accent.hex(),
			"imageText":       p.base0C.hex(),
			"codeBlock":       p.base05.hex(),
		},
		"@context:elevated": map[string]any{
			"text": map[string]any{
				"action": map[string]any{
					"primary": map[string]string{"default": accentText.hex()},
				},
			},
			"background": map[string]any{
				"default": p.base01.hex(),
				"action": map[string]any{
					"primary": map[string]string{"default": p.accent.hex(), "$hovered": p.base02.hex()},
				},
			},
		},
		"@context:overlay": map[string]any{
			"text": map[string]any{
				"action": map[string]any{
					"primary": map[string]string{"default": accentText.hex()},
				},
			},
			"background": map[string]any{
				"default": p.base02.hex(),
				"action": map[string]any{
					"primary": map[string]string{"default": p.accent.hex(), "$hovered": p.base01.hex()},
				},
			},
		},
	}
}

func opencodeThemeDoc(theme *pkg.Theme) (map[string]any, error) {
	palette, err := parseOpencodePalette(theme)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"version":  2,
		theme.Type: palette.mode(),
	}, nil
}

// OpencodeThemePath resolves the configured theme path, defaulting to the global opencode
// theme directory. The file name is the theme name opencode selects via theme.name.
func OpencodeThemePath(configured string) (string, error) {
	if configured != "" {
		return configured, nil
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return filepath.Join(homeDir, ".config", "opencode", "themes", "matheme.json"), nil
}

// ApplyOpencodeTheme writes a version 2 opencode theme generated from the base16 palette.
// opencode uses it once cli.json selects the file name via theme.name.
func ApplyOpencodeTheme(theme *pkg.Theme, themePath string) error {
	doc, err := opencodeThemeDoc(theme)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal opencode theme: %w", err)
	}
	data = append(data, '\n')

	themePath, err = OpencodeThemePath(themePath)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(themePath), 0755); err != nil {
		return fmt.Errorf("failed to create opencode theme directory: %w", err)
	}
	if err := os.WriteFile(themePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write opencode theme: %w", err)
	}

	return nil
}

var opencodeSubcommands = map[string]bool{
	"acp": true, "api": true, "auth": true, "debug": true, "mcp": true, "mini": true,
	"models": true, "plugin": true, "run": true, "serve": true, "service": true,
	"session": true, "stats": true, "uninstall": true, "upgrade": true, "update": true,
}

// parseOpencodeTuiProcesses picks the terminal clients out of `ps -Ao pid=,tty=,command=`
// output, leaving out background services and non-interactive subcommands.
func parseOpencodeTuiProcesses(output string) []int {
	var pids []int
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 || fields[1] == "??" {
			continue
		}
		name := filepath.Base(fields[2])
		if name != "opencode" && name != "opencode.exe" {
			continue
		}
		if len(fields) > 3 && opencodeSubcommands[fields[3]] {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		pids = append(pids, pid)
	}
	return pids
}

// ReloadOpencode asks running opencode terminal clients to re-read the theme files. The TUI
// re-discovers custom themes on SIGUSR2, so switching themes does not require a restart.
func ReloadOpencode() error {
	out, err := exec.Command("ps", "-Ao", "pid=,tty=,command=").Output()
	if err != nil {
		return fmt.Errorf("failed to list processes: %w", err)
	}

	for _, pid := range parseOpencodeTuiProcesses(string(out)) {
		process, err := os.FindProcess(pid)
		if err != nil {
			continue
		}
		process.Signal(syscall.SIGUSR2)
	}

	return nil
}
