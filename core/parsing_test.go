package core

import (
	"math/big"
	"testing"
)

func TestStripChainPrefix(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"eth:0xabc", "0xabc"},
		{"oeth:0xabc", "0xabc"},
		{"base:0x123", "0x123"},
		{"0xdef", "0xdef"},
		{"no-colon", "no-colon"},
		{":leading", "leading"},
	}

	for _, tc := range tests {
		got := StripChainPrefix(tc.in)
		if got != tc.want {
			t.Fatalf("StripChainPrefix(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestAddCommas(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"1", "1"},
		{"12", "12"},
		{"123", "123"},
		{"1234", "1,234"},
		{"12345", "12,345"},
		{"123456", "123,456"},
		{"1234567", "1,234,567"},
	}
	for _, tc := range tests {
		if got := addCommas(tc.in); got != tc.want {
			t.Fatalf("addCommas(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestParseDecimals_AllZerosFractionShowsTwoDecimals(t *testing.T) {
	// 1000000 with 6 decimals => 1.00
	amount := big.NewInt(1000000)
	got := ParseDecimals(amount, 6)
	if got != "1.00" {
		t.Fatalf("ParseDecimals all zeros = %q, want %q", got, "1.00")
	}
}

func TestParseDecimals_TrimsTrailingZeros(t *testing.T) {
	// 1234500 with 4 decimals => 123.45
	amount := big.NewInt(1234500)
	got := ParseDecimals(amount, 4)
	if got != "123.45" {
		t.Fatalf("ParseDecimals trim zeros = %q, want %q", got, "123.45")
	}
}

func TestParseDecimals_PadsLeadingZeros(t *testing.T) {
	// 123 with 6 decimals => 0.000123 -> allZeros=false so should be 0.000123
	amount := big.NewInt(123)
	got := ParseDecimals(amount, 6)
	if got != "0.000123" {
		t.Fatalf("ParseDecimals pad = %q, want %q", got, "0.000123")
	}
}

func TestParseDecimals_NoDecimals(t *testing.T) {
	amount := big.NewInt(1234567)
	got := ParseDecimals(amount, 0)
	if got != "1,234,567" {
		t.Fatalf("ParseDecimals no decimals = %q, want %q", got, "1,234,567")
	}
}
