package apply

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/matheme/pkg"
)

var opencodeTestTheme = &pkg.Theme{
	Type: "dark",
	Base16: map[string]string{
		"base00": "#191724", "base01": "#1f1d2e", "base02": "#26233a", "base03": "#6e6a86",
		"base04": "#908caa", "base05": "#e0def4", "base06": "#e0def4", "base07": "#524f67",
		"base08": "#eb6f92", "base09": "#f6c177", "base0A": "#ebbcba", "base0B": "#31748f",
		"base0C": "#9ccfd8", "base0D": "#c4a7e7", "base0E": "#f6c177", "base0F": "#524f67",
	},
}

func TestOpencodeThemeDoc(t *testing.T) {
	doc, err := opencodeThemeDoc(opencodeTestTheme)
	if err != nil {
		t.Fatalf("opencodeThemeDoc returned error: %v", err)
	}

	if doc["version"] != 2 {
		t.Errorf("expected version 2, got %v", doc["version"])
	}
	if _, ok := doc["dark"]; !ok {
		t.Fatalf("dark theme is missing: %v", doc)
	}
	if _, ok := doc["light"]; ok {
		t.Errorf("dark theme should not declare a light mode")
	}

	dark, ok := doc["dark"].(map[string]any)
	if !ok {
		t.Fatalf("dark mode is not an object")
	}
	text := dark["text"].(map[string]any)
	background := dark["background"].(map[string]any)
	syntax := dark["syntax"].(map[string]string)

	if text["default"] != "#e0def4" {
		t.Errorf("text.default = %v, want base05", text["default"])
	}
	if background["default"] != "#191724" {
		t.Errorf("background.default = %v, want base00", background["default"])
	}
	if syntax["keyword"] != "#f6c177" {
		t.Errorf("syntax.keyword = %v, want base0E", syntax["keyword"])
	}

	// base03 of the rosepine palette is readable, so it must be kept as the comment color.
	if syntax["comment"] != "#6e6a86" {
		t.Errorf("syntax.comment = %v, want base03", syntax["comment"])
	}

	scale := dark["hue"].(map[string]any)["blue"].(map[string]string)
	if len(scale) != 9 {
		t.Errorf("hue scale has %d steps, want 9", len(scale))
	}
	if scale["500"] != "#c4a7e7" {
		t.Errorf("hue scale step 500 = %v, want base0D", scale["500"])
	}
}

func TestOpencodeThemeDocLight(t *testing.T) {
	light := *opencodeTestTheme
	light.Type = "light"
	light.Base16 = map[string]string{
		"base00": "#fafafa", "base01": "#f4f4f4", "base02": "#e5e5e6", "base03": "#dfdfe0",
		"base04": "#d7d7d8", "base05": "#383a42", "base06": "#202227", "base07": "#090a0b",
		"base08": "#d84a3d", "base09": "#d75f00", "base0A": "#c18401", "base0B": "#50a14f",
		"base0C": "#0070a8", "base0D": "#4078f2", "base0E": "#a626a4", "base0F": "#986801",
	}

	doc, err := opencodeThemeDoc(&light)
	if err != nil {
		t.Fatalf("opencodeThemeDoc returned error: %v", err)
	}
	if _, ok := doc["light"]; !ok {
		t.Fatalf("light theme is missing: %v", doc)
	}

	mode := doc["light"].(map[string]any)
	syntax := mode["syntax"].(map[string]string)
	text := mode["text"].(map[string]any)

	if syntax["comment"] == "#dfdfe0" {
		t.Errorf("syntax.comment must not reuse the washed out base03")
	}
	if syntax["comment"] == text["default"] {
		t.Errorf("syntax.comment must be dimmer than the default text")
	}
}

func TestOpencodeThemeDocRejectsIncompleteTheme(t *testing.T) {
	if _, err := opencodeThemeDoc(&pkg.Theme{Type: "dark", Base16: map[string]string{"base00": "#000000"}}); err == nil {
		t.Fatal("expected an error for an incomplete palette")
	}
	if _, err := opencodeThemeDoc(&pkg.Theme{Type: "blue", Base16: opencodeTestTheme.Base16}); err == nil {
		t.Fatal("expected an error for an unknown theme type")
	}
}

func TestApplyOpencodeThemeWritesJSON(t *testing.T) {
	themePath := filepath.Join(t.TempDir(), "themes", "matheme.json")
	if err := ApplyOpencodeTheme(opencodeTestTheme, themePath); err != nil {
		t.Fatalf("ApplyOpencodeTheme returned error: %v", err)
	}

	data, err := os.ReadFile(themePath)
	if err != nil {
		t.Fatalf("failed to read generated theme: %v", err)
	}

	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("generated theme is not valid JSON: %v", err)
	}
	if doc["version"] != float64(2) {
		t.Errorf("expected version 2, got %v", doc["version"])
	}
}

func TestParseOpencodeTuiProcesses(t *testing.T) {
	ps := ` 50537 ttys007 opencode
 96974 ??      /opt/homebrew/lib/node_modules/@opencode/cli/bin/opencode.exe serve --service
 12345 ttys002 opencode run "explain this"
 54321 ttys003 opencode --standalone
 22222 ttys004 /opt/homebrew/bin/opencode.exe -c
 33333 ttys005 /opt/homebrew/bin/opencode.exe serve
 44444 ttys006 /opt/homebrew/bin/opencode mini
 66666 ??      opencode serve
 77777 ttys008 htop
`

	pids := parseOpencodeTuiProcesses(ps)
	want := []int{50537, 54321, 22222}
	if len(pids) != len(want) {
		t.Fatalf("parseOpencodeTuiProcesses = %v, want %v", pids, want)
	}
	for i, pid := range want {
		if pids[i] != pid {
			t.Errorf("pid[%d] = %d, want %d", i, pids[i], pid)
		}
	}
}

func TestOpencodeThemePath(t *testing.T) {
	path, err := OpencodeThemePath("/tmp/custom.json")
	if err != nil || path != "/tmp/custom.json" {
		t.Errorf("OpencodeThemePath = %q, %v; want the configured path", path, err)
	}

	path, err = OpencodeThemePath("")
	if err != nil {
		t.Fatalf("OpencodeThemePath returned error: %v", err)
	}
	if want := filepath.Join(".config", "opencode", "themes", "matheme.json"); !strings.HasSuffix(path, want) {
		t.Errorf("OpencodeThemePath = %q, want a path ending in %q", path, want)
	}
}

func TestApplyOpencodeThemeForRepoThemes(t *testing.T) {
	matches, err := filepath.Glob("../../themes/*.toml")
	if err != nil || len(matches) == 0 {
		t.Skipf("no repo themes found: %v", err)
	}

	for _, path := range matches {
		var theme pkg.Theme
		if _, err := toml.DecodeFile(path, &theme); err != nil {
			t.Fatalf("failed to decode %s: %v", path, err)
		}
		if _, err := opencodeThemeDoc(&theme); err != nil {
			t.Errorf("%s: %v", path, err)
		}
	}
}
