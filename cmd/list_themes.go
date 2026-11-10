package cmd

import (
	"fmt"

	"github.com/matheme/pkg"

	"github.com/spf13/cobra"
)

type themeEntry struct {
	Name    string            `json:"name"`
	Type    string            `json:"type"`
	Accent  string            `json:"accent"`
	Path    string            `json:"path"`
	Current bool              `json:"current"`
	Base16  map[string]string `json:"base16"`
	Error   string            `json:"error,omitempty"`
}

var listThemesCmd = &cobra.Command{
	Use:     "list-themes",
	Aliases: []string{"ls"},
	Short:   "List all available themes",
	RunE: func(cmd *cobra.Command, args []string) error {
		entries, err := themeEntries()
		if err != nil {
			return reportError(map[string]any{"ok": false, "error": err.Error()}, "%v", err)
		}

		if jsonOut {
			return emitJSON(entries)
		}

		for _, entry := range entries {
			fmt.Println(entry.Name)
		}
		return nil
	},
}

// themeEntries reads every theme definition plus the recorded state. A broken theme file is
// reported per entry instead of failing the whole list.
func themeEntries() ([]themeEntry, error) {
	names, err := pkg.ListThemeNames()
	if err != nil {
		return nil, err
	}

	current, err := pkg.CurrentTheme()
	if err != nil {
		return nil, err
	}

	entries := make([]themeEntry, 0, len(names))
	for _, name := range names {
		path, err := pkg.ThemePath(name)
		if err != nil {
			return nil, err
		}
		entry := themeEntry{Name: name, Path: path, Current: name == current}

		theme, err := pkg.ParseTheme(name)
		if err != nil {
			entry.Error = err.Error()
			entries = append(entries, entry)
			continue
		}

		entry.Type = theme.Type
		entry.Accent = theme.Accent
		entry.Base16 = theme.Base16
		entries = append(entries, entry)
	}

	return entries, nil
}

func init() {
	rootCmd.AddCommand(listThemesCmd)
}
