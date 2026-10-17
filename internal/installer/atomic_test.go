package installer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFileAtomicCreatesAndOverwrites(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")

	if err := WriteFileAtomic(path, []byte("one"), 0644); err != nil {
		t.Fatalf("first write: %v", err)
	}
	if err := WriteFileAtomic(path, []byte("two"), 0755); err != nil {
		t.Fatalf("second write: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(data) != "two" {
		t.Errorf("content = %q, want %q", data, "two")
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0755 {
		t.Errorf("mode = %o, want 755", info.Mode().Perm())
	}

	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("expected only the target file in dir, found %d entries (leftover temp file?)", len(entries))
	}
}

func TestWriteFileAtomicFollowsSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "dotfiles-zshrc")
	link := filepath.Join(dir, ".zshrc")

	if err := os.WriteFile(target, []byte("original"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	if err := WriteFileAtomic(link, []byte("updated"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	info, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("symlink was replaced by a regular file")
	}
	data, _ := os.ReadFile(target)
	if string(data) != "updated" {
		t.Errorf("target content = %q, want %q", data, "updated")
	}
}

func TestRemoveDeployedTreeKeepsUserFiles(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "sketchybar")
	deployed := []string{"sketchybarrc", "plugins/battery.sh", "plugins/clock.sh"}
	for _, rel := range append(deployed, "plugins/my-custom.sh") {
		path := filepath.Join(dest, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	kept, err := removeDeployedTree(dest, deployed)
	if err != nil {
		t.Fatalf("removeDeployedTree: %v", err)
	}
	if len(kept) != 1 || kept[0] != filepath.Join("plugins", "my-custom.sh") {
		t.Errorf("kept = %v, want [plugins/my-custom.sh]", kept)
	}
	for _, rel := range deployed {
		if _, err := os.Stat(filepath.Join(dest, rel)); !os.IsNotExist(err) {
			t.Errorf("%s should have been removed", rel)
		}
	}
}

func TestRemoveDeployedTreeRemovesEmptyDirs(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "sketchybar")
	deployed := []string{"sketchybarrc", "plugins/battery.sh"}
	for _, rel := range deployed {
		path := filepath.Join(dest, rel)
		os.MkdirAll(filepath.Dir(path), 0755)
		if err := os.WriteFile(path, []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// A file that was already removed by the user must not cause an error.
	kept, err := removeDeployedTree(dest, append(deployed, "plugins/gone.sh"))
	if err != nil {
		t.Fatalf("removeDeployedTree: %v", err)
	}
	if len(kept) != 0 {
		t.Errorf("kept = %v, want none", kept)
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Error("empty destination directory should have been removed")
	}
}

func TestExistingModeOr(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".zshrc")

	if got := ExistingModeOr(path, 0644); got != 0644 {
		t.Errorf("missing file: got %o, want 644", got)
	}
	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if got := ExistingModeOr(path, 0644); got != 0600 {
		t.Errorf("existing file: got %o, want 600", got)
	}
}
