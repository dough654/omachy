#!/usr/bin/env bash
# Show CPU usage percentage with color-coded Nerd Font icon.

CPU=$(top -l 1 -n 0 | awk '/CPU usage/ {gsub(/%/,""); print int($3 + $5)}')

if [ "$CPU" -ge 80 ]; then
    COLOR=0xfff38ba8   # red
elif [ "$CPU" -ge 50 ]; then
    COLOR=0xfff9e2af   # yellow
else
    COLOR=0xffa6e3a1   # green
fi

sketchybar --set "$NAME" icon.color="$COLOR" label="${CPU}%"
