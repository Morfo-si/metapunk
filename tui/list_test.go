package tui

import "testing"

func TestTruncate(t *testing.T) {
	cases := []struct {
		input string
		max   int
		want  string
	}{
		{"short", 10, "short"},
		{"exactly10c", 10, "exactly10c"},
		{"this is a very long string", 10, "this is a…"},
		{"", 5, ""},
		{"  spaces  ", 10, "spaces"}, // TrimSpace applied first
		{"abcde", 5, "abcde"},        // exactly at limit
		{"abcdef", 5, "abcd…"},       // one over
		{"日本語テスト", 4, "日本語…"},        // multibyte runes
	}
	for _, tc := range cases {
		got := truncate(tc.input, tc.max)
		if got != tc.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tc.input, tc.max, got, tc.want)
		}
	}
}
