package gh

import (
	"reflect"
	"testing"
)

func TestParseMergedClosed(t *testing.T) {
	data := []byte(`[
		{"headRefName":"feat/open","state":"OPEN"},
		{"headRefName":"feat/merged","state":"MERGED"},
		{"headRefName":"fix/closed","state":"CLOSED"},
		{"headRefName":"feat/also-open","state":"OPEN"}
	]`)

	got, err := parseMergedClosed(data)
	if err != nil {
		t.Fatalf("parseMergedClosed: %v", err)
	}
	want := map[string]bool{"feat/merged": true, "fix/closed": true}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseMergedClosed() = %v, want %v", got, want)
	}
}

func TestParseMergedClosed_Empty(t *testing.T) {
	got, err := parseMergedClosed([]byte(`[]`))
	if err != nil {
		t.Fatalf("parseMergedClosed: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("parseMergedClosed([]) = %v, want empty", got)
	}
}
