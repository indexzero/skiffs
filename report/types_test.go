package report

import (
	"encoding/json"
	"testing"
)

func TestFileStatus_Count(t *testing.T) {
	tests := []struct {
		name  string
		files FileStatus
		want  int
	}{
		{"empty", FileStatus{}, 0},
		{"modified only", FileStatus{Modified: []string{"a.go"}}, 1},
		{"all types", FileStatus{
			Modified:  []string{"a.go", "b.go"},
			Untracked: []string{"c.go"},
			Staged:    []string{"d.go"},
		}, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.files.Count(); got != tt.want {
				t.Errorf("Count() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestRepoStatus_IsDirty(t *testing.T) {
	tests := []struct {
		name string
		repo RepoStatus
		want bool
	}{
		{"clean", RepoStatus{}, false},
		{"has uncommitted", RepoStatus{Files: FileStatus{Modified: []string{"a"}}}, true},
		{"ahead", RepoStatus{Ahead: 1}, true},
		{"behind", RepoStatus{Behind: 1}, true},
		{"no upstream but clean", RepoStatus{NoUpstream: true}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.repo.IsDirty(); got != tt.want {
				t.Errorf("IsDirty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRepoStatus_ErrorField(t *testing.T) {
	tests := []struct {
		name      string
		repo      RepoStatus
		wantInJSON bool
	}{
		{"no error omits field", RepoStatus{Path: "/test"}, false},
		{"error included", RepoStatus{Path: "/test", Error: "git failed"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.repo)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}
			hasError := json.Valid(data) && contains(string(data), `"error"`)
			if hasError != tt.wantInJSON {
				t.Errorf("JSON error field presence = %v, want %v; json: %s", hasError, tt.wantInJSON, data)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
