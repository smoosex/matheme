package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/matheme/pkg"
	"github.com/matheme/pkg/apply"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type targetResult struct {
	Name    string   `json:"name"`
	OK      bool     `json:"ok"`
	Skipped bool     `json:"skipped"`
	Error   string   `json:"error,omitempty"`
	files   []string // paths to hand to chezmoi, empty when nothing was generated
}

type chezmoiReport struct {
	Enabled bool   `json:"enabled"`
	OK      bool   `json:"ok"`
	Skipped bool   `json:"skipped"`
	Files   int    `json:"files"`
	Error   string `json:"error,omitempty"`
}

type switchReport struct {
	OK           bool           `json:"ok"`
	Theme        string         `json:"theme"`
	Changed      bool           `json:"changed"`
	DryRun       bool           `json:"dryRun,omitempty"`
	Planned      []string       `json:"planned,omitempty"`
	StateWritten bool           `json:"stateWritten"`
	Targets      []targetResult `json:"targets"`
	Chezmoi      chezmoiReport  `json:"chezmoi"`
	Error        string         `json:"error,omitempty"`
}

type switchTarget struct {
	name      string
	enableKey string
	run       func(tmpDir string) ([]string, error)
}

// switchRun collects one result per target. Targets run in parallel, results stay ordered.
type switchRun struct {
	mu      sync.Mutex
	wg      sync.WaitGroup
	order   []string
	results map[string]*targetResult
}

func newSwitchRun() *switchRun {
	return &switchRun{results: make(map[string]*targetResult)}
}

func (r *switchRun) add(name string, enabled bool, fn func() ([]string, error)) {
	result := &targetResult{Name: name, OK: true, Skipped: !enabled}
	r.order = append(r.order, name)
	r.results[name] = result
	if !enabled {
		return
	}

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		files, err := fn()

		r.mu.Lock()
		defer r.mu.Unlock()
		// Files a target managed to write are kept even when the target fails: the chezmoi sync
		// records what is actually on disk, not only the steps that went all the way through.
		result.files = files
		if err != nil {
			result.OK = false
			result.Error = err.Error()
		}
	}()
}

func (r *switchRun) wait() []targetResult {
	r.wg.Wait()

	results := make([]targetResult, 0, len(r.order))
	for _, name := range r.order {
		results = append(results, *r.results[name])
	}
	return results
}

func processRunning(name string) bool {
	return exec.Command("pgrep", "-x", name).Run() == nil
}

var dryRun bool
var force bool

