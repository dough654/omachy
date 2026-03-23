#!/bin/sh
# run-tests.sh — Run Omachy integration tests in a disposable macOS VM.
#
# Clones the base VM, builds omachy, copies it in, runs install + uninstall,
# and verifies the system is clean afterwards.
#
# Prerequisites:
#   1. Run setup-vm.sh first to create the omachy-base image
#   2. brew install cirruslabs/cli/tart cirruslabs/cli/sshpass
#
# Usage: ./test/integration/run-tests.sh

set -eu

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

VM_BASE="omachy-base"
VM_NAME="omachy-test-$$"
USER="admin"
PASS="admin"
PASS_COUNT=0
FAIL_COUNT=0

# ── Helpers ──────────────────────────────────────────────────────────────────

ssh_cmd() {
    sshpass -p "$PASS" ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -q "$USER@$VM_IP" "$@"
}

scp_to() {
    sshpass -p "$PASS" scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -q "$1" "$USER@$VM_IP:$2"
}

assert_eq() {
    label="$1"; expected="$2"; actual="$3"
    if [ "$expected" = "$actual" ]; then
        echo "  PASS: $label"
        PASS_COUNT=$((PASS_COUNT + 1))
    else
        echo "  FAIL: $label (expected '$expected', got '$actual')"
        FAIL_COUNT=$((FAIL_COUNT + 1))
    fi
}

assert_contains() {
    label="$1"; needle="$2"; haystack="$3"
    if echo "$haystack" | grep -q "$needle"; then
        echo "  PASS: $label"
        PASS_COUNT=$((PASS_COUNT + 1))
    else
        echo "  FAIL: $label (expected to contain '$needle')"
        FAIL_COUNT=$((FAIL_COUNT + 1))
    fi
}

assert_not_contains() {
    label="$1"; needle="$2"; haystack="$3"
    if echo "$haystack" | grep -q "$needle"; then
        echo "  FAIL: $label (should not contain '$needle')"
        FAIL_COUNT=$((FAIL_COUNT + 1))
    else
        echo "  PASS: $label"
        PASS_COUNT=$((PASS_COUNT + 1))
    fi
}

cleanup() {
    echo ""
    echo "==> Cleaning up VM '$VM_NAME'..."
    kill "$VM_PID" 2>/dev/null || true
    wait "$VM_PID" 2>/dev/null || true
    tart delete "$VM_NAME" 2>/dev/null || true
}

# ── Setup ────────────────────────────────────────────────────────────────────

echo "==> Building omachy binary..."
cd "$PROJECT_ROOT"
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags "-s -w -X main.version=test" -o "$PROJECT_ROOT/test/integration/omachy-test-bin" .

echo "==> Cloning VM from $VM_BASE..."
if ! tart list | grep -q "$VM_BASE"; then
    echo "ERROR: Base VM '$VM_BASE' not found. Run setup-vm.sh first."
    exit 1
fi
tart clone "$VM_BASE" "$VM_NAME"

echo "==> Starting VM (headless)..."
tart run "$VM_NAME" --no-graphics &
VM_PID=$!
trap cleanup EXIT

echo "==> Waiting for VM to boot..."
VM_IP=""
for i in $(seq 1 60); do
    VM_IP=$(tart ip "$VM_NAME" 2>/dev/null || true)
    if [ -n "$VM_IP" ]; then
        if ssh_cmd true 2>/dev/null; then
            break
        fi
    fi
    sleep 5
done

if [ -z "$VM_IP" ]; then
    echo "ERROR: VM did not boot within timeout"
    exit 1
fi
echo "    VM is up at $VM_IP"

echo "==> Copying omachy binary to VM..."
scp_to "$PROJECT_ROOT/test/integration/omachy-test-bin" "/tmp/omachy"
ssh_cmd "chmod +x /tmp/omachy"

# ── Pre-install checks ──────────────────────────────────────────────────────

echo ""
echo "── Pre-install state ──"

pre_aerospace=$(ssh_cmd "pgrep -x AeroSpace >/dev/null 2>&1 && echo running || echo stopped")
assert_eq "AeroSpace not running before install" "stopped" "$pre_aerospace"

