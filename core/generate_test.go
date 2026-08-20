package core

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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

const (
	testOuterSafe = "0x847B5c174615B1B7fDF770882256e2D3E95b9D92"
	testInnerSafe = "0x5a0Aae59D09fccBdDb6C6CcEB07B7279367C3d2A"
	testInnerHash = "0xbefcc37ec0dd42e4ebe3c6389c4929047b958910f3cf37376174d1f43b882e9f"
	testOuterHash = "0x1c0f1b5ee7e4b0a4c31d95c5a37c0b0e4f5cd3ca2b6a0e1d9f8c7b6a5f4e3d2c"
	testZeroAddr  = "0x0000000000000000000000000000000000000000"
)

// innerTxJSON is the v2 response for the approved transaction, with the fields under test
// overridden. The API returns every numeric field as a decimal string.
func innerTxJSON(overrides map[string]string) string {
	fields := map[string]string{
		"safeTxHash":     testInnerHash,
		"safe":           testInnerSafe,
		"to":             OPL1StandardBridge,
		"value":          "0",
		"data":           "0x",
		"safeTxGas":      "0",
		"baseGas":        "0",
		"gasPrice":       "0",
		"gasToken":       testZeroAddr,
		"refundReceiver": testZeroAddr,
		"nonce":          "39",
	}
	for field, value := range overrides {
		fields[field] = value
	}
	body, err := json.Marshal(fields)
	if err != nil {
		panic(err)
	}
	return string(body)
}

// outerTxJSON is the nonce-lookup response for the approveHash transaction, with the fields under
// test overridden.
func outerTxJSON(overrides map[string]string) string {
	fields := map[string]string{
		"safeTxHash":     testOuterHash,
		"to":             testInnerSafe,
		"value":          "0",
		"data":           "0xd4d9bdcd" + strings.TrimPrefix(testInnerHash, "0x"),
		"gasPrice":       "0",
		"gasToken":       testZeroAddr,
		"refundReceiver": testZeroAddr,
	}
	for field, value := range overrides {
		fields[field] = value
	}
	body, err := json.Marshal(map[string]any{
		"count":   1,
		"results": []any{fields},
	})
	if err != nil {
		panic(err)
	}
	return string(body)
}

// safeAPI serves the three endpoints a nested transaction is assembled from.
func safeAPI(t *testing.T, innerTx string) *httptest.Server {
	t.Helper()
	return safeAPIWith(t, outerTxJSON(nil), innerTx, http.StatusOK)
}

// safeAPIWith is safeAPI with the outer response and the inner endpoint's status supplied.
func safeAPIWith(t *testing.T, outerTx, innerTx string, innerStatus int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/safes/" + testOuterSafe + "/", "/api/v1/safes/" + testInnerSafe + "/":
			fmt.Fprint(w, `{"version":"1.3.0"}`)
		case "/api/v1/safes/" + testOuterSafe + "/multisig-transactions/":
			fmt.Fprint(w, outerTx)
		case "/api/v2/multisig-transactions/" + testInnerHash + "/":
			w.WriteHeader(innerStatus)
			fmt.Fprint(w, innerTx)
		default:
			t.Errorf("unexpected request for %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestGenerateTransaction_HashesTheInnerNonce(t *testing.T) {
	server := safeAPI(t, innerTxJSON(nil))
	defer server.Close()

	tx, err := generateTransaction(server.URL, MainnetChainID, testOuterSafe, 65)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tx.Safe != testInnerSafe || tx.Nonce != 39 {
		t.Errorf("hashed transaction = %s nonce %d, want the inner safe %s nonce 39", tx.Safe, tx.Nonce, testInnerSafe)
	}
	if tx.Nested == nil {
		t.Fatal("nested transaction missing")
	}
	if tx.Nested.Safe != testOuterSafe || tx.Nested.Nonce != 65 {
		t.Errorf("nested transaction = %s nonce %d, want the outer safe %s nonce 65", tx.Nested.Safe, tx.Nested.Nonce, testOuterSafe)
	}
}

func TestGenerateTransaction_RejectsMalformedNumbers(t *testing.T) {
	tests := []struct{ field, value string }{
		{"nonce", "12abc"},
		{"nonce", ""},
		{"safeTxGas", "12abc"},
		{"baseGas", "null"},
		{"gasPrice", "18446744073709551615"},
	}

	for _, tc := range tests {
		t.Run(tc.field+"="+tc.value, func(t *testing.T) {
			server := safeAPI(t, innerTxJSON(map[string]string{tc.field: tc.value}))
			defer server.Close()

			tx, err := generateTransaction(server.URL, MainnetChainID, testOuterSafe, 65)
			if err == nil {
				t.Fatalf("expected rejection, got %s %d", tc.field, tx.Nonce)
			}
			t.Log(err)
		})
	}
}

func TestGenerateTransaction_CarriesTheHashesTheFieldsAreBoundTo(t *testing.T) {
	server := safeAPI(t, innerTxJSON(nil))
	defer server.Close()

	tx, err := generateTransaction(server.URL, MainnetChainID, testOuterSafe, 65)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tx.SafeTxHash != testInnerHash {
		t.Errorf("hashed transaction is bound to %s, want the approved transaction %s", tx.SafeTxHash, testInnerHash)
	}
	if tx.Nested.SafeTxHash != testOuterHash {
		t.Errorf("nested transaction is bound to %s, want the queued transaction %s", tx.Nested.SafeTxHash, testOuterHash)
	}
}

// Everything below used to produce a transaction the tool went on to hash and display as if it had
// read what it asked for.
func TestGenerateTransaction_RejectsUnboundResponses(t *testing.T) {
	otherHash := "0x" + strings.Repeat("9", 64)

	tests := []struct {
		name        string
		outerTx     string
		innerTx     string
		innerStatus int
		want        string
	}{
		{
			name:        "the queued transaction carries no hash",
			outerTx:     outerTxJSON(map[string]string{"safeTxHash": ""}),
			innerTx:     innerTxJSON(nil),
			innerStatus: http.StatusOK,
			want:        "cannot be bound to a hash",
		},
		{
			name:        "the approved transaction is not the one the calldata names",
			outerTx:     outerTxJSON(nil),
			innerTx:     innerTxJSON(map[string]string{"safeTxHash": otherHash}),
			innerStatus: http.StatusOK,
			want:        "returned",
		},
		{
			name:        "the service is down for the approved transaction",
			outerTx:     outerTxJSON(nil),
			innerTx:     `{"detail":"unavailable"}`,
			innerStatus: http.StatusServiceUnavailable,
			want:        "503",
		},
		{
			name:        "the approveHash argument is truncated",
			outerTx:     outerTxJSON(map[string]string{"data": "0xd4d9bdcdbeef"}),
			innerTx:     innerTxJSON(nil),
			innerStatus: http.StatusOK,
			want:        "too short",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := safeAPIWith(t, tc.outerTx, tc.innerTx, tc.innerStatus)
			defer server.Close()

			tx, err := generateTransaction(server.URL, MainnetChainID, testOuterSafe, 65)
			if err == nil {
				t.Fatalf("expected rejection, got a transaction on %s nonce %d", tx.Safe, tx.Nonce)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error does not mention %q: %v", tc.want, err)
			}
		})
	}
}
