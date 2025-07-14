#!/bin/sh

# I use chezmoi to manage my dotfiles, if you don't use it, you should substitute the dst paths below with your own and remove all cmd about chezmoi.
# for expamle:
# mkdir -p $HOME/.config/matheme
# cp -rf ./scripts $HOME/.config/matheme
# ......

cp -rf ./scripts $HOME/.local/share/chezmoi/dot_config/matheme
cp -f ./config.toml $HOME/.local/share/chezmoi/dot_config/matheme
cp -rf themes $HOME/.local/share/chezmoi/dot_config/matheme
cp -rf wallpaper $HOME/.local/share/chezmoi/dot_config/matheme

chezmoi apply --force

go build -o matheme
mkdir -p $HOME/.config/sketchybar/bin
cp -f matheme $HOME/.config/sketchybar/bin
chmod +x $HOME/.config/sketchybar/bin/matheme
chezmoi add $HOME/.config/sketchybar/bin
