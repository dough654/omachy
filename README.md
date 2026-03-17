# Omachy

**Omarchy for the rest of us.**

Omachy brings the [Omarchy](https://omakub.org/) experience to macOS — a tiling WM, custom menu bar, terminal emulator, editor, and sane system defaults, all configured in one shot. For people who'd rather be on Linux but can't.

<video src="docs/omachy.mp4" autoplay loop muted playsinline></video>

## What Gets Installed

| Tool | Type | Description |
|------|------|-------------|
| [AeroSpace](https://github.com/nikitabobko/AeroSpace) | Cask | Tiling window manager |
| [SketchyBar](https://github.com/FelixKratz/SketchyBar) | Formula | Custom menu bar replacement |
| [JankyBorders](https://github.com/FelixKratz/JankyBorders) | Formula | Window border highlights |
| [Ghostty](https://ghostty.org/) | Cask | Terminal emulator |
| [Neovim](https://neovim.io/) | Formula | Text editor (Kickstart.nvim cloned if no config exists) |
| [Tmux](https://github.com/tmux/tmux) | Formula | Terminal multiplexer (TPM + plugins) |
| [Starship](https://starship.rs/) | Formula | Cross-shell prompt |
| [fzf](https://github.com/junegunn/fzf) | Formula | Fuzzy finder |
| [Lazygit](https://github.com/jesseduffield/lazygit) | Formula | Git TUI |
| [Atuin](https://atuin.sh/) | Formula | Shell history search & sync |
| [Hack Nerd Font](https://www.nerdfonts.com/) | Cask | Nerd Font for SketchyBar icons |
| [JetBrains Mono](https://www.jetbrains.com/lp/mono/) | Cask | Monospace font for terminal |

## What Gets Configured

**Config files** are deployed from the embedded `configs/` directory:

| Source | Destination |
|--------|------------|
| `aerospace/aerospace.toml` | `~/.config/aerospace/aerospace.toml` |
| `sketchybar/` | `~/.config/sketchybar/` |
| `borders/bordersrc` | `~/.config/borders/bordersrc` |
| `ghostty/config` | `~/Library/Application Support/com.mitchellh.ghostty/config` |
| `tmux/tmux.conf` | `~/.tmux.conf` *(NeverOverwrite)* |
| `starship.toml` | `~/.config/starship.toml` |
| `zsh/zshrc` | `~/.zshrc` *(NeverOverwrite)* |
| Kickstart.nvim (cloned) | `~/.config/nvim/` *(skipped if exists)* |

**macOS system defaults** are adjusted:

- Auto-hide Dock with zero delay and fast animation
- Disable MRU Spaces reordering (important for tiling WM)
- Disable window open/close animations
- Fastest key repeat rate with shortest initial delay
- Disable press-and-hold for key repeat
- Auto-hide the menu bar (replaced by SketchyBar)
- Hide desktop widgets
- Show all file extensions
- Scale minimize effect

## Prerequisites

- macOS 13 (Ventura) or later
- [Homebrew](https://brew.sh)
- Xcode Command Line Tools (`xcode-select --install`) — ensure they're up to date via Software Update

## Installation

```bash
brew tap dough654/omachy
brew install omachy
omachy install
```

### Building from source

```bash
go build -o omachy .
./omachy install
```

## Usage

### `omachy install`

Runs the full installation through an interactive TUI with five phases:

1. **Preflight** — checks architecture, macOS version, Homebrew, Xcode CLI tools, and Spaces settings
2. **Backup** — copies any existing config files to `~/.omachy/backups/<timestamp>/`
3. **Packages** — taps Homebrew repos and installs all packages
4. **Configs** — deploys embedded config files to their destinations
5. **System** — applies macOS defaults, starts brew services, prompts for AeroSpace accessibility permissions

**Flags:**

| Flag | Description |
|------|-------------|
| `--dry-run` | Show what would be done without making changes |
| `--force` | Overwrite existing configs without prompting |
| `--skip-backup` | Skip backing up existing configs |
| `--verbose` | Show detailed output |

### `omachy uninstall`

Reverses the installation — stops services, removes configs, uninstalls packages, and restores original macOS defaults from saved state.

**Flags:**

| Flag | Description |
|------|-------------|
| `--dry-run` | Show what would be done without making changes |
| `--keep-configs` | Keep deployed config files |
| `--keep-packages` | Keep installed Homebrew packages |

### `omachy doctor`

Runs preflight checks and reports system readiness without installing anything.

### `omachy status`

Shows current installation status: which packages are installed, whether deployed configs have been modified (drift detection via SHA-256 checksums), and backup location.

### `omachy version`

Prints the version string.

## Architecture

```
.
├── main.go                    # Entry point, wires embedded FS
├── configs_embed.go           # go:embed directive for configs/
├── configs/                   # Embedded config files (deployed at install time)
│   ├── aerospace/
│   ├── sketchybar/
│   ├── borders/
│   ├── ghostty/
│   ├── nvim/
│   └── tmux/
├── cmd/                       # Cobra CLI commands
│   ├── root.go                # Root command + version var
│   ├── install.go             # Install command + flags
│   ├── uninstall.go           # Uninstall command + flags
│   ├── doctor.go              # System readiness checker
│   ├── status.go              # Installation status + drift detection
│   └── version.go             # Version printer
└── internal/
    ├── manifest/              # Package and config declarations (pure data)
    ├── checksum/              # SHA-256 hashing for files and directories
    ├── backup/                # File/directory backup with tilde expansion
    ├── preflight/             # System readiness checks
    ├── installer/             # Install orchestrator + phase implementations
    │   ├── installer.go       # Phase runner
    │   ├── state.go           # JSON state persistence (~/.omachy/state.json)
    │   ├── preflight_phase.go # Preflight phase
    │   ├── brew.go            # Package installation phase
    │   ├── config.go          # Backup + config deployment phases
    │   ├── defaults.go        # macOS defaults + service start phase
    │   └── services.go        # (placeholder)
    ├── uninstaller/           # Uninstall orchestrator (reverse of installer)
    ├── shell/                 # exec.Command wrapper (Run, RunStreaming, Which)
    ├── brew/                  # Homebrew CLI wrapper (tap, install, services)
    └── tui/                   # Bubbletea terminal UI
        ├── app.go             # Root model + state machine
        ├── events.go          # Message types (PhaseStarted, LogLine, etc.)
        ├── splash.go          # Pre-install splash screen
        ├── phases.go          # Left panel phase list with status icons
        ├── header.go          # Top bar with title and progress percentage
        ├── output.go          # Scrollable log viewport
        ├── help.go            # Bottom key-binding hints
        └── styles.go          # Lipgloss color palette and styles
```

### Key Design Decisions

- **Embedded configs** — Config files are compiled into the binary via `go:embed`, making the tool a single self-contained binary with no external file dependencies.
- **State tracking** — `~/.omachy/state.json` tracks installed packages, deployed config checksums, original macOS defaults, backup paths, and running services. This enables clean uninstall and drift detection.
- **Idempotent operations** — Package installs check whether a package is already installed before running `brew install`. Taps are deduplicated.
- **Reversible by default** — Existing configs are backed up before overwriting. Original macOS defaults are read and saved before being changed. The uninstaller restores both.
- **No interfaces or mocks** — The codebase favors simple functions over abstraction layers. Shell calls go through `internal/shell/` as a thin wrapper but aren't abstracted behind interfaces.

### State File

The installer writes `~/.omachy/state.json` with the following structure:

```json
{
  "installed_packages": ["neovim", "tmux", "..."],
  "deployed_configs": {
    "/Users/you/.tmux.conf": "sha256hash",
    "/Users/you/.config/nvim": "sha256hash"
  },
  "original_defaults": {
    "com.apple.dock:autohide": "false",
    "com.apple.dock:tilesize": "64"
  },
  "backup_path": "/Users/you/.omachy/backups/20260314-182007",
  "services": ["sketchybar", "borders"]
}
```

## Testing

```bash
# Run all tests
go test ./...

# Verbose
go test ./... -v
```

Tests cover the pure logic and I/O packages: manifest data, checksum computation, backup file operations, state serialization, TUI components (phases, header, help, splash, app state machine), and preflight logic. Shell-heavy wrappers (brew, system commands) are not unit tested — they're thin wrappers best verified by running the tool.

## License

Open source. See [LICENSE](LICENSE) for details.
