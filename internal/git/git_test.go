package git

import "testing"

func TestParseRemoteURL(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		// HTTPS URLs
		{"https://github.com/user/repo.git", "repo"},
		{"https://github.com/user/repo", "repo"},

		// SSH URLs (scp-style)
		{"git@github.com:user/repo.git", "repo"},
		{"git@github.com:user/repo", "repo"},

		// SSH URLs (full)
		{"ssh://git@github.com/user/repo.git", "repo"},

		// GitLab/other hosts
		{"git@gitlab.com:org/project.git", "project"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := parseRemoteURL(tt.url)
			if got != tt.want {
				t.Errorf("parseRemoteURL(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}
