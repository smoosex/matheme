package apply

import (
	"os"
	"strings"
	"testing"

	"github.com/matheme/pkg"
)

func TestStarshipLadder(t *testing.T) {
	for _, name := range []string{"everforest", "everforest_light", "tokyonight", "rosepine", "gruvchad", "tundra", "one_light"} {
		theme, err := pkg.ParseTheme(name)
		if err != nil {
			t.Fatalf("failed to parse theme %s: %v", name, err)
		}
		colors, err := buildStarshipColors(theme)
		if err != nil {
			t.Fatalf("failed to build colors for %s: %v", name, err)
		}
		t.Logf("%-18s s1 %s/%s  s2 %s/%s  s3 %s/%s  s4 %s/%s  s5 %s/%s",
			name, colors.S1, colors.S1FG, colors.S2, colors.S2FG,
			colors.S3, colors.S3FG, colors.S4, colors.S4FG, colors.S5, colors.S5FG)
	}
}

func TestApplyStarshipTheme(t *testing.T) {
	theme, err := pkg.ParseTheme("everforest")
	if err != nil {
		t.Fatalf("failed to parse theme: %v", err)
	}
	if err := ApplyStarshipTheme(theme); err != nil {
		t.Fatalf("failed to apply starship theme: %v", err)
	}

	rendered, err := os.ReadFile("/tmp/matheme/starship_theme.toml")
	if err != nil {
		t.Fatalf("failed to read rendered file: %v", err)
	}
	content := string(rendered)

	colors, err := buildStarshipColors(theme)
	if err != nil {
		t.Fatalf("failed to build colors: %v", err)
	}
	for _, color := range []string{colors.S1, colors.S2, colors.S3, colors.S4, colors.S5} {
		if !strings.Contains(content, color) {
			t.Errorf("rendered config is missing color %s", color)
		}
	}
	if strings.Contains(content, "{{") {
		t.Errorf("rendered config contains unexpanded template placeholders")
	}
}
