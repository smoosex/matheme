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
		nvimConfigDir := filepath.Join(homeDir, ".config", "nvim", "init.lua")
		scriptsDir := filepath.Join(homeDir, ".config", "matheme", "scripts")

		switchNvimDirScript := filepath.Join(scriptsDir, "switch_nvim_theme.lua")
		genAlacrittyThemeScript := filepath.Join(scriptsDir, "gen_alacritty_theme.lua")
		genSketchybarThemeScript := filepath.Join(scriptsDir, "gen_sketchybar_theme.lua")
		switchWallpaperScript := filepath.Join(scriptsDir, "switch_wallpaper.lua")

		tmpDir := "/tmp/matheme"

		// Neovim
		if err := exec.Command("nvim", "-u", nvimConfigDir, "-l", switchNvimDirScript, "--theme", curTheme).Run(); err != nil {
			panic(fmt.Errorf("failed to run switch nvim: %w", err))
		}

		// Alacritty
		if err := exec.Command(
			"lua", genAlacrittyThemeScript, curTheme).
			Run(); err != nil {
			panic(fmt.Errorf("failed to run gen alacritty theme: %w", err))
		}

		dst := viper.GetString("theme.path.alacritty")
		if err := os.Rename(tmpDir+"/theme.toml", dst); err != nil {
			panic(fmt.Errorf("failed to rename theme.toml to %s: %w", dst, err))
		}
		exec.Command("chezmoi", "apply", "--force").Run()
		now := time.Now()
		if err := os.Chtimes(filepath.Join(os.Getenv("HOME"), ".config/alacritty/alacritty.toml"), now, now); err != nil {
			panic(fmt.Errorf("failed to update config file timestamp: %w", err))
		}

		// Sketchybar
		if err := exec.Command(
			"lua", genSketchybarThemeScript, curTheme).
			Run(); err != nil {
			panic(fmt.Errorf("failed to run gen sketchybar theme: %w", err))
		}
		dst = viper.GetString("theme.path.sketchybar")
		if err := os.Rename(tmpDir+"/init.lua", dst); err != nil {
			panic(fmt.Errorf("failed to rename init.lua to %s: %w", dst, err))
		}
		exec.Command("chezmoi", "apply", "--force").Run()

		// Switch wallpaper
		if viper.GetBool("wallpaper.auto") {
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
	},
}

func init() {
	rootCmd.AddCommand(switchCmd)

	switchCmd.Flags().StringVarP(&curTheme, "theme", "t", "", "Specify theme")
}
