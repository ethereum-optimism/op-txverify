package core

import (
	"testing"
)

func TestCalculateDomainHash(t *testing.T) {
	testCases := []struct {
		name     string
		tx       SafeTransaction
		expected string
	}{
		{
			name: "Ethereum Mainnet Safe 1",
			tx: SafeTransaction{
				Safe:  "0x847B5c174615B1B7fDF770882256e2D3E95b9D92",
				Chain: 1,
			},
			expected: "0xa4a9c312badf3fcaa05eafe5dc9bee8bd9316c78ee8b0bebe3115bb21b732672",
		},
		{
			name: "OP Mainnet Safe 1",
			tx: SafeTransaction{
				Safe:  "0xE2Ed962948005AB01F2cEfE8326a0730B7D268af",
				Chain: 10,
			},
			expected: "0x2d170d0f028e39b4926fa91cf7bbb44ef0e203f3995905ec6f1bdd3c657edfb3",
		},
		{
			name: "OP Mainnet Safe 2",
			tx: SafeTransaction{
				Safe:  "0x2501c477D0A35545a387Aa4A3EEe4292A9a8B3F0",
				Chain: 10,
			},
			expected: "0xb34978142f4478f3e5633915597a756daa58a1a59a3e0234f9acd5444f1ca70e",
		},
		{
			name: "Sepolia Safe 1",
			tx: SafeTransaction{
				Safe:  "0xf64bc17485f0B4Ea5F06A96514182FC4cB561977",
				Chain: 11155111,
			},
			expected: "0xbe081970e9fc104bd1ea27e375cd21ec7bb1eec56bfe43347c3e36c5d27b8533",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hash, err := CalculateDomainHash(tc.tx)
			if err != nil {
				t.Fatalf("Failed to calculate domain hash: %v", err)
			}

			if hash != tc.expected {
				t.Errorf("Domain hash mismatch. Got %s, want %s", hash, tc.expected)
			}
		})
	}
}

func TestCalculateMessageHash(t *testing.T) {
	testCases := []struct {
		name     string
		tx       SafeTransaction
		expected string
	}{
		{
			name: "Ethereum Mainnet Safe Tx 1",
			tx: SafeTransaction{
				Safe:           "0x847B5c174615B1B7fDF770882256e2D3E95b9D92",
				To:             "0xcA11bde05977b3631167028862bE2a173976CA11",
				Value:          0,
				Data:           "0x82ad56cb0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000200000000000000000000000005a0aae59d09fccbddb6c6cceb07b7279367c3d2a000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000000000000000024d4d9bdcd493ad64b8f788ed9808c7bf527a10a017d9f263bb7889868ce18b451d685762d00000000000000000000000000000000000000000000000000000000",
				Operation:      1,
				SafeTxGas:      0,
				BaseGas:        0,
				GasPrice:       0,
				GasToken:       "0x0000000000000000000000000000000000000000",
				RefundReceiver: "0x0000000000000000000000000000000000000000",
				Nonce:          15,
				Chain:          1,
			},
			expected: "0xa6d60aba6b1426cec097593348a9d36ed42ddd1ac52f1b05a5f46a9c0401a11a",
		},
		{
			name: "Sepolia Safe Tx 1",
			tx: SafeTransaction{
				Safe:           "0xf64bc17485f0B4Ea5F06A96514182FC4cB561977",
				To:             "0xcA11bde05977b3631167028862bE2a173976CA11",
				Value:          0,
				Data:           "0x82ad56cb0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000200000000000000000000000001eb2ffc903729a0f03966b917003800b145f56e2000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000000000000000024d4d9bdcd076db0a8758739afdd098a3d9fed5147eb55f363cd85167c1b3e5f334d317f3e00000000000000000000000000000000000000000000000000000000",
				Operation:      1,
				SafeTxGas:      0,
				BaseGas:        0,
				GasPrice:       0,
				GasToken:       "0x0000000000000000000000000000000000000000",
				RefundReceiver: "0x0000000000000000000000000000000000000000",
				Nonce:          30,
				Chain:          11155111,
			},
			expected: "0x2044f4436f0a27ce0697bc3fadb46ee88568d74fe8abaf1a6a31ce5ecf888c5a",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hash, err := CalculateMessageHash(tc.tx)
			if err != nil {
				t.Fatalf("Failed to calculate message hash: %v", err)
			}

			if hash != tc.expected {
				t.Errorf("Message hash mismatch. Got %s, want %s", hash, tc.expected)
			}
		})
	}
}
