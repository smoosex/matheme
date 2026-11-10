package cmd

import (
	"fmt"

	"github.com/matheme/pkg"

	"github.com/spf13/cobra"
)

type currentReport struct {
	Theme  string `json:"theme"`
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
}

var currentCmd = &cobra.Command{
	Use:   "current",
	Short: "Print the active theme",
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := pkg.CurrentThemePath()
		if err != nil {
			return reportError(currentReport{}, "%v", err)
		}

		theme, err := pkg.CurrentTheme()
		if err != nil {
			return reportError(currentReport{Path: path}, "%v", err)
		}

		if jsonOut {
			return emitJSON(currentReport{Theme: theme, Path: path, Exists: theme != ""})
		}

		if theme != "" {
			fmt.Println(theme)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(currentCmd)
}
