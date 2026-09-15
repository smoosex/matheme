package apply

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/matheme/pkg"
)

var herdrTestTheme = &pkg.Theme{
	Type:   "dark",
	Accent: "#c4a7e7",
	Base16: map[string]string{
		"base00": "#191724", "base01": "#1f1d2e", "base02": "#26233a", "base03": "#6e6a86",
		"base04": "#908caa", "base05": "#e0def4", "base06": "#e0def4", "base07": "#524f67",
		"base08": "#eb6f92", "base09": "#f6c177", "base0A": "#ebbcba", "base0B": "#31748f",
		"base0C": "#9ccfd8", "base0D": "#c4a7e7", "base0E": "#f6c177", "base0F": "#524f67",
	},
}

var herdrLightTestTheme = &pkg.Theme{
	Type:   "light",
	Accent: "#4078f2",
	Base16: map[string]string{
		"base00": "#fafafa", "base01": "#f4f4f4", "base02": "#e5e5e6", "base03": "#dfdfe0",
		"base04": "#d7d7d8", "base05": "#383a42", "base06": "#202227", "base07": "#090a0b",
		"base08": "#d84a3d", "base09": "#d75f00", "base0A": "#c18401", "base0B": "#50a14f",
		"base0C": "#0070a8", "base0D": "#4078f2", "base0E": "#a626a4", "base0F": "#986801",
	},
}

type herdrTestDoc struct {
	Theme struct {
		Name       string            `toml:"name"`
		AutoSwitch bool              `toml:"auto_switch"`
		Custom     map[string]string `toml:"custom"`
	} `toml:"theme"`
}

func decodeHerdrBlock(t *testing.T, theme *pkg.Theme) herdrTestDoc {
	t.Helper()

	block, err := herdrThemeBlock(theme)
	if err != nil {
		t.Fatalf("herdrThemeBlock returned error: %v", err)
	}

	var doc herdrTestDoc
	if _, err := toml.Decode(block, &doc); err != nil {
		t.Fatalf("generated block is not valid TOML: %v\n%s", err, block)
	}
	return doc
}

func TestHerdrThemeBlock(t *testing.T) {
	doc := decodeHerdrBlock(t, herdrTestTheme)

	if doc.Theme.Name != "catppuccin" {
		t.Errorf("theme.name = %q, want catppuccin", doc.Theme.Name)
	}
	if doc.Theme.AutoSwitch {
		t.Errorf("theme.auto_switch should be disabled")
	}
	if len(doc.Theme.Custom) != len(herdrThemeTokens) {
		t.Errorf("expected %d custom tokens, got %d", len(herdrThemeTokens), len(doc.Theme.Custom))
	}

	want := map[string]string{
		"accent":        "#c4a7e7",
		"panel_bg":      "#191724",
		"sidebar_bg":    "#1f1d2e",
		"active_row_bg": "#26233a",
		"selection_bg":  "#26233a",
		"overlay0":      "#6e6a86",
		"overlay1":      "#908caa",
		"text":          "#e0def4",
		"subtext0":      "#a3a2b3",
		"mauve":         "#f6c177",
		"green":         "#31748f",
		"yellow":        "#ebbcba",
		"red":           "#eb6f92",
		"blue":          "#c4a7e7",
		"teal":          "#9ccfd8",
		"peach":         "#f6c177",
	}
	for token, value := range want {
		if doc.Theme.Custom[token] != value {
			t.Errorf("custom.%s = %q, want %q", token, doc.Theme.Custom[token], value)
		}
	}
}

func TestHerdrThemeBlockLight(t *testing.T) {
	doc := decodeHerdrBlock(t, herdrLightTestTheme)

	if doc.Theme.Name != "catppuccin-latte" {
		t.Errorf("theme.name = %q, want catppuccin-latte", doc.Theme.Name)
	}

	// one_light base03/base04 are nearly the same as its background, so the dim tokens must
	// fall back to readable blends of the background and foreground.
	bg, err := parseHex(doc.Theme.Custom["panel_bg"])
	if err != nil {
		t.Fatalf("panel_bg is not a hex color: %v", err)
	}
	for _, token := range []string{"overlay0", "overlay1"} {
		color, err := parseHex(doc.Theme.Custom[token])
		if err != nil {
			t.Fatalf("%s is not a hex color: %v", token, err)
		}
		want := 3.0
		if token == "overlay1" {
			want = 4.5
		}
		if contrast(bg, color) < want {
			t.Errorf("%s has contrast %.2f, want at least %.1f", token, contrast(bg, color), want)
		}
	}
	subtext, err := parseHex(doc.Theme.Custom["subtext0"])
	if err != nil {
		t.Fatalf("subtext0 is not a hex color: %v", err)
	}
	if contrast(bg, subtext) < 7 {
		t.Errorf("subtext0 has contrast %.2f, want at least 7", contrast(bg, subtext))
	}
}

func TestHerdrThemeBlockMissingColor(t *testing.T) {
	theme := &pkg.Theme{Type: "dark", Base16: map[string]string{"base00": "#191724"}}
	if _, err := herdrThemeBlock(theme); err == nil {
		t.Fatal("expected an error for a palette without base05")
	}
}

func TestHerdrThemeBlockInvalidType(t *testing.T) {
	theme := &pkg.Theme{Type: "auto", Base16: herdrTestTheme.Base16}
	if _, err := herdrThemeBlock(theme); err == nil {
		t.Fatal("expected an error for an unknown theme type")
	}
}

