# matheme

A macOS theme manager CLI that synchronizes themes across multiple applications.

## Showcase

<img width="1920" height="1080" alt="Theme showcase" src="https://github.com/user-attachments/assets/b6f10662-207f-49a4-85ef-595845ed6c1f" />
<img width="1920" height="1080" alt="Theme showcase 2" src="https://github.com/user-attachments/assets/0ecd9686-abd9-4de2-9780-5ea003a1515d" />
<img width="1920" height="1080" alt="Theme showcase 3" src="https://github.com/user-attachments/assets/327e7345-8c1f-43b5-9c09-0a8e73279509" />

## Features

- **One-command theme switching**: Apply a consistent theme across all supported applications
- **Multiple app support**: Neovim (NvChad), SketchyBar, Alacritty, Ghostty, Kitty, Tmux, Borders, pi, Rime (Squirrel), Starship, opencode
- **Wallpaper automation**: Automatically switch wallpapers across all Spaces and displays based on selected theme
- **System appearance**: Toggles macOS dark/light mode
- ** chezmoi integration**: Optional dotfile management support

## Supported Tools

- [x] Neovim (NvChad only)
- [x] SketchyBar
- [x] Alacritty
- [x] Ghostty (1.20+)
- [x] Kitty
- [x] Tmux
- [x] Borders
- [x] macOS System Appearance
- [x] Desktop Wallpaper (all Spaces)
- [x] pi
- [x] Rime (Squirrel)
- [x] Starship
- [x] opencode

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

[tmux]
enable = true
switch_script_path = "/Users/yourname/.local/share/tmux/plugins/tmux-theme/scripts/theme_menu.sh"
current_theme_path = "/Users/yourname/.config/tmux/theme/current_theme.conf"

[macos_system_appearance]
enable = true

[pi]
enable = true
control_file_path = "/Users/yourname/.pi/agent/pi-theme.json"

# opencode TUI
[opencode]
enable = true
theme_path = "/Users/yourname/.config/opencode/themes/matheme.json"

# Rime input method (Squirrel on macOS)
[rime]
enable = false
config_dir = "/Users/yourname/Library/Rime"

[borders]
enable = false
```

Add your theme files (TOML format) to `~/.config/matheme/themes/`:

Built-in themes include `everforest`, `everforest_light`, `one_light`, `rosepine`, `tokyonight`, `gruvchad`, `tundra`, and `bearded-arc`.

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

When `[pi].enable` is true, `matheme` also updates `[pi].control_file_path` if the selected theme has a pi theme mapping. This requires the [`smoosex/pi-themes`](https://github.com/smoosex/pi-themes) plugin for pi. If `control_file_path` is empty, it falls back to `~/.pi/agent/pi-theme.json`:

| matheme theme | pi theme |
| --- | --- |
| `one_light` | `onedark-light` |
| `gruvchad` | `gruvbox-dark` |
| `rosepine` | `rosepine-dark` |
| `tokyonight` | `tokyonight-dark` |
| `everforest` | `everforest-dark` |
| `everforest_light` | `everforest-light` |
| `tundra` | `tundra-dark` |
| `bearded-arc` | `bearded-arc-dark` |

### opencode

`matheme` converts the base16 palette into a version 2 opencode theme and writes it to `[opencode].theme_path`
(default `~/.config/opencode/themes/matheme.json`). The file name is the theme name opencode sees, so point
`~/.config/opencode/cli.json` at it once:

```json
{
  "theme": {
    "name": "matheme"
  }
}
```

From then on every `matheme switch` rewrites `matheme.json` and opencode follows the rest of your setup. Running
terminal clients are asked to re-read the theme files, so opencode does not need a restart. The theme uses the mode
from the theme's `type` (`dark` or `light`); opencode falls back to that mode when the other one is requested.

### Rime (Squirrel)

When `[rime].enable` is true, `matheme` generates `matheme_squirrel.yaml` (a `preset_color_schemes` block converted from the base16 palette) into `[rime].config_dir` and invokes Squirrel's `--reload` to redeploy.

This needs a one-time wiring in your `squirrel.custom.yaml`:

```yaml
patch:
  "style/color_scheme": matheme
  "style/color_scheme_dark": matheme
  "preset_color_schemes/+":
    __include: matheme_squirrel:/preset_color_schemes
```

After this, every `matheme switch` updates the look of your input method candidate window along with everything else.

If you enable Tmux support, install [`smoosex/tmux-theme`](https://github.com/smoosex/tmux-theme) first.
`matheme` calls the plugin's `theme_menu.sh switch <theme>` script directly, and the plugin is responsible for persisting the selected theme and reloading tmux when needed.

TPM example:

```tmux
source-file -q ~/.config/tmux/theme/current_theme.conf

set -g @plugin 'smoosex/tmux-theme'
set -g @theme_switch_key "T"

run '~/.tmux/plugins/tpm/tpm'
```

Then make sure your tmux config loads the plugin:

```tmux
run ~/.config/tmux/plugins/tmux-theme/theme.tmux
```

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

## Credits

- Themes from [base46](https://github.com/NvChad/base46)
- Wallpapers sourced from NvChad Discord
- Inspired by [siduck](https://github.com/siduck)

## License

MIT
