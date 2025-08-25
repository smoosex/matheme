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
			fmt.Fprintf(os.Stderr, "theme %s not found\n", curTheme)
			os.Exit(1)
		}

		homeDir := os.Getenv("HOME")
		scriptsDir := filepath.Join(homeDir, ".config", "matheme", "scripts")
		tmpDir := "/tmp/matheme"

		chezmoiApply := func() {
			if viper.GetBool("chezmoi.enable") {
				exec.Command("chezmoi", "apply", "--force").Run()
			}
		}

		// Neovim
		if viper.GetBool("nvim.enable") {
			nvimConfigDir := viper.GetString("nvim.init_path")
			switchNvimDirScript := filepath.Join(scriptsDir, "switch_nvim_theme.lua")
			if err := exec.Command("nvim", "-u", nvimConfigDir, "-l", switchNvimDirScript, "--theme", curTheme).Run(); err != nil {
				fmt.Fprintf(os.Stderr, "failed to run switch nvim: %v\n", err)
				os.Exit(1)
			}
			if viper.GetBool("chezmoi.enable") {
				chadrcPath := viper.GetString("nvim.chadrc_path")
				exec.Command("chezmoi", "add", chadrcPath).Run()
			}
		}

		// Alacritty
		if viper.GetBool("alacritty.enable") {
			genAlacrittyThemeScript := filepath.Join(scriptsDir, "gen_alacritty_theme.lua")
			if err := exec.Command(
				"lua", genAlacrittyThemeScript, curTheme).
				Run(); err != nil {
				fmt.Fprintf(os.Stderr, "failed to run gen alacritty theme: %v\n", err)
				os.Exit(1)
			}

			dst := viper.GetString("alacritty.theme_path")
			if err := os.Rename(tmpDir+"/alacritty_theme.toml", dst); err != nil {
				fmt.Fprintf(os.Stderr, "failed to rename theme.toml to %s: %v\n", dst, err)
				os.Exit(1)
			}

			chezmoiApply()

			now := time.Now()
			if err := os.Chtimes(viper.GetString("alacritty.config_path"), now, now); err != nil {
				fmt.Fprintf(os.Stderr, "failed to update config file timestamp: %v\n", err)
				os.Exit(1)
			}
		}

		// Sketchybar
		if viper.GetBool("sketchybar.enable") {
			genSketchybarThemeScript := filepath.Join(scriptsDir, "gen_sketchybar_theme.lua")
			if err := exec.Command(
				"lua", genSketchybarThemeScript, curTheme).
				Run(); err != nil {
				fmt.Fprintf(os.Stderr, "failed to run gen sketchybar theme: %v\n", err)
				os.Exit(1)
			}
			dst := viper.GetString("sketchybar.theme_path")
			if err := os.Rename(tmpDir+"/sketchybar_theme.lua", dst); err != nil {
				fmt.Fprintf(os.Stderr, "failed to rename init.lua to %s: %v\n", dst, err)
				os.Exit(1)
			}
			chezmoiApply()
		}

		// Switch wallpaper
		if viper.GetBool("wallpaper.auto") {
			switchWallpaperScript := filepath.Join(scriptsDir, "switch_wallpaper.lua")
			curWallpaper := viper.GetString("wallpaper.wallpapers." + curTheme)
			if curWallpaper == "" {
				curWallpaper = viper.GetString("wallpaper.wallpapers.default")
			}
			if curWallpaper == "" {
				fmt.Fprintf(os.Stderr, "wallpaper for theme %s not found\n", curTheme)
				os.Exit(1)
			}
			if err := exec.Command("lua", switchWallpaperScript, curWallpaper).Run(); err != nil {
				fmt.Fprintf(os.Stderr, "failed to run switch wallpaper: %v\n", err)
				os.Exit(1)
			}

		}

		// Ghostty
		if viper.GetBool("ghostty.enable") {
			genGhosttyThemeScript := filepath.Join(scriptsDir, "gen_ghostty_theme.lua")
			if err := exec.Command(
				"lua", genGhosttyThemeScript, curTheme).
				Run(); err != nil {
				fmt.Fprintf(os.Stderr, "failed to run gen ghostty theme: %v\n", err)
				os.Exit(1)
			}

			dst := viper.GetString("ghostty.theme_path")
			if err := os.Rename(tmpDir+"/ghostty_theme", dst); err != nil {
				fmt.Fprintf(os.Stderr, "failed to rename ghostty_theme to %s: %v\n", dst, err)
				os.Exit(1)
			}

			chezmoiApply()
			exec.Command("pkill", "-SIGUSR2", "ghostty").Run()
		}

		// Kitty
		if viper.GetBool("kitty.enable") {
			genKittyThemeScript := filepath.Join(scriptsDir, "gen_kitty_theme.lua")
			if err := exec.Command(
				"lua", genKittyThemeScript, curTheme).
				Run(); err != nil {
				fmt.Fprintf(os.Stderr, "failed to run gen kitty theme: %v\n", err)
				os.Exit(1)
			}

			dst := viper.GetString("kitty.theme_path")
			if err := os.Rename(tmpDir+"/kitty_theme", dst); err != nil {
				fmt.Fprintf(os.Stderr, "failed to rename kitty_theme to %s: %v\n", dst, err)
				os.Exit(1)
			}

			chezmoiApply()
			exec.Command("pkill", "-SIGUSR1", "kitty").Run()
		}

		// MacOS System Appearance Mode
		if viper.GetBool("macos_system_appearance.enable") {
			switchMacOSSystemAppearanceScript := filepath.Join(scriptsDir, "switch_system_appearance.lua")
			if err := exec.Command("lua", switchMacOSSystemAppearanceScript, curTheme).Run(); err != nil {
				fmt.Fprintf(os.Stderr, "failed to run switch macos system appearance: %v\n", err)
				os.Exit(1)
			}
		}

		// Borders
		if viper.GetBool("borders.enable") {
			fp := viper.GetString("borders.file_path")
			switchBordersScript := filepath.Join(scriptsDir, "switch_borders.lua")
			if err := exec.Command("lua", switchBordersScript, fp, curTheme).Run(); err != nil {
				fmt.Fprintf(os.Stderr, "failed to run switch borders: %v\n", err)
				os.Exit(1)
			}

			exec.Command("chezmoi", "add", fp).Run()
			exec.Command("brew", "services", "restart", "borders").Run()
		}
	},
}

func init() {
	rootCmd.AddCommand(switchCmd)

	switchCmd.Flags().StringVarP(&curTheme, "theme", "t", "", "Specify theme")
}
