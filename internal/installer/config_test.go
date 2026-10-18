package installer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateZshrcBlock(t *testing.T) {
	tests := []struct {
		name      string
		existing  string
		wantIn    []string // lines expected inside the managed block
		wantNotIn []string // lines expected to be left out of the managed block
	}{
		{
			name:     "empty zshrc gets all integrations",
			existing: "",
			wantIn: []string{
				`function ls { if [ -t 0 ]; then eza`,
				`eval "$(zoxide init zsh)"`,
				`FZF_CTRL_T_OPTS`,
			},
		},
		{
			name:      "user's own ls alias and zoxide are kept",
			existing:  "alias ls='ls -G'\neval \"$(zoxide init zsh --cmd cd)\"\n",
			wantIn:    []string{`alias ll='ls -lh'`, `alias lt='eza --tree`},
			wantNotIn: []string{`function ls`, `eval "$(zoxide init zsh)"`},
		},
		{
			name:     "old managed block is rebuilt with new lines",
			existing: "export FOO=1\n\n" + zshrcMarkerStart + "\neval \"$(starship init zsh)\"\n" + zshrcMarkerEnd + "\n",
			wantIn:   []string{`eval "$(starship init zsh)"`, `eval "$(zoxide init zsh)"`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), ".zshrc")
			if tt.existing != "" {
				if err := os.WriteFile(path, []byte(tt.existing), 0644); err != nil {
					t.Fatalf("seed zshrc: %v", err)
				}
			}
			if err := updateZshrcBlock(path, func(string) {}); err != nil {
				t.Fatalf("updateZshrcBlock: %v", err)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read zshrc: %v", err)
			}
			content := string(data)
			if strings.Count(content, zshrcMarkerStart) != 1 || strings.Count(content, zshrcMarkerEnd) != 1 {
				t.Fatalf("want exactly one managed block, got:\n%s", content)
			}
			block := content[strings.Index(content, zshrcMarkerStart):]
			for _, s := range tt.wantIn {
				if !strings.Contains(block, s) {
					t.Errorf("managed block missing %q", s)
				}
			}
			for _, s := range tt.wantNotIn {
				if strings.Contains(block, s) {
					t.Errorf("managed block should not contain %q", s)
				}
			}
			if !strings.HasPrefix(content, strings.TrimRight(removeManagedBlock(tt.existing), "\n")) {
				t.Errorf("user content before the block was not preserved:\n%s", content)
			}
		})
	}
}
