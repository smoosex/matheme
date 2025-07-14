package cmd

import (
	"fmt"

	"github.com/matheme/cmd/common"

	"github.com/spf13/cobra"
)

var listThemesCmd = &cobra.Command{
	Use:     "list-themes",
	Aliases: []string{"ls"},
	Short:   "List all available themes",
	Run: func(cmd *cobra.Command, args []string) {
		themes := common.ListThemes()
		for _, theme := range themes {
			fmt.Println(theme)
		}
	},
}

func init() {
	rootCmd.AddCommand(listThemesCmd)
}
