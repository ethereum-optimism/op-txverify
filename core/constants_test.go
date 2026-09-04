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

func TestOPCMv800Known(t *testing.T) {
	for _, tc := range []struct {
		addr    string
		chainID uint64
	}{
		{OPCMv800Mainnet, MainnetChainID},
		{OPCMv800Sepolia, SepoliaChainID},
	} {
		info, ok := GetKnownContract(tc.addr, tc.chainID)
		if !ok {
			t.Fatalf("expected %s to be known on chain %d", tc.addr, tc.chainID)
		}
		if info.Name != "OPContractsManager V8.0.0" {
			t.Fatalf("unexpected name: %q", info.Name)
		}
	}
}

func TestSetInteropDisputeGamesSelector(t *testing.T) {
	// OPContractsManagerV2.setInteropDisputeGames, new in op-contracts/v8.0.0.
	info, ok := KnownFunctions["73aaee86"]
	if !ok {
		t.Fatal("expected setInteropDisputeGames selector to be known")
	}
	want := "setInteropDisputeGames((address[],(bool,uint256,uint32,bytes)[],(bytes32,uint256),uint32))"
	if info.Signature != want {
		t.Fatalf("got %q, want %q", info.Signature, want)
	}
}
