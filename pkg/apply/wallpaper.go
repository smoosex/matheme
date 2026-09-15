package apply

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"howett.net/plist"
)

const (
	wallpaperStoreName = "Library/Application Support/com.apple.wallpaper/Store/Index.plist"
	wallpaperAgent     = "WallpaperAgent"
)

// ApplyWallpaper sets the wallpaper for every space. macOS stores wallpapers per
// space, and the Finder/osascript API only covers the active space, so the
// wallpaper store's "all spaces and displays" slot is written instead.
func ApplyWallpaper(wallpaperPath string) error {
	imagePath, err := expandPath(wallpaperPath)
	if err != nil {
		return err
	}

	storePath, err := wallpaperStorePath()
	if err != nil {
		return applyWallpaperToActiveSpace(imagePath, err)
	}

	raw, err := os.ReadFile(storePath)
	if err != nil {
		return applyWallpaperToActiveSpace(imagePath, err)
	}

	if err := writeWallpaperStoreForAllSpaces(storePath, raw, imagePath); err != nil {
		return applyWallpaperToActiveSpace(imagePath, err)
	}

	return reloadWallpaperAgent()
}

func applyWallpaperToActiveSpace(imagePath string, cause error) error {
	if err := applyWallpaperToActiveSpaceFinder(imagePath); err != nil {
		return fmt.Errorf("%v (fallback failed: %w)", cause, err)
	}
	return nil
}

func wallpaperStorePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return filepath.Join(homeDir, wallpaperStoreName), nil
}

func expandPath(path string) (string, error) {
	expanded := path
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user home directory: %w", err)
		}
		expanded = filepath.Join(homeDir, path[2:])
	}
	absolute, err := filepath.Abs(expanded)
	if err != nil {
		return "", fmt.Errorf("failed to resolve wallpaper path %s: %w", path, err)
	}
	return absolute, nil
}

func writeWallpaperStoreForAllSpaces(storePath string, raw []byte, imagePath string) error {
	var store map[string]interface{}
	if _, err := plist.Unmarshal(raw, &store); err != nil {
		return fmt.Errorf("failed to parse wallpaper store %s: %w", storePath, err)
	}

	if err := setWallpaperForAllSpaces(store, imagePath); err != nil {
		return err
	}

	encoded, err := plist.Marshal(store, plist.BinaryFormat)
	if err != nil {
		return fmt.Errorf("failed to encode wallpaper store: %w", err)
	}

	tmpPath := filepath.Join(filepath.Dir(storePath), ".Index.plist.matheme")
	if err := os.WriteFile(tmpPath, encoded, 0o644); err != nil {
		return fmt.Errorf("failed to write wallpaper store: %w", err)
	}
	if err := os.Rename(tmpPath, storePath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to replace wallpaper store: %w", err)
	}

	return nil
}

func setWallpaperForAllSpaces(store map[string]interface{}, imagePath string) error {
	all, ok := store["AllSpacesAndDisplays"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("wallpaper store has no AllSpacesAndDisplays section")
	}

	entry, ok := findDesktopEntry(store)
	if !ok {
		return fmt.Errorf("wallpaper store has no existing desktop entry to reuse")
	}

	clone, err := cloneEntry(entry)
	if err != nil {
		return err
	}
	if err := setDesktopEntryImage(clone, imagePath); err != nil {
		return err
	}

	all["Desktop"] = clone
	all["Type"] = "individual"

	return nil
}

// findDesktopEntry returns any per-space or per-display desktop entry, preferring
// image wallpapers so their placement options can be reused.
func findDesktopEntry(store map[string]interface{}) (map[string]interface{}, bool) {
	var fallback map[string]interface{}

	for _, entry := range desktopEntries(store) {
		if fallback == nil {
			fallback = entry
		}
		if config, err := decodeConfiguration(entry); err == nil && config["type"] == "imageFile" {
			return entry, true
		}
	}

	if fallback != nil {
		return fallback, true
	}
	return nil, false
}

