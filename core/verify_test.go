package core

import (
	"encoding/json"
	"math/big"
	"testing"
)

func TestSafeTransactionValueUnmarshal(t *testing.T) {
	maxUint256, _ := new(big.Int).SetString("115792089237316195423570985008687907853269984665640564039457584007913129639935", 10)

	testCases := []struct {
		name     string
		json     string
		expected *big.Int
	}{
		// A string above 2^53-1 wei is the case the writer page used to corrupt via parseInt.
		{"decimal string", `{"value":"1000000000000000000"}`, big.NewInt(1000000000000000000)},
		{"bare number", `{"value":1000000000000000000}`, big.NewInt(1000000000000000000)},
		{"max uint256 string", `{"value":"115792089237316195423570985008687907853269984665640564039457584007913129639935"}`, maxUint256},
		{"zero string", `{"value":"0"}`, big.NewInt(0)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var tx SafeTransaction
			if err := json.Unmarshal([]byte(tc.json), &tx); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}
			if tx.Value == nil || tx.Value.Cmp(tc.expected) != 0 {
				t.Errorf("Value mismatch. Got %v, want %s", tx.Value, tc.expected)
			}
		})
	}

	// Offline transaction files and compressed URL payloads omit value for zero-value transactions.
	// It must land on zero, not nil: a nil Value panics when the struct hash is ABI-packed.
	for _, payload := range []string{`{}`, `{"value":null}`} {
		var tx SafeTransaction
		if err := json.Unmarshal([]byte(payload), &tx); err != nil {
			t.Errorf("Failed to unmarshal %s: %v", payload, err)
			continue
		}
		if tx.Value == nil || tx.Value.Sign() != 0 {
			t.Errorf("Expected zero for %s, got %v", payload, tx.Value)
		}
	}

	// Garbage and out-of-range values must fail loudly. abi.Pack reduces modulo 2^256 rather than
	// erroring, so a negative or oversized value would print one number and hash another.
	rejected := []string{
		`{"value":"not-a-number"}`,
		`{"value":"0x1234"}`,
		`{"value":true}`,
		`{"value":-1}`,
		`{"value":"-1"}`,
		`{"value":"115792089237316195423570985008687907853269984665640564039457584007913129639936"}`,
	}
	for _, payload := range rejected {
		var tx SafeTransaction
		if err := json.Unmarshal([]byte(payload), &tx); err == nil {
			t.Errorf("Expected an error for %s, got %v", payload, tx.Value)
		}
	}
}
