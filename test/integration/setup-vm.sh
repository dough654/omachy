#!/bin/sh
# setup-vm.sh — Create a base macOS VM image for Omachy integration tests.
#
# This pulls a macOS Tahoe base image from the Tart registry, enables
# Remote Login (SSH), and installs Homebrew. The result is saved as
# "omachy-base" and can be cloned cheaply for each test run.
#
# Prerequisites: brew install cirruslabs/cli/tart cirruslabs/cli/sshpass
#
# Usage: ./test/integration/setup-vm.sh

set -eu

VM_BASE="omachy-base"
IMAGE="ghcr.io/cirruslabs/macos-tahoe-base:latest"
USER="admin"
PASS="admin"

# See run-tests.sh: retries the connect only, plus keepalives.
SSH_OPTS="-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=10 -o ConnectionAttempts=5 -o ServerAliveInterval=15 -o ServerAliveCountMax=4"

ssh_cmd() {
    # shellcheck disable=SC2086
    sshpass -p "$PASS" ssh $SSH_OPTS -q "$USER@$VM_IP" "$@"
}

echo "==> Pulling macOS Tahoe base image..."
if tart list | grep -q "$VM_BASE"; then
    echo "    $VM_BASE already exists, delete it first to rebuild"
    echo "    Run: tart delete $VM_BASE"
    exit 1
fi

tart clone "$IMAGE" "$VM_BASE"

echo "==> Starting VM to configure it..."
tart run "$VM_BASE" --no-graphics &
VM_PID=$!
trap 'kill $VM_PID 2>/dev/null; wait $VM_PID 2>/dev/null' EXIT

echo "==> Waiting for VM to boot..."
for i in $(seq 1 60); do
    VM_IP=$(tart ip "$VM_BASE" 2>/dev/null || true)
    if [ -n "$VM_IP" ]; then
        # Wait for SSH to be ready
        if sshpass -p "$PASS" ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=2 -q "$USER@$VM_IP" true 2>/dev/null; then
            break
        fi
    fi
    sleep 5
done

if [ -z "${VM_IP:-}" ]; then
    echo "ERROR: VM did not boot within timeout"
    exit 1
fi
echo "    VM is up at $VM_IP"

echo "==> Enabling Remote Login (SSH)..."
ssh_cmd "sudo systemsetup -setremotelogin on" 2>/dev/null || true

echo "==> Installing Homebrew..."
ssh_cmd 'NONINTERACTIVE=1 /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"'
ssh_cmd 'echo "eval \"\$(/opt/homebrew/bin/brew shellenv)\"" >> ~/.zprofile'
ssh_cmd 'eval "$(/opt/homebrew/bin/brew shellenv)" && brew --version'

echo "==> Shutting down VM..."
ssh_cmd "sudo shutdown -h now" 2>/dev/null || true
sleep 5
kill $VM_PID 2>/dev/null || true
wait $VM_PID 2>/dev/null || true
trap - EXIT

echo "==> Base VM '$VM_BASE' is ready."
echo "    Clone it for test runs: tart clone $VM_BASE omachy-test-run"
