package core

import (
	"net"
	"testing"
)

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

func TestListenLoopback_BindsOnlyLoopback(t *testing.T) {
	listeners, err := listenLoopback(0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() {
		for _, listener := range listeners {
			_ = listener.Close()
		}
	}()

	for _, listener := range listeners {
		host, _, err := net.SplitHostPort(listener.Addr().String())
		if err != nil {
			t.Fatalf("unexpected address %q: %v", listener.Addr(), err)
		}
		ip := net.ParseIP(host)
		if ip == nil || !ip.IsLoopback() {
			t.Errorf("listening on %s, which is not loopback", listener.Addr())
		}
	}
}
