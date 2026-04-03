# matheme

A macOS theme manager CLI that synchronizes themes across multiple applications.

## Features

- **One-command theme switching**: Apply a consistent theme across all supported applications
- **Multiple app support**: Neovim (NvChad), SketchyBar, Alacritty, Ghostty, Kitty, Borders
- **Wallpaper automation**: Automatically switch wallpapers based on selected theme
- **System appearance**: Toggles macOS dark/light mode
- ** chezmoi integration**: Optional dotfile management support

## Supported Tools

- [x] Neovim (NvChad only)
- [x] SketchyBar
- [x] Alacritty
- [x] Ghostty (1.20+)
- [x] Kitty
- [x] Borders
- [x] macOS System Appearance
- [x] Desktop Wallpaper

## Installation

```bash
git clone https://github.com/matheme/matheme.git
cd matheme
go build
```

Or install directly:

```bash
go install github.com/matheme/matheme@latest
```

## Configuration

Create `~/.config/matheme/config.toml`:

```toml
[chezmoi]
enable = false  # Enable chezmoi integration for dotfile management

[wallpaper]
auto = true     # Automatically change wallpaper with theme

[wallpaper.wallpapers]
default = "stairs.jpg"
everforest_light = "beach.jpg"
rosepine = "wallhaven.png"
everforest = "stairs.jpg"
# ... add more theme -> wallpaper mappings

[alacritty]
enable = true
config_path = "/Users/yourname/.config/alacritty/alacritty.toml"
theme_path = "/Users/yourname/.config/alacritty/theme.toml"

[sketchybar]
enable = true
theme_path = "/Users/yourname/.config/sketchybar/helpers/theme/palette.lua"

[neovim]
enable = true
chadrc_path = "/Users/yourname/.config/nvim/lua/chadrc.lua"

[ghostty]
enable = true
theme_path = "/Users/yourname/.config/ghostty/themes/matheme"

[kitty]
enable = true
theme_path = "/Users/yourname/.config/kitty/themes/matheme.conf"

[macos_system_appearance]
enable = true

[borders]
enable = false
```

Add your theme files (TOML format) to `~/.config/matheme/themes/`:

Built-in themes include `everforest`, `everforest_light`, `one_light`, `rosepine`, `gruvchad`, `tundra`, and `bearded-arc`.

```toml
type = "dark"

[base_16]
base00 = "#191724"  # background
base01 = "#1f1d2e"  # lighter bg
base02 = "#26233a"
base03 = "#6e6a86"  # comments
base04 = "#908caa"
base05 = "#e0def4"  # foreground
base06 = "#e0def4"
base07 = "#524f67"
base08 = "#eb6f92"  # red
base09 = "#f6c177"  # orange
base0A = "#ebbcba"  # yellow
base0B = "#31748f"  # green
base0C = "#9ccfd8"  # cyan
base0D = "#c4a7e7"  # blue
base0E = "#f6c177"  # magenta
base0F = "#524f67"
```

Place wallpapers in `~/.config/matheme/wallpaper/`.

## Usage

List available themes:

```bash
matheme list-themes
# or
matheme ls
```

Switch to a theme:

```bash
matheme switch -t rosepine
# or
matheme sw -t rosepine

# switch to Ever Forest Light
matheme sw -t everforest_light
```

## Showcase

<img width="1920" height="1080" alt="Theme showcase" src="https://github.com/user-attachments/assets/b6f10662-207f-49a4-85ef-595845ed6c1f" />
<img width="1920" height="1080" alt="Theme showcase 2" src="https://github.com/user-attachments/assets/0ecd9686-abd9-4de2-9780-5ea003a1515d" />
<img width="1920" height="1080" alt="Theme showcase 3" src="https://github.com/user-attachments/assets/327e7345-8c1f-43b5-9c09-0a8e73279509" />

## Credits

- Themes from [base46](https://github.com/NvChad/base46)
- Wallpapers sourced from NvChad Discord
- Inspired by [siduck](https://github.com/siduck)

## License

MIT
