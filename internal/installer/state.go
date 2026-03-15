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
	DeployedConfigs   map[string]string  `json:"deployed_configs"`   // dest path → sha256
	OriginalDefaults  map[string]string  `json:"original_defaults"`  // key → original value
	BackupPath        string             `json:"backup_path"`
	Services          []string           `json:"services"`
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
		return nil, err
	}
	if s.DeployedConfigs == nil {
		s.DeployedConfigs = make(map[string]string)
	}
	if s.OriginalDefaults == nil {
		s.OriginalDefaults = make(map[string]string)
	}
	return &s, nil
}

// SaveState writes the state file.
func SaveState(s *State) error {
	path := statePath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