func desktopEntries(store map[string]interface{}) []map[string]interface{} {
	var entries []map[string]interface{}

	appendDesktop := func(holder interface{}) {
		item, ok := holder.(map[string]interface{})
		if !ok {
			return
		}
		if entry, ok := item["Desktop"].(map[string]interface{}); ok {
			entries = append(entries, entry)
		}
	}

	spaces, _ := store["Spaces"].(map[string]interface{})
	for _, rawSpace := range spaces {
		space, ok := rawSpace.(map[string]interface{})
		if !ok {
			continue
		}
		appendDesktop(space)

		displays, _ := space["Displays"].(map[string]interface{})
		for _, rawDisplay := range displays {
			appendDesktop(rawDisplay)
		}
	}

	displays, _ := store["Displays"].(map[string]interface{})
	for _, rawDisplay := range displays {
		appendDesktop(rawDisplay)
	}

	return entries
}

func cloneEntry(entry map[string]interface{}) (map[string]interface{}, error) {
	encoded, err := plist.Marshal(entry, plist.BinaryFormat)
	if err != nil {
		return nil, fmt.Errorf("failed to encode wallpaper entry: %w", err)
	}

	var clone map[string]interface{}
	if _, err := plist.Unmarshal(encoded, &clone); err != nil {
		return nil, fmt.Errorf("failed to decode wallpaper entry: %w", err)
	}

	return clone, nil
}

func decodeConfiguration(entry map[string]interface{}) (map[string]interface{}, error) {
	content, ok := entry["Content"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("wallpaper entry has no content")
	}

	choices, ok := content["Choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return nil, fmt.Errorf("wallpaper entry has no choices")
	}

	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("wallpaper entry choice is malformed")
	}

	blob, ok := choice["Configuration"].([]byte)
	if !ok || len(blob) == 0 {
		return nil, fmt.Errorf("wallpaper entry has no configuration")
	}

	var config map[string]interface{}
	if _, err := plist.Unmarshal(blob, &config); err != nil {
		return nil, fmt.Errorf("failed to parse wallpaper configuration: %w", err)
	}

	return config, nil
}

func setDesktopEntryImage(entry map[string]interface{}, imagePath string) error {
	content, ok := entry["Content"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("wallpaper entry has no content")
	}

	choices, ok := content["Choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return fmt.Errorf("wallpaper entry has no choices")
	}

	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		return fmt.Errorf("wallpaper entry choice is malformed")
	}

	config, err := decodeConfiguration(entry)
	if err != nil {
		config = map[string]interface{}{}
	}
	config["type"] = "imageFile"
	config["url"] = map[string]interface{}{"relative": fileURL(imagePath)}

	blob, err := plist.Marshal(config, plist.BinaryFormat)
	if err != nil {
		return fmt.Errorf("failed to encode wallpaper configuration: %w", err)
	}

	choice["Configuration"] = blob
	choice["Provider"] = "com.apple.wallpaper.choice.image"

	return nil
}

func fileURL(path string) string {
	return (&url.URL{Scheme: "file", Path: path}).String()
}

// reloadWallpaperAgent restarts the wallpaper agent so it re-reads the store.
// SIGKILL is used because a graceful shutdown flushes the agent's stale state
// back over the file. launchd relaunches it immediately.
func reloadWallpaperAgent() error {
	exec.Command("killall", "-9", wallpaperAgent).Run()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if wallpaperAgentIsRunning() {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}

	exec.Command("open", "-a", "/System/Library/CoreServices/WallpaperAgent.app").Run()
	time.Sleep(time.Second)
	if wallpaperAgentIsRunning() {
		return nil
	}

	return fmt.Errorf("failed to restart %s to pick up the new wallpaper", wallpaperAgent)
}

func wallpaperAgentIsRunning() bool {
	return exec.Command("pgrep", "-x", wallpaperAgent).Run() == nil
}
