#!/usr/bin/env bash
# Show memory usage percentage with color-coded Nerd Font icon.

TOTAL=$(sysctl -n hw.memsize)
TOTAL_GB=$(( TOTAL / 1073741824 ))

# vm_stat reports pages; page size is 4096 bytes on Apple Silicon
PAGE_SIZE=4096
VM=$(vm_stat)
WIRED=$(echo "$VM"   | awk '/wired down/   {gsub(/\./,""); print $4}')
ACTIVE=$(echo "$VM"  | awk '/Pages active/  {gsub(/\./,""); print $3}')
COMPRESSED=$(echo "$VM" | awk '/occupied by compressor/ {gsub(/\./,""); print $5}')

USED_BYTES=$(( (WIRED + ACTIVE + COMPRESSED) * PAGE_SIZE ))
USED_GB=$(( USED_BYTES / 1073741824 ))
PCT=$(( USED_BYTES * 100 / TOTAL ))

if [ "$PCT" -ge 80 ]; then
    COLOR=0xfff38ba8   # red
elif [ "$PCT" -ge 60 ]; then
    COLOR=0xfff9e2af   # yellow
else
    COLOR=0xffa6e3a1   # green
fi

sketchybar --set "$NAME" icon="󰍛" icon.color="$COLOR" label="${USED_GB}/${TOTAL_GB}GB"
