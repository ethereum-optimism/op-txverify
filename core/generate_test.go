package core

import (
	"strings"
	"testing"
)

func TestSelectQueuedTransaction(t *testing.T) {
	const safe = "0x847B5c174615B1B7fDF770882256e2D3E95b9D92"

	t.Run("no results is an error", func(t *testing.T) {
		if _, err := selectQueuedTransaction(APIResponse{Count: 0}, safe, 15); err == nil {
			t.Fatal("expected an error for an empty result set")
		}
	})

	t.Run("exactly one result is returned", func(t *testing.T) {
		resp := APIResponse{Count: 1, Results: []SafeAPITransaction{{SafeTxHash: "0xaaa", To: "0xdead"}}}
		tx, err := selectQueuedTransaction(resp, safe, 15)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tx.SafeTxHash != "0xaaa" {
			t.Fatalf("got %q, want 0xaaa", tx.SafeTxHash)
		}
	})

	t.Run("two results at one nonce is an error naming both", func(t *testing.T) {
		resp := APIResponse{Count: 2, Results: []SafeAPITransaction{
			{SafeTxHash: "0xaaa"},
			{SafeTxHash: "0xbbb"},
		}}
		_, err := selectQueuedTransaction(resp, safe, 15)
		if err == nil {
			t.Fatal("expected an error for a nonce collision")
		}
		for _, want := range []string{"0xaaa", "0xbbb"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("error %q does not name candidate %s", err, want)
			}
		}
	})

	t.Run("a paginated collision is still an error", func(t *testing.T) {
		resp := APIResponse{Count: 2, Results: []SafeAPITransaction{{SafeTxHash: "0xaaa"}}}
		if _, err := selectQueuedTransaction(resp, safe, 15); err == nil {
			t.Fatal("expected an error when count exceeds the returned results")
		}
	})
}

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

	url, chain, err = getNetworkInfo("base")
	if err != nil {
		t.Fatalf("base: unexpected error: %v", err)
	}
	if url == "" || chain != BaseMainnetChainID {
		t.Fatalf("base: got (%q, %d), want (non-empty, %d)", url, chain, BaseMainnetChainID)
	}
}
