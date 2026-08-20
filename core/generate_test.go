package core

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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

const (
	testOuterSafe = "0x847B5c174615B1B7fDF770882256e2D3E95b9D92"
	testInnerSafe = "0x5a0Aae59D09fccBdDb6C6CcEB07B7279367C3d2A"
	testInnerHash = "0xbefcc37ec0dd42e4ebe3c6389c4929047b958910f3cf37376174d1f43b882e9f"
	testZeroAddr  = "0x0000000000000000000000000000000000000000"
)

// innerTxJSON is the v2 response for the approved transaction, with the fields under test
// overridden. The API returns every numeric field as a decimal string.
func innerTxJSON(overrides map[string]string) string {
	fields := map[string]string{
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

// safeAPI serves the three endpoints a nested transaction is assembled from.
func safeAPI(t *testing.T, innerTx string) *httptest.Server {
	t.Helper()
	outerTx := fmt.Sprintf(`{"count":1,"results":[{"to":%q,"value":"0","data":%q,"operation":0,`+
		`"safeTxGas":0,"baseGas":0,"gasPrice":"0","gasToken":%q,"refundReceiver":%q}]}`,
		testInnerSafe, "0xd4d9bdcd"+strings.TrimPrefix(testInnerHash, "0x"), testZeroAddr, testZeroAddr)

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/safes/" + testOuterSafe + "/", "/api/v1/safes/" + testInnerSafe + "/":
			fmt.Fprint(w, `{"version":"1.3.0"}`)
		case "/api/v1/safes/" + testOuterSafe + "/multisig-transactions/":
			fmt.Fprint(w, outerTx)
		case "/api/v2/multisig-transactions/" + testInnerHash + "/":
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
