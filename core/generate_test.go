package core

import "testing"

func TestGetNetworkInfo(t *testing.T) {
	url, chain, err := getNetworkInfo("ethereum")
	if err != nil {
		t.Fatalf("ethereum: unexpected error: %v", err)
	}
	if url == "" || chain != MainnetChainID {
		t.Fatalf("ethereum: got (%q, %d), want (non-empty, %d)", url, chain, MainnetChainID)
	}

	url, chain, err = getNetworkInfo("op")
	if err != nil {
		t.Fatalf("op: unexpected error: %v", err)
	}
	if url == "" || chain != OPMainnetChainID {
		t.Fatalf("op: got (%q, %d), want (non-empty, %d)", url, chain, OPMainnetChainID)
	}

	if _, _, err := getNetworkInfo("base"); err == nil {
		t.Fatalf("base: expected error, got nil")
	}
	if _, _, err := getNetworkInfo("unknown"); err == nil {
		t.Fatalf("unknown: expected error, got nil")
	}
}
