package installer

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const stateDir = ".omachy"
const stateFile = "state.json"

// statePathOverride can be set in tests to redirect state file location.
var statePathOverride string

// InstalledPackage records a package that Omachy installed (not pre-existing).
type InstalledPackage struct {
	Name string `json:"name"`
	Cask bool   `json:"cask,omitempty"`
}

// State tracks what was installed for uninstall/status.
type State struct {
	InstalledPackages []InstalledPackage `json:"installed_packages"`
	InstalledTaps     []string           `json:"installed_taps"`
	DeployedConfigs   map[string]string  `json:"deployed_configs"`  // dest path → sha256
	OriginalDefaults  map[string]string  `json:"original_defaults"` // key → original value
	BackupPath        string             `json:"backup_path"`
	Services          []string           `json:"services"`
	RunningProcesses  []string           `json:"running_processes"` // processes already running before install
}

func statePath() string {
	if statePathOverride != "" {
		return statePathOverride
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, stateDir, stateFile)
}

// LoadState reads the state file, returning an empty state if it doesn't exist.
func LoadState() (*State, error) {
	path := statePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &State{
				DeployedConfigs:  make(map[string]string),
				OriginalDefaults: make(map[string]string),
			}, nil
		}
		return nil, err
	}

	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		// Corrupted state file — treat as empty rather than failing
		return &State{
			DeployedConfigs:  make(map[string]string),
			OriginalDefaults: make(map[string]string),
		}, nil
	}
	if s.DeployedConfigs == nil {
		s.DeployedConfigs = make(map[string]string)
	}
	if s.OriginalDefaults == nil {
		s.OriginalDefaults = make(map[string]string)
	}
	return &s, nil
}

// SaveState writes the state file atomically (write to temp, then rename).
func SaveState(s *State) error {
	path := statePath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	// Write to temp file then rename for atomic update
	tmp, err := os.CreateTemp(dir, "state-*.json")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}

	return os.Rename(tmpPath, path)
}
