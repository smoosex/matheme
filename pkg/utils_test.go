package pkg

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("user home dir: %v", err)
	}

	cases := map[string]string{
		"~/Library/Rime":              filepath.Join(home, "Library", "Rime"),
		"~":                           home,
		"/Users/someone/.config/nvim": "/Users/someone/.config/nvim",
		"relative/path":               "relative/path",
		"":                            "",
		"~someone/elsewhere":          "~someone/elsewhere",
	}

	for input, want := range cases {
		if got := ExpandHome(input); got != want {
			t.Errorf("ExpandHome(%q) = %q, want %q", input, got, want)
		}
	}
}
