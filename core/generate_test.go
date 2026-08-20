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

// Asserting the exact URL matters as much as the chain id: if a network were mapped to another
// network's tx service, a non-empty check would still pass while the CLI queried the wrong chain.
func TestGetNetworkInfo(t *testing.T) {
	testCases := []struct {
		network string
		url     string
		chainID uint64
	}{
		{"ethereum", "https://api.safe.global/tx-service/eth", MainnetChainID},
		{"op", "https://api.safe.global/tx-service/oeth", OPMainnetChainID},
		{"optimism", "https://api.safe.global/tx-service/oeth", OPMainnetChainID},
		{"base", "https://api.safe.global/tx-service/base", BaseMainnetChainID},
		{"sepolia", "https://api.safe.global/tx-service/sep", SepoliaChainID},
		{"ETHEREUM", "https://api.safe.global/tx-service/eth", MainnetChainID},
	}

	for _, tc := range testCases {
		t.Run(tc.network, func(t *testing.T) {
			url, chain, err := getNetworkInfo(tc.network)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if url != tc.url {
				t.Errorf("URL mismatch. Got %q, want %q", url, tc.url)
			}
			if chain != tc.chainID {
				t.Errorf("chain id mismatch. Got %d, want %d", chain, tc.chainID)
			}
		})
	}

	// The 4 distinct networks must not share a tx service, so a copy-paste slip is caught.
	seen := map[string]string{}
	for _, network := range []string{"ethereum", "op", "base", "sepolia"} {
		url, _, err := getNetworkInfo(network)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", network, err)
		}
		if prior, ok := seen[url]; ok {
			t.Errorf("%s and %s share the tx service %q", prior, network, url)
		}
		seen[url] = network
	}

	if _, _, err := getNetworkInfo("opsep"); err == nil {
		t.Error("expected an error for an unsupported network")
	}
}
