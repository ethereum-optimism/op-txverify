package core

import (
	"encoding/base64"
	"encoding/hex"
	"math/big"
	"testing"
)

func TestFastLZDecompress(t *testing.T) {
	tests := []struct {
		name       string
		compressed string
		want       string
	}{
		{
			name:       "literal run",
			compressed: "1068656c6c6f2068656c6c6f2068656c6c6f",
			want:       "hello hello hello",
		},
		{
			name:       "repeated json",
			compressed: "0b7b2273616665223a22307830e01d001631222c22636861696e223a312c2276616c7565223a307d",
			want:       "{\"safe\":\"0x0000000000000000000000000000000000000001\",\"chain\":1,\"value\":0}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compressed, err := hex.DecodeString(tt.compressed)
			if err != nil {
				t.Fatal(err)
			}
			got, err := FastLZDecompress(compressed)
			if err != nil {
				t.Fatalf("FastLZDecompress error = %v", err)
			}
			if string(got) != tt.want {
				t.Fatalf("FastLZDecompress = %q, want %q", string(got), tt.want)
			}
		})
	}
}

func TestFastLZDecompressInvalidInput(t *testing.T) {
	if _, err := FastLZDecompress([]byte{0x20}); err == nil {
		t.Fatal("expected truncated match error")
	}
	if _, err := FastLZDecompress([]byte{0x20, 0x00}); err == nil {
		t.Fatal("expected invalid distance error")
	}
}

func TestDecodeTransactionURL(t *testing.T) {
	const jsonPayload = `{"safe":"0x0000000000000000000000000000000000000001","chain":1,"value":0}`
	legacyURL := "https://op-txverify.optimism.io/?tx=" + base64.StdEncoding.EncodeToString([]byte(jsonPayload))
	compressedURL := "https://op-txverify.optimism.io/?txz=C3sic2FmZSI6IjB4MOAdABYxIiwiY2hhaW4iOjEsInZhbHVlIjowfQ"

	for _, rawURL := range []string{legacyURL, compressedURL} {
		tx, err := DecodeTransactionURL(rawURL)
		if err != nil {
			t.Fatalf("DecodeTransactionURL(%q) error = %v", rawURL, err)
		}
		if tx.Safe != "0x0000000000000000000000000000000000000001" {
			t.Fatalf("Safe = %q", tx.Safe)
		}
		if tx.Chain != 1 {
			t.Fatalf("Chain = %d", tx.Chain)
		}
		if tx.Value == nil || tx.Value.Cmp(big.NewInt(0)) != 0 {
			t.Fatalf("Value = %v", tx.Value)
		}
	}
}

func TestDecodeTransactionURLMissingPayload(t *testing.T) {
	if _, err := DecodeTransactionURL("https://op-txverify.optimism.io/"); err == nil {
		t.Fatal("expected missing payload error")
	}
}
