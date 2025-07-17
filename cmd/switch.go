package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/matheme/cmd/common"
	"github.com/matheme/pkg"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var switchCmd = &cobra.Command{
	Use:     "switch",
	Aliases: []string{"sw"},
	Short:   "Switch themes",
	Long:    `Switch to specified theme.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Check if theme exists
		themes := common.ListThemes()
		if !pkg.Contains(themes, curTheme) {
			panic(fmt.Errorf("theme %s not found", curTheme))
		}

		homeDir := os.Getenv("HOME")
		scriptsDir := filepath.Join(homeDir, ".config", "matheme", "scripts")
		tmpDir := "/tmp/matheme"

		// Neovim
		if viper.GetBool("nvim.enable") {
			nvimConfigDir := viper.GetString("nvim.init_path")
			switchNvimDirScript := filepath.Join(scriptsDir, "switch_nvim_theme.lua")
			if err := exec.Command("nvim", "-u", nvimConfigDir, "-l", switchNvimDirScript, "--theme", curTheme).Run(); err != nil {
				panic(fmt.Errorf("failed to run switch nvim: %w", err))
			}
		}

		// Alacritty
		if viper.GetBool("alacritty.enable") {
			genAlacrittyThemeScript := filepath.Join(scriptsDir, "gen_alacritty_theme.lua")
			if err := exec.Command(
				"lua", genAlacrittyThemeScript, curTheme).
				Run(); err != nil {
				panic(fmt.Errorf("failed to run gen alacritty theme: %w", err))
			}

			dst := viper.GetString("alacritty.theme_path")
			if err := os.Rename(tmpDir+"/theme.toml", dst); err != nil {
				panic(fmt.Errorf("failed to rename theme.toml to %s: %w", dst, err))
			}
			now := time.Now()
			if err := os.Chtimes(viper.GetString("alacritty.config_path"), now, now); err != nil {
				panic(fmt.Errorf("failed to update config file timestamp: %w", err))
			}
		}

		// Sketchybar
		if viper.GetBool("sketchybar.enable") {
			genSketchybarThemeScript := filepath.Join(scriptsDir, "gen_sketchybar_theme.lua")
			if err := exec.Command(
				"lua", genSketchybarThemeScript, curTheme).
				Run(); err != nil {
				panic(fmt.Errorf("failed to run gen sketchybar theme: %w", err))
			}
			dst := viper.GetString("sketchybar.theme_path")
			if err := os.Rename(tmpDir+"/init.lua", dst); err != nil {
				panic(fmt.Errorf("failed to rename init.lua to %s: %w", dst, err))
			}
		}

		// Switch wallpaper
		if viper.GetBool("wallpaper.auto") {
			switchWallpaperScript := filepath.Join(scriptsDir, "switch_wallpaper.lua")
			curWallpaper := viper.GetString("wallpaper.wallpapers." + curTheme)
			if curWallpaper == "" {
				curWallpaper = viper.GetString("wallpaper.wallpapers.default")
			}
			if curWallpaper == "" {
				panic(fmt.Errorf("wallpaper for theme %s not found", curTheme))
			}
			if err := exec.Command("lua", switchWallpaperScript, curWallpaper).Run(); err != nil {
				panic(fmt.Errorf("failed to run switch wallpaper: %w", err))
			}

		}

		// Chezmoi
		if viper.GetBool("chezmoi.enable") {
			exec.Command("chezmoi", "apply", "--force").Run()
		}

	},
}

func init() {
	rootCmd.AddCommand(switchCmd)

	switchCmd.Flags().StringVarP(&curTheme, "theme", "t", "", "Specify theme")
}
