package manifest

import (
	"strings"
	"testing"
)

func TestPackages(t *testing.T) {
	pkgs := Packages()
	if len(pkgs) != 28 {
		t.Fatalf("expected 28 packages, got %d", len(pkgs))
	}
	for i, pkg := range pkgs {
		if pkg.Name == "" {
			t.Errorf("package %d has empty Name", i)
		}
	}
}

func TestTapPackagesFullyQualified(t *testing.T) {
	// Homebrew 6+ requires tap trust; only fully qualified installs
	// (user/repo/name) are trusted implicitly.
	for _, pkg := range Packages() {
		if pkg.Tap == "" {
			continue
		}
		if !strings.HasPrefix(strings.ToLower(pkg.Name), strings.ToLower(pkg.Tap)+"/") {
			t.Errorf("package %q from tap %q must use fully qualified name %q", pkg.Name, pkg.Tap, pkg.Tap+"/"+pkg.Name)
		}
	}
}

func TestTaps(t *testing.T) {
	taps := Taps()
	if len(taps) == 0 {
		t.Fatal("expected at least one tap")
	}

	// Should be unique
	seen := map[string]bool{}
	for _, tap := range taps {
		if tap == "" {
			t.Error("tap is empty string")
		}
		if seen[tap] {
			t.Errorf("duplicate tap: %s", tap)
		}
		seen[tap] = true
	}

	// Every tap should be referenced by at least one package
	pkgs := Packages()
	for _, tap := range taps {
		found := false
		for _, pkg := range pkgs {
			if pkg.Tap == tap {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("tap %q not referenced by any package", tap)
		}
	}
}

func TestServices(t *testing.T) {
	// Services are now managed by AeroSpace (after-startup-command), not brew services.
	// Services() should return empty since no packages have Service: true.
	svcs := Services()
	if len(svcs) != 0 {
		t.Errorf("expected no brew services (managed by AeroSpace), got %d", len(svcs))
	}
}

func TestConfigs(t *testing.T) {
	configs := Configs()
	if len(configs) == 0 {
		t.Fatal("expected at least one config")
	}
	for i, cfg := range configs {
		if cfg.Source == "" {
			t.Errorf("config %d has empty Source", i)
		}
		if cfg.Dest == "" {
			t.Errorf("config %d has empty Dest", i)
		}
		if cfg.Mode == 0 {
			t.Errorf("config %d (%s) has zero Mode", i, cfg.Source)
		}
	}
}
