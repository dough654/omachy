#!/usr/bin/env bash
# Show date and time. %-d omits the leading zero on the day.
sketchybar --set "$NAME" label="$(date '+%a %b %-d %-I:%M %p')"
