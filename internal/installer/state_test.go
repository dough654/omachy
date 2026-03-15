package installer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoadState(t *testing.T) {
	tmp := t.TempDir()
	statePathOverride = filepath.Join(tmp, "state.json")
	defer func() { statePathOverride = "" }()

	original := &State{
		InstalledPackages: []InstalledPackage{
			{Name: "neovim"},
			{Name: "ghostty", Cask: true},
		},
		InstalledTaps:    []string{"nikitabobko/tap"},
		DeployedConfigs:  map[string]string{"~/.config/nvim": "abc123"},
		OriginalDefaults: map[string]string{"dock-autohide": "false"},
		BackupPath:       "/tmp/backup-123",
		Services:         []string{"sketchybar"},
	}

	if err := SaveState(original); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadState()
	if err != nil {
		t.Fatal(err)
	}

	if len(loaded.InstalledPackages) != 2 {
		t.Errorf("InstalledPackages: got %d, want 2", len(loaded.InstalledPackages))
	}
	if loaded.InstalledPackages[0].Name != "neovim" {
		t.Errorf("InstalledPackages[0].Name: got %q, want %q", loaded.InstalledPackages[0].Name, "neovim")
	}
	if loaded.InstalledPackages[1].Cask != true {
		t.Error("InstalledPackages[1].Cask should be true")
	}
	if len(loaded.InstalledTaps) != 1 || loaded.InstalledTaps[0] != "nikitabobko/tap" {
		t.Errorf("InstalledTaps mismatch: got %v", loaded.InstalledTaps)
	}
	if loaded.DeployedConfigs["~/.config/nvim"] != "abc123" {
		t.Errorf("DeployedConfigs mismatch")
	}
	if loaded.OriginalDefaults["dock-autohide"] != "false" {
		t.Errorf("OriginalDefaults mismatch")
	}
	if loaded.BackupPath != "/tmp/backup-123" {
		t.Errorf("BackupPath: got %q", loaded.BackupPath)
	}
	if len(loaded.Services) != 1 || loaded.Services[0] != "sketchybar" {
		t.Errorf("Services mismatch: got %v", loaded.Services)
	}
}

func TestLoadStateMissing(t *testing.T) {
	tmp := t.TempDir()
	statePathOverride = filepath.Join(tmp, "nonexistent", "state.json")
	defer func() { statePathOverride = "" }()

	state, err := LoadState()
	if err != nil {
		t.Fatal(err)
	}

	if state.DeployedConfigs == nil {
		t.Error("DeployedConfigs should be initialized, got nil")
	}
	if state.OriginalDefaults == nil {
		t.Error("OriginalDefaults should be initialized, got nil")
	}
	if len(state.InstalledPackages) != 0 {
		t.Errorf("InstalledPackages should be empty, got %v", state.InstalledPackages)
	}
}

func TestStateJsonFormat(t *testing.T) {
	tmp := t.TempDir()
	statePathOverride = filepath.Join(tmp, "state.json")
	defer func() { statePathOverride = "" }()

	s := &State{
		InstalledPackages: []InstalledPackage{{Name: "pkg1"}},
		InstalledTaps:     []string{"tap1"},
		DeployedConfigs:   map[string]string{"a": "b"},
		OriginalDefaults:  map[string]string{"c": "d"},
		BackupPath:        "/backup",
		Services:          []string{"svc1"},
	}
	SaveState(s)

	data, err := os.ReadFile(statePathOverride)
	if err != nil {
		t.Fatal(err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("state file is not valid JSON: %v", err)
	}

	expectedKeys := []string{"installed_packages", "installed_taps", "deployed_configs", "original_defaults", "backup_path", "services"}
	for _, key := range expectedKeys {
		if _, ok := raw[key]; !ok {
			t.Errorf("missing expected key %q in state JSON", key)
		}
	}
}
