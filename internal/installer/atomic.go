package installer

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// WriteFileAtomic writes data to path via a temp file in the same directory
// followed by a rename, so an interrupted write never leaves a truncated file.
//
// Symlinks are resolved first: if path is a symlink (e.g. ~/.zshrc pointing
// into a dotfiles repo), the link is kept and its target is updated, matching
// os.WriteFile's write-through behavior.
func WriteFileAtomic(path string, data []byte, mode fs.FileMode) error {
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		path = resolved
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	cleanup := func() {
		tmp.Close()
		os.Remove(tmpPath)
	}

	if _, err := tmp.Write(data); err != nil {
		cleanup()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return nil
}

// RemoveDeployedDir removes the files Omachy deployed into dest from the
// embedded source directory, then any directories left empty. Files the user
// added (e.g. custom SketchyBar plugins) are kept and returned as relative paths.
func RemoveDeployedDir(source, dest string) ([]string, error) {
	srcDir := filepath.Join("configs", source)
	var rels []string
	err := fs.WalkDir(EmbeddedConfigs, srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() == ".gitkeep" {
			return nil
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		rels = append(rels, rel)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("list embedded %s: %w", source, err)
	}
	return removeDeployedTree(dest, rels)
}

// removeDeployedTree removes rels (paths relative to dest), prunes directories
// left empty, and returns the files that remain.
func removeDeployedTree(dest string, rels []string) ([]string, error) {
	dirs := map[string]bool{dest: true}
	var firstErr error
	for _, rel := range rels {
		path := filepath.Join(dest, rel)
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) && firstErr == nil {
			firstErr = err
		}
		for d := filepath.Dir(path); strings.HasPrefix(d, dest); d = filepath.Dir(d) {
			dirs[d] = true
			if d == dest {
				break
			}
		}
	}

	// Deepest directories first; os.Remove fails on non-empty dirs, which is
	// exactly the "keep user content" behavior we want.
	ordered := make([]string, 0, len(dirs))
	for d := range dirs {
		ordered = append(ordered, d)
	}
	sort.Slice(ordered, func(i, j int) bool { return len(ordered[i]) > len(ordered[j]) })
	for _, d := range ordered {
		os.Remove(d)
	}

	var kept []string
	filepath.WalkDir(dest, func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			if rel, err := filepath.Rel(dest, path); err == nil {
				kept = append(kept, rel)
			}
		}
		return nil
	})
	return kept, firstErr
}

// ExistingModeOr returns the permission bits of the file at path, or def if it
// doesn't exist. Used when rewriting user-owned files like ~/.zshrc so an
// atomic rewrite doesn't loosen or tighten their permissions.
func ExistingModeOr(path string, def fs.FileMode) fs.FileMode {
	if info, err := os.Stat(path); err == nil {
		return info.Mode().Perm()
	}
	return def
}
