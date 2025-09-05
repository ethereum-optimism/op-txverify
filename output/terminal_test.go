package output

import (
	"testing"
)

func TestFormatHash(t *testing.T) {
	if got := formatHash("0xdeadbeef"); got != "0xDEADBEEF" {
		t.Fatalf("formatHash lower -> upper = %q", got)
	}
	if got := formatHash("DEADBEEF"); got != "DEADBEEF" {
		t.Fatalf("formatHash no prefix stays upper = %q", got)
	}
}
