package cmd

import "testing"

func TestResolveVersion(t *testing.T) {
	tests := []struct {
		name    string
		ldflags string
		build   string
		want    string
	}{
		{"ldflags wins (release build)", "1.2.3", "v1.2.3", "1.2.3"},
		{"ldflags wins even over empty build", "1.2.3", "", "1.2.3"},
		{"go install: module version, v trimmed", "dev", "v0.2.0", "0.2.0"},
		{"go install: pseudo-version", "dev", "v0.0.0-20260101000000-abcdef123456", "0.0.0-20260101000000-abcdef123456"},
		{"local build: devel falls back to dev", "dev", "(devel)", "dev"},
		{"no build info falls back to dev", "dev", "", "dev"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveVersion(tt.ldflags, tt.build); got != tt.want {
				t.Errorf("resolveVersion(%q, %q) = %q, want %q", tt.ldflags, tt.build, got, tt.want)
			}
		})
	}
}
