package preflight

import "testing"

func TestAllPassedAllGreen(t *testing.T) {
	checks := []Check{
		{Name: "A", Passed: true},
		{Name: "B", Passed: true},
		{Name: "C", Passed: true},
	}
	if !AllPassed(checks) {
		t.Error("all checks passed, AllPassed should return true")
	}
}

func TestAllPassedWithWarning(t *testing.T) {
	checks := []Check{
		{Name: "A", Passed: true},
		{Name: "B", Passed: false, Warning: true},
		{Name: "C", Passed: true},
	}
	if !AllPassed(checks) {
		t.Error("warning-only failure should not cause AllPassed to return false")
	}
}

func TestAllPassedWithFailure(t *testing.T) {
	checks := []Check{
		{Name: "A", Passed: true},
		{Name: "B", Passed: false, Warning: false},
		{Name: "C", Passed: true},
	}
	if AllPassed(checks) {
		t.Error("non-warning failure should cause AllPassed to return false")
	}
}

func TestParseAeroSpaceVersion(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{
			"aerospace CLI client version: 0.20.3-Beta 6dde91ba\nAeroSpace.app server version: 0.20.3-Beta 6dde91ba\n",
			"0.20.3-Beta",
		},
		{
			"aerospace CLI client version: 0.15.0 abc123\n",
			"0.15.0",
		},
		{"", ""},
		{"some random output\n", ""},
	}
	for _, tt := range tests {
		got := parseAeroSpaceVersion(tt.input)
		if got != tt.want {
			t.Errorf("parseAeroSpaceVersion(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestVersionAtLeast(t *testing.T) {
	tests := []struct {
		version string
		minimum string
		want    bool
	}{
		{"0.20.3-Beta", "0.15.0", true},
		{"0.15.0", "0.15.0", true},
		{"0.14.9", "0.15.0", false},
		{"0.12.0", "0.15.0", false},
		{"1.0.0", "0.15.0", true},
		{"0.15.1", "0.15.0", true},
		{"v0.20.0", "0.15.0", true},
	}
	for _, tt := range tests {
		got := versionAtLeast(tt.version, tt.minimum)
		if got != tt.want {
			t.Errorf("versionAtLeast(%q, %q) = %v, want %v", tt.version, tt.minimum, got, tt.want)
		}
	}
}

func TestMacOSName(t *testing.T) {
	tests := []struct {
		major int
		want  string
	}{
		{13, "Ventura"},
		{14, "Sonoma"},
		{15, "Sequoia"},
		{16, "Tahoe"},
		{99, "macOS"},
	}
	for _, tt := range tests {
		got := macOSName(tt.major)
		if got != tt.want {
			t.Errorf("macOSName(%d) = %q, want %q", tt.major, got, tt.want)
		}
	}
}
