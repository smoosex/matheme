#!/bin/sh

# !!!Important!!!
# I use chezmoi to manage my dotfiles, if you don't use it, you should substitute the paths below with your own and remove all cmd about chezmoi.
# for expamle:
# mkdir -p $HOME/.config/matheme
# cp -rf ./scripts $HOME/.config/matheme
# ......

cp -f ./config.toml $HOME/.config/matheme
cp -rf themes $HOME/.config/matheme
cp -rf wallpaper $HOME/.config/matheme

go build -o matheme
mkdir -p $HOME/.config/matheme/bin
cp -f matheme $HOME/.config/matheme/bin
sketchybar --reload 
