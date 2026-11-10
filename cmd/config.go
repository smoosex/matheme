package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/matheme/pkg/configfile"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type configReport struct {
	Path   string         `json:"path"`
	Values map[string]any `json:"values"`
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show the effective configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		report, err := readConfig()
		if err != nil {
			return reportError(configReport{}, "%v", err)
		}

		if jsonOut {
			return emitJSON(report)
		}

		fmt.Println(report.Path)
		for _, key := range sortedKeys(report.Values) {
			fmt.Printf("%s = %v\n", key, report.Values[key])
		}
		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set key=value [key=value...]",
	Short: "Update configuration values in place",
	Long: `Update configuration values in place.

Dotted keys address nested tables, for example chezmoi.enable or
wallpaper.wallpapers.default. Comments, ordering and formatting of the
untouched lines are preserved.`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		changes := make(map[string]string, len(args))
		for _, arg := range args {
			idx := strings.Index(arg, "=")
			if idx <= 0 {
				return reportError(map[string]any{"ok": false, "error": "expected key=value"}, "invalid assignment %q, expected key=value", arg)
			}
			key := strings.TrimSpace(arg[:idx])
			if key == "" {
				return reportError(map[string]any{"ok": false, "error": "empty key"}, "invalid assignment %q, key is empty", arg)
			}
			changes[key] = arg[idx+1:]
		}

		path, err := configPath()
		if err != nil {
			return reportError(map[string]any{"ok": false, "error": err.Error()}, "%v", err)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return reportError(map[string]any{"ok": false, "path": path, "error": err.Error()}, "failed to read config %s: %v", path, err)
		}

		updated, err := configfile.Set(data, changes)
		if err != nil {
			return reportError(map[string]any{"ok": false, "path": path, "error": err.Error()}, "%v", err)
		}

		if err := writeFileAtomic(path, updated); err != nil {
			return reportError(map[string]any{"ok": false, "path": path, "error": err.Error()}, "%v", err)
		}

		report, err := readConfig()
		if err != nil {
			return reportError(map[string]any{"ok": false, "path": path, "error": err.Error()}, "%v", err)
		}

		if jsonOut {
			return emitJSON(report)
		}

		for _, key := range sortedKeys(changes) {
			fmt.Printf("%s = %v\n", key, report.Values[key])
		}
		return nil
	},
}

func configPath() (string, error) {
	if used := viper.ConfigFileUsed(); used != "" {
		return used, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return filepath.Join(homeDir, ".config", "matheme", "config.toml"), nil
}

// readConfig re-reads the file so that a value written by `config set` shows up immediately.
func readConfig() (configReport, error) {
	path, err := configPath()
	if err != nil {
		return configReport{}, err
	}

	if err := viper.ReadInConfig(); err != nil {
		return configReport{Path: path}, fmt.Errorf("failed to read config %s: %w", path, err)
	}

	values := make(map[string]any)
	flattenSettings("", viper.AllSettings(), values)
	return configReport{Path: path, Values: values}, nil
}

func flattenSettings(prefix string, in map[string]any, out map[string]any) {
	for key, value := range in {
		full := key
		if prefix != "" {
			full = prefix + "." + key
		}
		if nested, ok := value.(map[string]any); ok {
			flattenSettings(full, nested, out)
			continue
		}
		out[full] = value
	}
}

func sortedKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func writeFileAtomic(path string, data []byte) error {
	mode := os.FileMode(0o644)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}

	tmp := path + ".matheme.tmp"
	if err := os.WriteFile(tmp, data, mode); err != nil {
		return fmt.Errorf("failed to write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("failed to replace %s: %w", path, err)
	}
	return nil
}

func init() {
	configCmd.AddCommand(configSetCmd)
	rootCmd.AddCommand(configCmd)
}
