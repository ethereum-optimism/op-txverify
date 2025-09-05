package core

import "testing"

func TestGetKnownContract(t *testing.T) {
	// Known on OP Mainnet
	info, ok := GetKnownContract(OPTokenAddress, OPMainnetChainID)
	if !ok {
		t.Fatalf("expected OP token to be known on OP Mainnet")
	}
	if info.Name == "" || info.Decimals != 18 {
		t.Fatalf("unexpected info: %+v", info)
	}

	// Not known on Ethereum mainnet (OP token address)
	if _, ok := GetKnownContract(OPTokenAddress, MainnetChainID); ok {
		t.Fatalf("did not expect OP token to be known on Ethereum Mainnet")
	}
}