pre_sketchybar=$(ssh_cmd "pgrep -x sketchybar >/dev/null 2>&1 && echo running || echo stopped")
assert_eq "sketchybar not running before install" "stopped" "$pre_sketchybar"

pre_borders=$(ssh_cmd "pgrep -x borders >/dev/null 2>&1 && echo running || echo stopped")
assert_eq "borders not running before install" "stopped" "$pre_borders"

# ── Install ──────────────────────────────────────────────────────────────────

echo ""
echo "==> Running omachy install..."
ssh_cmd "eval \"\$(/opt/homebrew/bin/brew shellenv)\" && /tmp/omachy install --force --skip-backup --quiet" 2>&1 | tee /tmp/omachy-install.log || true
# Give AeroSpace a moment to start (accessibility perms may block it in VM)
sleep 5

# ── Post-install checks ─────────────────────────────────────────────────────

echo ""
echo "── Post-install state ──"

# Check state file exists
state_exists=$(ssh_cmd "test -f ~/.omachy/state.json && echo yes || echo no")
assert_eq "state.json exists after install" "yes" "$state_exists"

# Check configs deployed
aero_cfg=$(ssh_cmd "test -f ~/.config/aerospace/aerospace.toml && echo yes || echo no")
assert_eq "aerospace.toml deployed" "yes" "$aero_cfg"

sketchybar_cfg=$(ssh_cmd "test -d ~/.config/sketchybar && echo yes || echo no")
assert_eq "sketchybar config deployed" "yes" "$sketchybar_cfg"

borders_cfg=$(ssh_cmd "test -f ~/.config/borders/bordersrc && echo yes || echo no")
assert_eq "bordersrc deployed" "yes" "$borders_cfg"

starship_cfg=$(ssh_cmd "test -f ~/.config/starship.toml && echo yes || echo no")
assert_eq "starship.toml deployed" "yes" "$starship_cfg"

dev_session=$(ssh_cmd "test -f ~/.config/omachy/dev-session.sh && echo yes || echo no")
assert_eq "dev-session.sh deployed" "yes" "$dev_session"

# Check zshrc has managed block
zshrc=$(ssh_cmd "cat ~/.zshrc 2>/dev/null || echo ''")
assert_contains "zshrc has Omachy managed block" "Omachy managed" "$zshrc"
assert_contains "zshrc has starship init" "starship init zsh" "$zshrc"
assert_contains "zshrc has vim motions" "set -o vi" "$zshrc"
assert_contains "zshrc has syntax highlighting" "zsh-syntax-highlighting" "$zshrc"
assert_contains "zshrc has fastfetch" "fastfetch" "$zshrc"
assert_contains "zshrc has dev function" "dev()" "$zshrc"

# Check macOS defaults were applied
dock_autohide=$(ssh_cmd "defaults read com.apple.dock autohide 2>/dev/null || echo unset")
assert_eq "Dock autohide enabled" "1" "$dock_autohide"

menu_hidden=$(ssh_cmd "defaults read NSGlobalDomain _HIHideMenuBar 2>/dev/null || echo unset")
assert_eq "Menu bar auto-hidden" "1" "$menu_hidden"

# ── Uninstall ────────────────────────────────────────────────────────────────

echo ""
echo "==> Running omachy uninstall..."
ssh_cmd "eval \"\$(/opt/homebrew/bin/brew shellenv)\" && /tmp/omachy uninstall --quiet" 2>&1 | tee /tmp/omachy-uninstall.log || true
sleep 3

# ── Post-uninstall checks ───────────────────────────────────────────────────

echo ""
echo "── Post-uninstall state ──"

# Processes should be killed
post_aerospace=$(ssh_cmd "pgrep -x AeroSpace >/dev/null 2>&1 && echo running || echo stopped")
assert_eq "AeroSpace stopped after uninstall" "stopped" "$post_aerospace"

post_sketchybar=$(ssh_cmd "pgrep -x sketchybar >/dev/null 2>&1 && echo running || echo stopped")
assert_eq "sketchybar stopped after uninstall" "stopped" "$post_sketchybar"

post_borders=$(ssh_cmd "pgrep -x borders >/dev/null 2>&1 && echo running || echo stopped")
assert_eq "borders stopped after uninstall" "stopped" "$post_borders"