var switchCmd = &cobra.Command{
	Use:     "switch",
	Aliases: []string{"sw"},
	Short:   "Switch themes",
	Long:    `Switch to the specified theme and report every target separately.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if curTheme == "" {
			return failSwitch("theme is required, use -t <theme>")
		}

		names, err := pkg.ListThemeNames()
		if err != nil {
			return failSwitch("%v", err)
		}
		if !pkg.Contains(names, curTheme) {
			return failSwitch("theme %s not found", curTheme)
		}

		theme, err := pkg.ParseTheme(curTheme)
		if err != nil {
			return failSwitch("failed to parse theme %s: %v", curTheme, err)
		}

		recorded, err := pkg.CurrentTheme()
		if err != nil {
			return failSwitch("%v", err)
		}

		targets := buildTargets(theme)

		if curTheme == recorded && !force {
			report := switchReport{
				OK:           true,
				Theme:        curTheme,
				Changed:      false,
				StateWritten: true,
				Targets:      []targetResult{},
				Chezmoi:      chezmoiReport{Enabled: viper.GetBool("chezmoi.enable"), OK: true, Skipped: true},
			}
			if jsonOut {
				return emitJSON(report)
			}
			fmt.Printf("Theme '%s' is already active. No action taken.\n", curTheme)
			return nil
		}

		if dryRun {
			planned := make([]string, 0, len(targets))
			for _, target := range targets {
				if viper.GetBool(target.enableKey) {
					planned = append(planned, target.name)
				}
			}

			chezmoiEnabled := viper.GetBool("chezmoi.enable")
			report := switchReport{
				OK:      true,
				Theme:   curTheme,
				Changed: true,
				DryRun:  true,
				Planned: planned,
				Targets: []targetResult{},
				Chezmoi: chezmoiReport{Enabled: chezmoiEnabled, OK: true, Skipped: !chezmoiEnabled},
			}
			if jsonOut {
				return emitJSON(report)
			}
			fmt.Printf("would switch to %s and run: %s\n", curTheme, strings.Join(planned, ", "))
			return nil
		}

		release, err := pkg.AcquireLock()
		if err != nil {
			return failSwitch("%v", err)
		}
		defer release()

		tmpDir, err := os.MkdirTemp("", "matheme-")
		if err != nil {
			return failSwitch("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		run := newSwitchRun()
		for _, target := range targets {
			target := target
			run.add(target.name, viper.GetBool(target.enableKey), func() ([]string, error) {
				return target.run(tmpDir)
			})
		}

		results := run.wait()

		ok := true
		applied := 0
		files := make([]string, 0, len(results))
		for _, result := range results {
			if !result.Skipped && !result.OK {
				ok = false
			}
			if !result.Skipped && result.OK {
				applied++
			}
			files = append(files, result.files...)
		}

		chezmoi := syncChezmoi(files)
		if !chezmoi.OK {
			ok = false
		}

		report := switchReport{
			OK:           ok,
			Theme:        curTheme,
			Changed:      true,
			StateWritten: false,
			Targets:      results,
			Chezmoi:      chezmoi,
		}

		// The state file records what is actually on disk, not whether every target was happy:
		// one failed target or a failed chezmoi sync must not leave the UI claiming the old theme
		// is still active. Only a run where nothing landed leaves it untouched, so the next attempt
		// retries from scratch (--force re-applies a theme that is already recorded).
		if applied > 0 {
			if err := pkg.WriteCurrentTheme(curTheme); err != nil {
				report.Error = err.Error()
				report.OK = false
				ok = false
			} else {
				report.StateWritten = true
			}
		}

		// Nothing ran at all: every target is disabled and chezmoi had nothing to sync. Claiming
		// success here would report a theme that never left the old one on disk.
		if applied == 0 && chezmoi.Skipped {
			report.OK = false
			report.Error = "no target is enabled, nothing was applied"
			ok = false
		}

		if jsonOut {
			if err := emitJSON(report); err != nil {
				return err
			}
		} else {
			reportHuman(report)
		}

		if !ok {
			return silentExit(1)
		}
		return nil
	},
}

func switchFailure(message string) switchReport {
	return switchReport{
		OK:      false,
		Theme:   curTheme,
		Targets: []targetResult{},
		Error:   message,
	}
}

func failSwitch(format string, args ...any) error {
	message := fmt.Sprintf(format, args...)
	return reportError(switchFailure(message), "%s", message)
}

// pathSetting reads a configured path with a leading ~ expanded, so the tilde form used in the
// Raycast preferences and in hand-edited configs behaves like an absolute path.
func pathSetting(key string) string {
	return pkg.ExpandHome(viper.GetString(key))
}

func buildTargets(theme *pkg.Theme) []switchTarget {
	return []switchTarget{
		{"neovim", "neovim.enable", func(tmpDir string) ([]string, error) {
			initPath := pathSetting("neovim.init_path")
			chadrcPath := pathSetting("neovim.chadrc_path")
			if err := apply.SwitchNvimThemeInGo(curTheme, initPath, chadrcPath); err != nil {
				return nil, fmt.Errorf("failed to switch nvim theme: %w", err)
			}
			return []string{chadrcPath}, nil
		}},
		{"alacritty", "alacritty.enable", func(tmpDir string) ([]string, error) {
			if err := apply.ApplyAlacrittyTheme(theme, tmpDir); err != nil {
				return nil, fmt.Errorf("failed to apply alacritty theme: %w", err)
			}
			dst := pathSetting("alacritty.theme_path")
			if err := os.Rename(filepath.Join(tmpDir, "alacritty_theme.toml"), dst); err != nil {
				return nil, fmt.Errorf("failed to write %s: %w", dst, err)
			}
			now := time.Now()
			if err := os.Chtimes(pathSetting("alacritty.config_path"), now, now); err != nil {
				return nil, fmt.Errorf("failed to update alacritty config timestamp: %w", err)
			}
			return []string{dst}, nil
		}},
		{"sketchybar", "sketchybar.enable", func(tmpDir string) ([]string, error) {
			barPath, err := exec.LookPath("sketchybar")
			if err != nil {
				return nil, fmt.Errorf("sketchybar is enabled but not installed: %w", err)
			}
			if err := apply.ApplySketchybarTheme(theme, tmpDir); err != nil {
				return nil, fmt.Errorf("failed to apply sketchybar theme: %w", err)
			}
			dst := pathSetting("sketchybar.theme_path")
			if err := os.Rename(filepath.Join(tmpDir, "sketchybar_theme.lua"), dst); err != nil {
				return nil, fmt.Errorf("failed to write %s: %w", dst, err)
			}
			// A reload error only means something while sketchybar is actually running; when it
			// is not, the palette written above is picked up the next time it starts.
			if err := exec.Command(barPath, "--reload").Run(); err != nil && processRunning("sketchybar") {
				return []string{dst}, fmt.Errorf("failed to reload sketchybar: %w", err)
			}
			return []string{dst}, nil
		}},
		{"wallpaper", "wallpaper.auto", func(tmpDir string) ([]string, error) {
			name := viper.GetString("wallpaper.wallpapers." + curTheme)
			if name == "" {
				name = viper.GetString("wallpaper.wallpapers.default")
			}
			if name == "" {
				return nil, fmt.Errorf("wallpaper for theme %s not found", curTheme)
			}

			homeDir, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("failed to get user home directory: %w", err)
			}
			path := filepath.Join(homeDir, ".config", "matheme", "wallpaper", name)
			if err := apply.ApplyWallpaper(path); err != nil {
				return nil, fmt.Errorf("failed to apply wallpaper: %w", err)
			}
			return nil, nil
		}},
		{"ghostty", "ghostty.enable", func(tmpDir string) ([]string, error) {
			if err := apply.ApplyGhosttyTheme(theme, tmpDir); err != nil {
				return nil, fmt.Errorf("failed to apply ghostty theme: %w", err)
			}
			dst := pathSetting("ghostty.theme_path")
			if err := os.Rename(filepath.Join(tmpDir, "ghostty_theme"), dst); err != nil {
				return nil, fmt.Errorf("failed to write %s: %w", dst, err)
			}
			exec.Command("pkill", "-SIGUSR2", "ghostty").Run()
			return []string{dst}, nil
		}},
		{"kitty", "kitty.enable", func(tmpDir string) ([]string, error) {
			if err := apply.ApplyKittyTheme(theme, tmpDir); err != nil {
				return nil, fmt.Errorf("failed to apply kitty theme: %w", err)
			}
			dst := pathSetting("kitty.theme_path")
			if err := os.Rename(filepath.Join(tmpDir, "kitty_theme"), dst); err != nil {
				return nil, fmt.Errorf("failed to write %s: %w", dst, err)
			}
			exec.Command("pkill", "-SIGUSR1", "kitty").Run()
			return []string{dst}, nil
		}},
		{"starship", "starship.enable", func(tmpDir string) ([]string, error) {
			if err := apply.ApplyStarshipTheme(theme, tmpDir); err != nil {
				return nil, fmt.Errorf("failed to apply starship theme: %w", err)
			}
			dst := pathSetting("starship.config_path")
			if err := os.Rename(filepath.Join(tmpDir, "starship_theme.toml"), dst); err != nil {
				return nil, fmt.Errorf("failed to write %s: %w", dst, err)
			}
			return []string{dst}, nil
		}},
		{"tmux", "tmux.enable", func(tmpDir string) ([]string, error) {
			if err := apply.ApplyTmuxTheme(pathSetting("tmux.switch_script_path"), curTheme); err != nil {
				return nil, fmt.Errorf("failed to apply tmux theme: %w", err)
			}
			if path := pathSetting("tmux.current_theme_path"); path != "" {
				return []string{path}, nil
			}
			return nil, nil
		}},
		{"macos-system-appearance", "macos_system_appearance.enable", func(tmpDir string) ([]string, error) {
			if err := apply.ApplySystemAppearance(theme); err != nil {
				return nil, fmt.Errorf("failed to apply macos system appearance: %w", err)
			}
			return nil, nil
		}},
		{"pi", "pi.enable", func(tmpDir string) ([]string, error) {
			if err := apply.ApplyPiTheme(curTheme, pathSetting("pi.control_file_path")); err != nil {
				return nil, fmt.Errorf("failed to apply pi theme: %w", err)
			}
			return nil, nil
		}},
		{"opencode", "opencode.enable", func(tmpDir string) ([]string, error) {
			themePath, err := apply.OpencodeThemePath(pathSetting("opencode.theme_path"))
			if err != nil {
				return nil, fmt.Errorf("failed to apply opencode theme: %w", err)
			}
			if err := apply.ApplyOpencodeTheme(theme, themePath); err != nil {
				return nil, fmt.Errorf("failed to apply opencode theme: %w", err)
			}
			if err := apply.ReloadOpencode(); err != nil {
				return nil, fmt.Errorf("failed to reload opencode: %w", err)
			}
			return []string{themePath}, nil
		}},
		{"herdr", "herdr.enable", func(tmpDir string) ([]string, error) {
			configPath, err := apply.HerdrConfigPath(pathSetting("herdr.config_path"))
			if err != nil {
				return nil, fmt.Errorf("failed to apply herdr theme: %w", err)
			}
			if err := apply.ApplyHerdrTheme(theme, configPath); err != nil {
				return nil, fmt.Errorf("failed to apply herdr theme: %w", err)
			}
			if err := apply.ReloadHerdr(); err != nil {
				return nil, fmt.Errorf("failed to reload herdr: %w", err)
			}
			return nil, nil
		}},
		{"rime", "rime.enable", func(tmpDir string) ([]string, error) {
			if err := apply.ApplyRimeTheme(theme, pathSetting("rime.config_dir")); err != nil {
				return nil, fmt.Errorf("failed to apply rime theme: %w", err)
			}
			if err := apply.ReloadSquirrel(); err != nil {
				return nil, fmt.Errorf("failed to reload squirrel: %w", err)
			}
			return nil, nil
		}},
		{"borders", "borders.enable", func(tmpDir string) ([]string, error) {
			color := strings.TrimPrefix(theme.Base16["base08"], "#")
			if err := exec.Command("borders", fmt.Sprintf("active_color=0xff%s", color)).Run(); err != nil {
				return nil, fmt.Errorf("failed to apply borders theme: %w", err)
			}
			return nil, nil
		}},
	}
}

// syncChezmoi commits the generated files to the dotfile source. It runs after every apply step
// so that one failed target cannot leave the source behind.
func syncChezmoi(files []string) chezmoiReport {
	report := chezmoiReport{Enabled: viper.GetBool("chezmoi.enable"), OK: true}

	if !report.Enabled {
		report.Skipped = true
		return report
	}

	unique := make([]string, 0, len(files))
	seen := make(map[string]bool, len(files))
	for _, file := range files {
		if file == "" || seen[file] {
			continue
		}
		seen[file] = true
		unique = append(unique, file)
	}

	if len(unique) == 0 {
		report.Skipped = true
		return report
	}

	report.Files = len(unique)
	args := append([]string{"add"}, unique...)
	if out, err := exec.Command("chezmoi", args...).CombinedOutput(); err != nil {
		report.OK = false
		report.Error = strings.TrimSpace(fmt.Sprintf("%v: %s", err, strings.TrimSpace(string(out))))
	}
	return report
}

func reportHuman(report switchReport) {
	for _, target := range report.Targets {
		if target.Error != "" {
			fmt.Fprintf(os.Stderr, "%s: %s\n", target.Name, target.Error)
		}
	}
	if !report.Chezmoi.OK {
		fmt.Fprintf(os.Stderr, "chezmoi: %s\n", report.Chezmoi.Error)
	}
	if report.Error != "" {
		fmt.Fprintln(os.Stderr, report.Error)
	}

	if report.OK {
		fmt.Printf("Theme '%s' applied.\n", report.Theme)
	}
}

func init() {
	rootCmd.AddCommand(switchCmd)
	switchCmd.Flags().StringVarP(&curTheme, "theme", "t", "", "Specify theme")
	switchCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Report what would change without touching anything")
	switchCmd.Flags().BoolVar(&force, "force", false, "Re-apply the theme even when it is already active")
}
