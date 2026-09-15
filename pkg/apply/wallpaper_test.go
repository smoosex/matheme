package apply

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"howett.net/plist"
)

func testDesktopEntry(t *testing.T, url string) map[string]interface{} {
	t.Helper()

	config, err := plist.Marshal(map[string]interface{}{
		"type": "imageFile",
		"url":  map[string]interface{}{"relative": url},
	}, plist.BinaryFormat)
	if err != nil {
		t.Fatalf("failed to encode test configuration: %v", err)
	}

	return map[string]interface{}{
		"Content": map[string]interface{}{
			"Choices": []interface{}{
				map[string]interface{}{
					"Provider":      "com.apple.wallpaper.choice.image",
					"Files":         []interface{}{},
					"Configuration": config,
				},
			},
			"EncodedOptionValues": []byte("placement-options"),
			"Shuffle":             "$null",
		},
		"LastSet": time.Now(),
		"LastUse": time.Now(),
	}
}

func testStore(t *testing.T) (map[string]interface{}, string) {
	t.Helper()

	spaceWallpaper := "file:///Users/test/old-space.jpg"
	spaceDesktop := testDesktopEntry(t, spaceWallpaper)

	store := map[string]interface{}{
		"AllSpacesAndDisplays": map[string]interface{}{
			"Idle": testDesktopEntry(t, "file:///System/Library/macintosh.heic"),
			"Type": "idle",
		},
		"Spaces": map[string]interface{}{
			"SPACE-1": map[string]interface{}{
				"Displays": map[string]interface{}{
					"DISPLAY-1": map[string]interface{}{"Desktop": spaceDesktop},
				},
			},
		},
		"Displays": map[string]interface{}{
			"DISPLAY-1": map[string]interface{}{"Desktop": testDesktopEntry(t, "file:///Users/test/old-display.jpg")},
		},
	}

	return store, spaceWallpaper
}

func TestSetWallpaperForAllSpaces(t *testing.T) {
	store, spaceWallpaper := testStore(t)

	if err := setWallpaperForAllSpaces(store, "/Users/test/new wall.jpg"); err != nil {
		t.Fatalf("setWallpaperForAllSpaces returned an error: %v", err)
	}

	all := store["AllSpacesAndDisplays"].(map[string]interface{})
	if all["Type"] != "individual" {
		t.Errorf("expected AllSpacesAndDisplays type to be individual, got %v", all["Type"])
	}

	entry, ok := all["Desktop"].(map[string]interface{})
	if !ok {
		t.Fatal("expected a desktop entry in AllSpacesAndDisplays")
	}

	config, err := decodeConfiguration(entry)
	if err != nil {
		t.Fatalf("failed to decode written configuration: %v", err)
	}
	if config["type"] != "imageFile" {
		t.Errorf("expected type imageFile, got %v", config["type"])
	}

	url, _ := config["url"].(map[string]interface{})
	if url["relative"] != "file:///Users/test/new%20wall.jpg" {
		t.Errorf("unexpected wallpaper url: %v", url["relative"])
	}

	spaces := store["Spaces"].(map[string]interface{})
	spaceDisplay := spaces["SPACE-1"].(map[string]interface{})["Displays"].(map[string]interface{})
	spaceConfig, err := decodeConfiguration(spaceDisplay["DISPLAY-1"].(map[string]interface{})["Desktop"].(map[string]interface{}))
	if err != nil {
		t.Fatalf("failed to decode per-space configuration: %v", err)
	}
	spaceURL, _ := spaceConfig["url"].(map[string]interface{})
	if spaceURL["relative"] != spaceWallpaper {
		t.Errorf("per-space wallpaper was modified: %v", spaceURL["relative"])
	}

	if _, err := plist.Marshal(store, plist.BinaryFormat); err != nil {
		t.Errorf("failed to encode modified store: %v", err)
	}
}

func TestSetWallpaperForAllSpacesWithoutSection(t *testing.T) {
	store, _ := testStore(t)
	delete(store, "AllSpacesAndDisplays")

	if err := setWallpaperForAllSpaces(store, "/Users/test/new.jpg"); err == nil {
		t.Fatal("expected an error when AllSpacesAndDisplays is missing")
	}
}

func TestFileURL(t *testing.T) {
	if got := fileURL("/Users/test/wall.jpg"); got != "file:///Users/test/wall.jpg" {
		t.Errorf("unexpected url: %s", got)
	}
	if got := fileURL("/Users/test/wall paper.jpg"); got != "file:///Users/test/wall%20paper.jpg" {
		t.Errorf("expected escaped url, got %s", got)
	}
}

// TestApplyWallpaperLive writes the real wallpaper store and restarts the
// wallpaper agent. Run it with:
//
//	MATHEME_LIVE_TEST=1 MATHEME_LIVE_WALLPAPER=/path/to.jpg go test ./pkg/apply -run Live -v
func TestApplyWallpaperLive(t *testing.T) {
	if os.Getenv("MATHEME_LIVE_TEST") == "" {
		t.Skip("set MATHEME_LIVE_TEST=1 and MATHEME_LIVE_WALLPAPER to run")
	}

	wallpaper := os.Getenv("MATHEME_LIVE_WALLPAPER")
	if wallpaper == "" {
		t.Fatal("MATHEME_LIVE_WALLPAPER is not set")
	}

	if err := ApplyWallpaper(wallpaper); err != nil {
		t.Fatalf("failed to apply wallpaper: %v", err)
	}

	storePath, err := wallpaperStorePath()
	if err != nil {
		t.Fatalf("failed to locate wallpaper store: %v", err)
	}
	raw, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("failed to read wallpaper store: %v", err)
	}

	var store map[string]interface{}
	if _, err := plist.Unmarshal(raw, &store); err != nil {
		t.Fatalf("failed to parse wallpaper store: %v", err)
	}

	all, ok := store["AllSpacesAndDisplays"].(map[string]interface{})
	if !ok {
		t.Fatal("wallpaper store has no AllSpacesAndDisplays section")
	}
	entry, ok := all["Desktop"].(map[string]interface{})
	if !ok {
		t.Fatal("wallpaper store has no global desktop entry")
	}

	config, err := decodeConfiguration(entry)
	if err != nil {
		t.Fatalf("failed to decode global configuration: %v", err)
	}
	url, _ := config["url"].(map[string]interface{})

	absolute, err := filepath.Abs(wallpaper)
	if err != nil {
		t.Fatalf("failed to resolve wallpaper path: %v", err)
	}
	if url["relative"] != fileURL(absolute) {
		t.Errorf("expected %s, got %v", fileURL(absolute), url["relative"])
	}
}
