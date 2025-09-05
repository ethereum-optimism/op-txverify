package core

import "testing"

func TestParseInt(t *testing.T) {
	tests := []struct {
		in   string
		want int
		ok   bool
	}{
		{"0", 0, true},
		{"42", 42, true},
		{"007", 7, true},
		{"-1", -1, true}, // parseInt uses fmt.Sscanf and accepts negatives
		{"abc", 0, false},
		{"", 0, false},
	}
	for _, tc := range tests {
		got, err := parseInt(tc.in)
		if tc.ok != (err == nil) {
			t.Fatalf("parseInt(%q) error = %v, ok=%v", tc.in, err, tc.ok)
		}
		if err == nil && got != tc.want {
			t.Fatalf("parseInt(%q) = %d, want %d", tc.in, got, tc.want)
		}
	}
}
