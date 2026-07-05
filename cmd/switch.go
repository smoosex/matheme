package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/matheme/cmd/common"
	"github.com/matheme/pkg"
	"github.com/matheme/pkg/apply"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var switchCmd = &cobra.Command{
	Use:     "switch",
	Aliases: []string{"sw"},
	Short:   "Switch themes",
	Long:    `Switch to specified theme.`,
	Run: func(cmd *cobra.Command, args []string) {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to get user home directory: %v\n", err)
			os.Exit(1)
		}
		stateFilePath := filepath.Join(homeDir, ".config", "matheme", "current_theme")

		var recordedTheme string
		if content, err := os.ReadFile(stateFilePath); err == nil {
			recordedTheme = strings.TrimSpace(string(content))
		} else if !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "failed to read state file %s: %v\n", stateFilePath, err)
			os.Exit(1)
		}

		if curTheme == recordedTheme {
			fmt.Printf("Theme '%s' is already active. No action taken.\n", curTheme)
			os.Exit(0)
		}

		// Check if theme exists
		themes := common.ListThemes()
		if !pkg.Contains(themes, curTheme) {
			fmt.Fprintf(os.Stderr, "theme %s not found\n", curTheme)
			os.Exit(1)
		}

		tmpDir := "/tmp/matheme"
		theme, err := pkg.ParseTheme(curTheme)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to parse theme %s: %v\n", curTheme, err)
			os.Exit(1)
		}

		var wg sync.WaitGroup
		var mu sync.Mutex
		errs := make(chan error, 13)

		chezmoiFiles := make([]string, 0)
		chezmoiFiles = append(chezmoiFiles, "add")

		addChezmoiFiles := func(path string) {
			if viper.GetBool("chezmoi.enable") {
				mu.Lock()
				defer mu.Unlock()
				chezmoiFiles = append(chezmoiFiles, path)
			}
		}

		// Neovim
		if viper.GetBool("neovim.enable") {
			wg.Add(1)
			go func() {
				defer wg.Done()
				initPath := viper.GetString("neovim.init_path")
				chadrcPath := viper.GetString("neovim.chadrc_path")
				if err := apply.SwitchNvimThemeInGo(curTheme, initPath, chadrcPath); err != nil {
					errs <- fmt.Errorf("failed to switch nvim theme: %v", err)
					return
				}
				addChezmoiFiles(chadrcPath)
			}()
		}

		// Alacritty
		if viper.GetBool("alacritty.enable") {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := apply.ApplyAlacrittyTheme(theme); err != nil {
					errs <- fmt.Errorf("failed to apply alacritty theme: %v", err)
					return
				}
				dst := viper.GetString("alacritty.theme_path")
				if err := os.Rename(tmpDir+"/alacritty_theme.toml", dst); err != nil {
					errs <- fmt.Errorf("failed to rename theme.toml to %s: %v", dst, err)
					return
				}
				now := time.Now()
				if err := os.Chtimes(viper.GetString("alacritty.config_path"), now, now); err != nil {
					errs <- fmt.Errorf("failed to update alacritty config file timestamp: %v", err)
					return
				}
				addChezmoiFiles(dst)
			}()
		}

		// Sketchybar
		if viper.GetBool("sketchybar.enable") {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := apply.ApplySketchybarTheme(theme); err != nil {
					errs <- fmt.Errorf("failed to apply sketchybar theme: %v", err)
					return
				}
				dst := viper.GetString("sketchybar.theme_path")
				if err := os.Rename(tmpDir+"/sketchybar_theme.lua", dst); err != nil {
					errs <- fmt.Errorf("failed to rename init.lua to %s: %v", dst, err)
					return
				}
				if err := exec.Command("sketchybar", "--reload").Run(); err != nil {
					errs <- fmt.Errorf("failed to reload sketchybar: %v", err)
					return
				}
				addChezmoiFiles(dst)
			}()
		}

		// Switch wallpaper
		if viper.GetBool("wallpaper.auto") {
			wg.Add(1)
			go func() {
				defer wg.Done()
				curWallpaper := viper.GetString("wallpaper.wallpapers." + curTheme)
				if curWallpaper == "" {
					curWallpaper = viper.GetString("wallpaper.wallpapers.default")
				}
				if curWallpaper == "" {
					errs <- fmt.Errorf("wallpaper for theme %s not found", curTheme)
					return
				}
				homeDir, _ := os.UserHomeDir() // 使用 UserHomeDir() 更可靠
				wallpaperDir := filepath.Join(homeDir, ".config", "matheme", "wallpaper")
				wallpaperPath := filepath.Join(wallpaperDir, curWallpaper)
				if err := apply.ApplyWallpaper(wallpaperPath); err != nil {
					errs <- fmt.Errorf("failed to apply wallpaper: %v", err)
				}
			}()
		}

		// Ghostty
		if viper.GetBool("ghostty.enable") {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := apply.ApplyGhosttyTheme(theme); err != nil {
					errs <- fmt.Errorf("failed to apply ghostty theme: %v", err)
					return
				}
				dst := viper.GetString("ghostty.theme_path")
				if err := os.Rename(tmpDir+"/ghostty_theme", dst); err != nil {
					errs <- fmt.Errorf("failed to rename ghostty_theme to %s: %v", dst, err)
					return
				}
				exec.Command("pkill", "-SIGUSR2", "ghostty").Run()
				addChezmoiFiles(dst)
			}()
		}

		// Kitty
		if viper.GetBool("kitty.enable") {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := apply.ApplyKittyTheme(theme); err != nil {
					errs <- fmt.Errorf("failed to apply kitty theme: %v", err)
					return
				}
				dst := viper.GetString("kitty.theme_path")
				if err := os.Rename(tmpDir+"/kitty_theme", dst); err != nil {
					errs <- fmt.Errorf("failed to rename kitty_theme to %s: %v", dst, err)
					return
				}
				exec.Command("pkill", "-SIGUSR1", "kitty").Run()
				addChezmoiFiles(dst)
			}()
		}

		// Tmux
		if viper.GetBool("tmux.enable") {
			wg.Add(1)
			go func() {
				defer wg.Done()
				scriptPath := viper.GetString("tmux.switch_script_path")
				if err := apply.ApplyTmuxTheme(scriptPath, curTheme); err != nil {
					errs <- fmt.Errorf("failed to apply tmux theme: %v", err)
					return
				}

				currentThemePath := viper.GetString("tmux.current_theme_path")
				if currentThemePath != "" {
					addChezmoiFiles(currentThemePath)
				}
			}()
		}

		// MacOS System Appearance Mode
		if viper.GetBool("macos_system_appearance.enable") {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := apply.ApplySystemAppearance(theme); err != nil {
					errs <- fmt.Errorf("failed to apply macos system appearance: %v", err)
				}
			}()
		}

		// Pi
		if viper.GetBool("pi.enable") {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := apply.ApplyPiTheme(curTheme, viper.GetString("pi.control_file_path")); err != nil {
					errs <- fmt.Errorf("failed to apply pi theme: %v", err)
				}
			}()
		}

		// Rime (Squirrel)
		if viper.GetBool("rime.enable") {
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := apply.ApplyRimeTheme(theme, viper.GetString("rime.config_dir"))
				if err != nil {
					errs <- fmt.Errorf("failed to apply rime theme: %v", err)
					return
				}
				if err := apply.ReloadSquirrel(); err != nil {
					errs <- fmt.Errorf("failed to reload squirrel: %v", err)
					return
				}
			}()
		}

		// Borders
		if viper.GetBool("borders.enable") {
			wg.Add(1)
			go func() {
				defer wg.Done()
				newColor := strings.TrimPrefix(theme.Base16["base08"], "#")
				acConfig := fmt.Sprintf("active_color=0xff%s", newColor)
				if err := exec.Command("borders", acConfig).Run(); err != nil {
					errs <- fmt.Errorf("failed to apply borders theme: %v", err)
					os.Exit(1)
				}
			}()
		}

		wg.Wait()
		close(errs)

		var hasErrors bool
		for err := range errs {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			hasErrors = true
		}
		if hasErrors {
			fmt.Fprintln(os.Stderr, "One or more tasks failed. Aborting.")
			os.Exit(1)
		}

		configDir := filepath.Dir(stateFilePath)
		if err := os.MkdirAll(configDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "failed to create config directory %s: %v\n", configDir, err)
			os.Exit(1)
		}
		if err := os.WriteFile(stateFilePath, []byte(curTheme), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "failed to write state file %s: %v\n", stateFilePath, err)
		}

		fmt.Println("Record current theme over...")

		if len(chezmoiFiles) > 1 {
			fmt.Println("Applying changes with chezmoi...")
			if err := exec.Command("chezmoi", chezmoiFiles...).Run(); err != nil {
				fmt.Fprintf(os.Stderr, "failed to run chezmoi: %v\n", err)
				os.Exit(1)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(switchCmd)

	switchCmd.Flags().StringVarP(&curTheme, "theme", "t", "", "Specify theme")
}
