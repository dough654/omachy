#!/usr/bin/env bash
# Show system volume with a Nerd Font speaker icon.
VOLUME=$(osascript -e 'output volume of (get volume settings)')
MUTED=$(osascript -e 'output muted of (get volume settings)')

if [ "$MUTED" = "true" ] || [ "$VOLUME" = "0" ]; then
    ICON="󰝟"
    COLOR=0xff6c7086
elif [ "$VOLUME" -lt 34 ]; then
    ICON="󰕿"
    COLOR=0xffcdd6f4
elif [ "$VOLUME" -lt 67 ]; then
    ICON="󰖀"
    COLOR=0xffcdd6f4
else
    ICON="󰕾"
    COLOR=0xffcdd6f4
fi

sketchybar --set "$NAME" icon="$ICON" icon.color="$COLOR" label="${VOLUME}%"