func TestReplaceHerdrThemeSection(t *testing.T) {
	content := `onboarding = false

[theme]
name = "rose-pine"
auto_switch = false

[theme.custom]
accent = "#ffffff"

[keys]
prefix = "ctrl+a"

[[keys.command]]
key = "prefix+g"
type = "popup"
command = "lazygit"
`
	result := replaceHerdrThemeSection(content, "[theme]\nname = \"catppuccin\"\n")

	if !strings.HasPrefix(result, "onboarding = false\n") {
		t.Errorf("root keys were dropped:\n%s", result)
	}
	for _, kept := range []string{"[keys]", `prefix = "ctrl+a"`, "[[keys.command]]", "lazygit"} {
		if !strings.Contains(result, kept) {
			t.Errorf("expected %q to be preserved:\n%s", kept, result)
		}
	}
	if strings.Contains(result, "rose-pine") || strings.Contains(result, "#ffffff") {
		t.Errorf("stale theme values survived:\n%s", result)
	}
	if strings.Count(result, "[keys]") != 1 {
		t.Errorf("keys table duplicated:\n%s", result)
	}
}

func TestReplaceHerdrThemeSectionAppends(t *testing.T) {
	result := replaceHerdrThemeSection("onboarding = false", "[theme]\nname = \"catppuccin\"\n")
	want := "onboarding = false\n\n[theme]\nname = \"catppuccin\"\n"
	if result != want {
		t.Errorf("replaceHerdrThemeSection = %q, want %q", result, want)
	}
}

func TestReplaceHerdrThemeSectionEmptyConfig(t *testing.T) {
	result := replaceHerdrThemeSection("", "[theme]\nname = \"catppuccin\"\n")
	if result != "[theme]\nname = \"catppuccin\"\n" {
		t.Errorf("unexpected result for empty config: %q", result)
	}
}

func TestReplaceHerdrThemeSectionAtEndOfFile(t *testing.T) {
	content := "onboarding = false\n\n[theme]\nname = \"rose-pine\"\n"
	result := replaceHerdrThemeSection(content, "[theme]\nname = \"catppuccin\"\n")
	if strings.Contains(result, "rose-pine") {
		t.Errorf("stale theme values survived:\n%s", result)
	}
	if !strings.HasSuffix(result, "onboarding = false\n\n[theme]\nname = \"catppuccin\"\n") {
		t.Errorf("unexpected result:\n%s", result)
	}
}

func TestReplaceHerdrThemeSectionKeepsLookalikeTables(t *testing.T) {
	content := "[theme]\nname = \"rose-pine\"\n\n[theme_picker]\nwidth = 40\n"
	result := replaceHerdrThemeSection(content, "[theme]\nname = \"catppuccin\"\n")
	if !strings.Contains(result, "[theme_picker]\nwidth = 40") {
		t.Errorf("theme_picker table was dropped:\n%s", result)
	}
}

func TestReloadHerdr(t *testing.T) {
	cases := []struct {
		name    string
		output  string
		exit    int
		wantErr bool
	}{
		{name: "applied", output: `{"id":"cli:server:reload-config","result":{"diagnostics":[],"status":"applied","type":"config_reload"}}`},
		{name: "no server", output: `{"id":"cli:server:reload-config","error":{"code":"server_not_running","message":"no herdr server is running"}}`, exit: 1},
		{name: "failure", output: `{"id":"cli:server:reload-config","error":{"code":"internal","message":"boom"}}`, exit: 1, wantErr: true},
		{name: "garbage", output: "not json", exit: 1, wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			script := filepath.Join(dir, "herdr")
			body := fmt.Sprintf("#!/bin/sh\nprintf '%%s' '%s'\nexit %d\n", tc.output, tc.exit)
			if err := os.WriteFile(script, []byte(body), 0755); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", dir)

			err := ReloadHerdr()
			if tc.wantErr && err == nil {
				t.Error("expected an error")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}

	t.Run("missing binary", func(t *testing.T) {
		t.Setenv("PATH", t.TempDir())
		if err := ReloadHerdr(); err == nil {
			t.Error("expected an error when herdr is not installed")
		}
	})
}

func TestApplyHerdrTheme(t *testing.T) {
	path := filepath.Join(t.TempDir(), "herdr", "config.toml")
	content := "onboarding = false\n\n[keys]\nprefix = \"ctrl+a\"\n"
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	if err := ApplyHerdrTheme(herdrTestTheme, path); err != nil {
		t.Fatalf("ApplyHerdrTheme returned error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	result := string(data)
	if !strings.HasPrefix(result, "onboarding = false\n") || !strings.Contains(result, "prefix = \"ctrl+a\"") {
		t.Errorf("existing config was not preserved:\n%s", result)
	}
	if !strings.Contains(result, "panel_bg = \"#191724\"") || !strings.Contains(result, "name = \"catppuccin\"") {
		t.Errorf("theme block is missing:\n%s", result)
	}

	var doc herdrTestDoc
	if _, err := toml.Decode(result, &doc); err != nil {
		t.Fatalf("rewritten config is not valid TOML: %v", err)
	}
	if doc.Theme.Name != "catppuccin" || doc.Theme.Custom["text"] != "#e0def4" {
		t.Errorf("unexpected theme after rewrite: %+v", doc.Theme)
	}
}
