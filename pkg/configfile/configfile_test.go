package configfile

import (
	"strings"
	"testing"
)

const sample = `# if you don't use chezmoi, you should set enable to false
[chezmoi]
enable = true

[wallpaper]
auto = true

# theme name -> wallpaper name
[wallpaper.wallpapers]
default = "stairs.jpg" # fallback
one_light = "beach.jpg"

[alacritty]
enable = true
config_path = "/Users/smoose/.config/alacritty/alacritty.toml" # triggers reload
theme_path = "/Users/smoose/.config/alacritty/theme.toml"

[sketchybar]
enable = false
theme_path = "/Users/smoose/.config/sketchybar/helpers/theme/palette.lua"

[borders]
enable = false
`

func apply(t *testing.T, changes map[string]string) string {
	t.Helper()
	out, err := Set([]byte(sample), changes)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	return string(out)
}

func TestSetBooleanKeepsCommentAndFormatting(t *testing.T) {
	out := apply(t, map[string]string{"chezmoi.enable": "false"})
	if !strings.Contains(out, "# if you don't use chezmoi, you should set enable to false\n[chezmoi]\nenable = false\n") {
		t.Fatalf("chezmoi.enable was not rewritten in place:\n%s", out)
	}
}

func TestSetOnlyTouchesMatchingSection(t *testing.T) {
	out := apply(t, map[string]string{"alacritty.enable": "false"})
	if !strings.Contains(out, "[chezmoi]\nenable = true") {
		t.Fatalf("chezmoi.enable changed unexpectedly:\n%s", out)
	}
	if !strings.Contains(out, "[sketchybar]\nenable = false") {
		t.Fatalf("sketchybar.enable changed unexpectedly:\n%s", out)
	}
	if !strings.Contains(out, "[alacritty]\nenable = false") {
		t.Fatalf("alacritty.enable was not rewritten:\n%s", out)
	}
}

func TestSetStringKeepsInlineComment(t *testing.T) {
	out := apply(t, map[string]string{"alacritty.config_path": "/opt/alacritty.toml"})
	if !strings.Contains(out, `config_path = "/opt/alacritty.toml" # triggers reload`) {
		t.Fatalf("inline comment was not preserved:\n%s", out)
	}
}

func TestSetWallpaperMapping(t *testing.T) {
	out := apply(t, map[string]string{"wallpaper.wallpapers.default": "mount.jpg"})
	if !strings.Contains(out, `default = "mount.jpg" # fallback`) {
		t.Fatalf("mapping value was not rewritten:\n%s", out)
	}
}

func TestSetAppendsMissingKeyToExistingSection(t *testing.T) {
	out := apply(t, map[string]string{"alacritty.enable": "true", "alacritty.font_size": "13"})
	if !strings.Contains(out, "theme_path = \"/Users/smoose/.config/alacritty/theme.toml\"\nfont_size = 13\n") {
		t.Fatalf("missing key was not appended to its section:\n%s", out)
	}
}

func TestSetAddsMissingSection(t *testing.T) {
	out := apply(t, map[string]string{"herdr.enable": "true"})
	if !strings.HasSuffix(out, "\n[herdr]\nenable = true\n") {
		t.Fatalf("missing section was not appended:\n%q", out)
	}
	if !strings.Contains(out, "[borders]\nenable = false") {
		t.Fatalf("existing content was modified:\n%s", out)
	}
}

func TestSetIgnoresHashInsideQuotes(t *testing.T) {
	out := apply(t, map[string]string{"alacritty.theme_path": "/tmp/a#b.toml"})
	if !strings.Contains(out, `theme_path = "/tmp/a#b.toml"`) {
		t.Fatalf("value with # was not written:\n%s", out)
	}
}

func TestSetRejectsTypeMismatch(t *testing.T) {
	if _, err := Set([]byte(sample), map[string]string{"chezmoi.enable": "maybe"}); err == nil {
		t.Fatal("expected an error for a non-boolean value")
	}
	if _, err := Set([]byte(sample), map[string]string{"wallpaper.auto": "1\n2"}); err == nil {
		t.Fatal("expected an error for a multi-line value")
	}
}

func TestSetRewritesTopLevelDottedKey(t *testing.T) {
	const dotted = "chezmoi.enable = true\n\n[alacritty]\nenable = true\n"
	out, err := Set([]byte(dotted), map[string]string{"chezmoi.enable": "false"})
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	if strings.Contains(string(out), "[chezmoi]") {
		t.Fatalf("a second chezmoi table was created:\n%s", out)
	}
	if !strings.HasPrefix(string(out), "chezmoi.enable = false\n") {
		t.Fatalf("dotted key was not rewritten in place:\n%s", out)
	}
}

func TestSetRefusesValuesItCannotRepresent(t *testing.T) {
	if _, err := Set([]byte("targets = [\"a\"]\n"), map[string]string{"targets": "b"}); err == nil {
		t.Fatal("expected an error for a list value")
	}
	if _, err := Set([]byte("chezmoi = { enable = true }\n"), map[string]string{"chezmoi.enable": "false"}); err == nil {
		t.Fatal("expected an error instead of a document that no longer parses")
	}
}

func TestSetIsIdempotent(t *testing.T) {
	once, err := Set([]byte(sample), map[string]string{"chezmoi.enable": "false", "herdr.enable": "true"})
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	twice, err := Set(once, map[string]string{"chezmoi.enable": "false", "herdr.enable": "true"})
	if err != nil {
		t.Fatalf("second Set failed: %v", err)
	}
	if string(once) != string(twice) {
		t.Fatalf("Set is not idempotent:\nfirst:\n%s\nsecond:\n%s", once, twice)
	}
	if strings.Count(string(once), "[herdr]") != 1 {
		t.Fatalf("duplicate section created:\n%s", once)
	}
}