# Configs should be removed
aero_cfg_gone=$(ssh_cmd "test -f ~/.config/aerospace/aerospace.toml && echo exists || echo gone")
assert_eq "aerospace.toml removed" "gone" "$aero_cfg_gone"

borders_cfg_gone=$(ssh_cmd "test -f ~/.config/borders/bordersrc && echo exists || echo gone")
assert_eq "bordersrc removed" "gone" "$borders_cfg_gone"

starship_cfg_gone=$(ssh_cmd "test -f ~/.config/starship.toml && echo exists || echo gone")
assert_eq "starship.toml removed" "gone" "$starship_cfg_gone"

dev_session_gone=$(ssh_cmd "test -f ~/.config/omachy/dev-session.sh && echo exists || echo gone")
assert_eq "dev-session.sh removed" "gone" "$dev_session_gone"

# State file should be cleaned up
state_gone=$(ssh_cmd "test -f ~/.omachy/state.json && echo exists || echo gone")
assert_eq "state.json cleaned up" "gone" "$state_gone"

# zshrc should not have managed block
zshrc_after=$(ssh_cmd "cat ~/.zshrc 2>/dev/null || echo ''")
assert_not_contains "zshrc managed block removed" "Omachy managed" "$zshrc_after"

# Dock autohide should be restored (default is no key / 0)
dock_autohide_after=$(ssh_cmd "defaults read com.apple.dock autohide 2>/dev/null || echo unset")
assert_not_contains "Dock autohide restored" "1" "$dock_autohide_after"

# ── Pre-existing cask app (Issue #8) ─────────────────────────────────────────
# Verify that install succeeds via --adopt when a cask app (Ghostty) already
# exists in /Applications but was not installed via Homebrew.

echo ""
echo "── Pre-existing cask app (Issue #8) ──"

# Install Ghostty via brew first, then remove the Caskroom entry so Homebrew
# "forgets" about it while leaving the .app in /Applications. This simulates
# a user who installed Ghostty outside Homebrew (e.g. via DMG).
ssh_cmd "eval \"\$(/opt/homebrew/bin/brew shellenv)\" && brew install --cask ghostty" >/dev/null 2>&1
ssh_cmd "sudo rm -rf /opt/homebrew/Caskroom/ghostty"

# Verify: .app exists but Homebrew doesn't know about it
ghostty_app=$(ssh_cmd "test -d /Applications/Ghostty.app && echo exists || echo missing")
assert_eq "Ghostty.app exists in /Applications" "exists" "$ghostty_app"

ghostty_brew=$(ssh_cmd "eval \"\$(/opt/homebrew/bin/brew shellenv)\" && brew list --cask ghostty >/dev/null 2>&1 && echo known || echo unknown")
assert_eq "Ghostty not known to Homebrew" "unknown" "$ghostty_brew"

echo "==> Running omachy install with pre-existing Ghostty.app..."
ssh_cmd "eval \"\$(/opt/homebrew/bin/brew shellenv)\" && /tmp/omachy install --force --skip-backup --quiet" 2>&1 | tee /tmp/omachy-issue8.log || true

# Install must not abort — state.json should exist
state8_exists=$(ssh_cmd "test -f ~/.omachy/state.json && echo yes || echo no")
assert_eq "install succeeds despite pre-existing Ghostty.app" "yes" "$state8_exists"

# Homebrew should now know about Ghostty (adopted)
ghostty_adopted=$(ssh_cmd "eval \"\$(/opt/homebrew/bin/brew shellenv)\" && brew list --cask ghostty >/dev/null 2>&1 && echo known || echo unknown")
assert_eq "Ghostty adopted by Homebrew after install" "known" "$ghostty_adopted"

# Clean up
ssh_cmd "eval \"\$(/opt/homebrew/bin/brew shellenv)\" && /tmp/omachy uninstall --quiet" 2>&1 >/dev/null || true

# ── Summary ──────────────────────────────────────────────────────────────────

echo ""
echo "════════════════════════════════════════"
echo "  Results: $PASS_COUNT passed, $FAIL_COUNT failed"
echo "════════════════════════════════════════"

# Clean up test binary
rm -f "$PROJECT_ROOT/test/integration/omachy-test-bin"

if [ "$FAIL_COUNT" -gt 0 ]; then
    exit 1
fi
