#!/usr/bin/env bash
# $1 = workspace ID this item represents
# $2 = this workspace's accent color
# $FOCUSED_WORKSPACE = set by the aerospace_workspace_change trigger

COLOR="${2:-0xffcdd6f4}"

if [ "$1" = "$FOCUSED_WORKSPACE" ]; then
    sketchybar --set "$NAME" \
        background.drawing=on    \
        background.color=$COLOR  \
        label.color=0xff1e1e2e
else
    sketchybar --set "$NAME" \
        background.drawing=off   \
        label.color=$COLOR
fi
