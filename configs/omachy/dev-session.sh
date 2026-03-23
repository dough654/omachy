#!/bin/sh
# dev-session.sh — open a tmux dev environment for the current project
# Usage: dev [path]  (defaults to current directory)

dir="${1:-$(pwd)}"
session="$(basename "$dir" | tr '.:' '-')"

cd "$dir" || exit 1

# Reattach if session already exists
if tmux has-session -t "$session" 2>/dev/null; then
    exec tmux attach-session -t "$session"
fi

tmux new-session -s "$session" -n "nvim" -c "$dir" -d
tmux send-keys -t "$session:nvim" "nvim ." Enter

tmux new-window -t "$session" -n "opencode" -c "$dir"
tmux send-keys -t "$session:opencode" "opencode" Enter

tmux new-window -t "$session" -n "git" -c "$dir"
tmux send-keys -t "$session:git" "lazygit" Enter

tmux new-window -t "$session" -n "server" -c "$dir"
tmux new-window -t "$session" -n "scratch" -c "$dir"

tmux select-window -t "$session:nvim"
exec tmux attach-session -t "$session"
