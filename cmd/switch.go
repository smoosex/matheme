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

		chezmoi := func() {
			if viper.GetBool("chezmoi.enable") {
				exec.Command("chezmoi", "apply", "--force").Run()
			}
		}

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
			if err := os.Rename(tmpDir+"/alacritty_theme.toml", dst); err != nil {
				panic(fmt.Errorf("failed to rename theme.toml to %s: %w", dst, err))
			}

			chezmoi()

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
			if err := os.Rename(tmpDir+"/sketchybar_theme.lua", dst); err != nil {
				panic(fmt.Errorf("failed to rename init.lua to %s: %w", dst, err))
			}
			chezmoi()
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

		// Ghostty
		if viper.GetBool("ghostty.enable") {
			genGhosttyThemeScript := filepath.Join(scriptsDir, "gen_ghostty_theme.lua")
			if err := exec.Command(
				"lua", genGhosttyThemeScript, curTheme).
				Run(); err != nil {
				panic(fmt.Errorf("failed to run gen ghostty theme: %w", err))
			}

			dst := viper.GetString("ghostty.theme_path")
			if err := os.Rename(tmpDir+"/ghostty_theme", dst); err != nil {
				panic(fmt.Errorf("failed to rename ghostty_theme to %s: %w", dst, err))
			}

			chezmoi()
			exec.Command("pkill", "-SIGUSR2", "ghostty").Run()
		}

		// Kitty
		if viper.GetBool("kitty.enable") {
			genKittyThemeScript := filepath.Join(scriptsDir, "gen_kitty_theme.lua")
			if err := exec.Command(
				"lua", genKittyThemeScript, curTheme).
				Run(); err != nil {
				panic(fmt.Errorf("failed to run gen kitty theme: %w", err))
			}

			dst := viper.GetString("kitty.theme_path")
			if err := os.Rename(tmpDir+"/kitty_theme", dst); err != nil {
				panic(fmt.Errorf("failed to rename kitty_theme to %s: %w", dst, err))
			}

			chezmoi()
			exec.Command("pkill", "-SIGUSR1", "kitty").Run()
		}
	},
}

func init() {
	rootCmd.AddCommand(switchCmd)

	switchCmd.Flags().StringVarP(&curTheme, "theme", "t", "", "Specify theme")
}
