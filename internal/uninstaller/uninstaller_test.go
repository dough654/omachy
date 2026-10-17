package uninstaller

import "testing"

func TestParseStoredDefault(t *testing.T) {
	tests := []struct {
		stored    string
		wantType  string
		wantValue string
	}{
		{"-bool:1", "-bool", "true"},
		{"-bool:0", "-bool", "false"},
		{"-bool:true", "-bool", "true"},
		{"-int:48", "-int", "48"},
		{"-float:0.15", "-float", "0.15"},
		{"-string:scale", "-string", "scale"},
		{"-string:a:b", "-string", "a:b"},
		{"legacy-value", "-string", "legacy-value"},
	}
	for _, tt := range tests {
		typ, value := parseStoredDefault(tt.stored)
		if typ != tt.wantType || value != tt.wantValue {
			t.Errorf("parseStoredDefault(%q) = (%q, %q), want (%q, %q)", tt.stored, typ, value, tt.wantType, tt.wantValue)
		}
	}
}

func TestRemoveManagedBlock(t *testing.T) {
	start := "# ── Omachy managed (do not edit between these markers) ──"
	end := "# ── End Omachy managed ──"
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"block removed", "export A=1\n" + start + "\neval x\n" + end + "\nexport B=2\n", "export A=1\nexport B=2\n"},
		{"no block", "export A=1\n", "export A=1\n"},
		{"orphaned start left alone", "export A=1\n" + start + "\n", "export A=1\n" + start + "\n"},
	}
	for _, tt := range tests {
		if got := removeManagedBlock(tt.input); got != tt.want {
			t.Errorf("%s: got %q, want %q", tt.name, got, tt.want)
		}
	}
}
