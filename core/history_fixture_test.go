package core

import (
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
)

const historyFixtureSource = "https://github.com/ethereum-optimism/op-txverify/commit/9126b14"

// historySelectorFixture is a compact, deterministic record of each previously
// unknown non-empty selector found while walking the reviewed Safe histories.
// The signatures in the task review are the source of truth; no Safe or
// explorer response is used during tests.
var historySelectorFixture = []struct {
	signature string
	chainID   uint64
	target    string
}{
	{"setRequired(uint256)", MainnetChainID, "0x3F3Cd78Ef9Bd85961C0729E6BbB11E94Ca6f61D2"},
	{"configureLivenessModule((uint256,address))", OPMainnetChainID, "0x3F3Cd78Ef9Bd85961C0729E6BbB11E94Ca6f61D2"},
	{"execute(bytes)", MainnetChainID, "0x9BA6e03D8B90dE867373Db8cF1A58d2F7F006b3A"},
	{"withdraw(uint256,uint256,address)", OPMainnetChainID, "0x4928B8F2399281140DBD8Ab9E4D12ecA9365CA6C"},
	{"setOwner(address)", MainnetChainID, "0x847B5c174615B1B7fDF770882256e2D3E95b9D92"},
	{"setImplementation(uint32,address)", MainnetChainID, "0x5a0Aae59D09fccBdDb6C6CcEB07B7279367C3d2A"},
	{"createProxyWithNonce(address,bytes,uint256)", MainnetChainID, "0x847B5c174615B1B7fDF770882256e2D3E95b9D92"},
	{"migrateEth(address)", MainnetChainID, "0x9BA6e03D8B90dE867373Db8cF1A58d2F7F006b3A"},
	{"setDeputy(address,bytes)", MainnetChainID, "0x9BA6e03D8B90dE867373Db8cF1A58d2F7F006b3A"},
	{"setUnsafeBlockSigner(address)", OPMainnetChainID, "0x4928B8F2399281140DBD8Ab9E4D12ecA9365CA6C"},
	{"setInitBond(uint32,uint256)", MainnetChainID, "0x5a0Aae59D09fccBdDb6C6CcEB07B7279367C3d2A"},
	{"unpause()", MainnetChainID, "0x9BA6e03D8B90dE867373Db8cF1A58d2F7F006b3A"},
	{"initialize(address,address)", MainnetChainID, "0x847B5c174615B1B7fDF770882256e2D3E95b9D92"},
	{"setBytes32(bytes32,bytes32)", MainnetChainID, "0x5a0Aae59D09fccBdDb6C6CcEB07B7279367C3d2A"},
	{"setRecommended(uint256)", MainnetChainID, "0x3F3Cd78Ef9Bd85961C0729E6BbB11E94Ca6f61D2"},
	{"enableModule(address)", MainnetChainID, "0x847B5c174615B1B7fDF770882256e2D3E95b9D92"},
	{"changeThreshold(uint256)", MainnetChainID, "0x847B5c174615B1B7fDF770882256e2D3E95b9D92"},
	{"pause()", MainnetChainID, "0x9BA6e03D8B90dE867373Db8cF1A58d2F7F006b3A"},
	{"supply(uint256,uint256,address)", OPMainnetChainID, "0x4928B8F2399281140DBD8Ab9E4D12ecA9365CA6C"},
	{"initialize(address,address,address,uint32)", OPMainnetChainID, "0x4928B8F2399281140DBD8Ab9E4D12ecA9365CA6C"},
	{"changeAdmin(address)", MainnetChainID, "0x5a0Aae59D09fccBdDb6C6CcEB07B7279367C3d2A"},
	{"setGasConfig(uint256,uint256)", OPMainnetChainID, "0x4928B8F2399281140DBD8Ab9E4D12ecA9365CA6C"},
	{"upgradeAndCall(address,address,bytes)", MainnetChainID, "0x5a0Aae59D09fccBdDb6C6CcEB07B7279367C3d2A"},
	{"upgrade(address,address)", MainnetChainID, "0x5a0Aae59D09fccBdDb6C6CcEB07B7279367C3d2A"},
	{"setAddress(string,address)", MainnetChainID, "0x5a0Aae59D09fccBdDb6C6CcEB07B7279367C3d2A"},
	{"setRespectedGameType(address,uint32)", MainnetChainID, "0x5a0Aae59D09fccBdDb6C6CcEB07B7279367C3d2A"},
	{"setImplementation(uint32,address,bytes)", MainnetChainID, "0x5a0Aae59D09fccBdDb6C6CcEB07B7279367C3d2A"},
	{"setGasLimit(uint64)", OPMainnetChainID, "0x4928B8F2399281140DBD8Ab9E4D12ecA9365CA6C"},
	{"setEIP1559Params(uint32,uint32)", OPMainnetChainID, "0x4928B8F2399281140DBD8Ab9E4D12ecA9365CA6C"},
	{"setBatcherHash(bytes32)", OPMainnetChainID, "0x4928B8F2399281140DBD8Ab9E4D12ecA9365CA6C"},
	{"setAddress(bytes32,address)", MainnetChainID, "0x5a0Aae59D09fccBdDb6C6CcEB07B7279367C3d2A"},
	{"updateDynamicConfig((uint256,uint256),bool)", OPMainnetChainID, "0x4928B8F2399281140DBD8Ab9E4D12ecA9365CA6C"},
	{"phase2()", MainnetChainID, "0x9BA6e03D8B90dE867373Db8cF1A58d2F7F006b3A"},
	{"disableModule(address,address)", MainnetChainID, "0x847B5c174615B1B7fDF770882256e2D3E95b9D92"},
	{"setGuard(address)", MainnetChainID, "0x847B5c174615B1B7fDF770882256e2D3E95b9D92"},
	{"phase1()", MainnetChainID, "0x9BA6e03D8B90dE867373Db8cF1A58d2F7F006b3A"},
	{"transferOwnership(address)", MainnetChainID, "0x5a0Aae59D09fccBdDb6C6CcEB07B7279367C3d2A"},
}

func TestHistorySelectorsAreKnown(t *testing.T) {
	for _, tt := range historySelectorFixture {
		t.Run(tt.signature, func(t *testing.T) {
			selector := hex.EncodeToString(crypto.Keccak256([]byte(tt.signature))[:4])
			function, ok := KnownFunctions[selector]
			if !ok {
				t.Fatalf("selector %s for %s is unknown", selector, tt.signature)
			}
			if function.Signature != tt.signature {
				t.Fatalf("selector %s signature = %s, want %s", selector, function.Signature, tt.signature)
			}

			call, err := ParseTransactionData(tt.target, historyRepresentativeCalldata(tt.signature), tt.chainID, VerifyOptions{})
			if err != nil {
				t.Fatalf("ParseTransactionData() error = %v", err)
			}
			if call.FunctionName == "unknown" {
				t.Fatalf("ParseTransactionData() left %s unknown", tt.signature)
			}
			if call.ParsedData == nil {
				t.Fatalf("ParseTransactionData() did not decode representative calldata for %s", tt.signature)
			}
			parsed, ok := call.ParsedData.(map[string]interface{})
			if !ok {
				t.Fatalf("ParseTransactionData() parsedData type = %T, want argument map", call.ParsedData)
			}
			gotNames := make([]string, 0, len(parsed))
			for name := range parsed {
				gotNames = append(gotNames, name)
			}
			sort.Strings(gotNames)
			wantNames := historyExpectedArgumentNames(tt.signature)
			sort.Strings(wantNames)
			if strings.Join(gotNames, ",") != strings.Join(wantNames, ",") {
				t.Fatalf("decoded argument names = %v, want %v", gotNames, wantNames)
			}
		})
	}
}

func historyExpectedArgumentNames(signature string) []string {
	switch signature {
	case "setRequired(uint256)", "setRecommended(uint256)":
		return []string{"required"}
	case "configureLivenessModule((uint256,address))":
		return []string{"config"}
	case "execute(bytes)":
		return []string{"data"}
	case "withdraw(uint256,uint256,address)", "supply(uint256,uint256,address)":
		return []string{"assets", "receiver", "shares"}
	case "setOwner(address)":
		return []string{"owner"}
	case "setImplementation(uint32,address)":
		return []string{"gameType", "implementation"}
	case "createProxyWithNonce(address,bytes,uint256)":
		return []string{"initializer", "saltNonce", "singleton"}
	case "migrateEth(address)":
		return []string{"recipient"}
	case "setDeputy(address,bytes)":
		return []string{"deputy", "permissions"}
	case "setUnsafeBlockSigner(address)":
		return []string{"signer"}
	case "setInitBond(uint32,uint256)":
		return []string{"gameType", "initBond"}
	case "unpause()", "pause()", "phase2()", "phase1()":
		return []string{}
	case "initialize(address,address)":
		return []string{"guardian", "owner"}
	case "setBytes32(bytes32,bytes32)", "setAddress(bytes32,address)", "setAddress(string,address)":
		return []string{"key", "value"}
	case "enableModule(address)":
		return []string{"module"}
	case "changeThreshold(uint256)":
		return []string{"threshold"}
	case "initialize(address,address,address,uint32)":
		return []string{"feeRecipient", "gasLimit", "oracle", "owner"}
	case "changeAdmin(address)":
		return []string{"admin"}
	case "setGasConfig(uint256,uint256)":
		return []string{"baseFee", "gasLimit"}
	case "upgradeAndCall(address,address,bytes)":
		return []string{"data", "implementation", "proxy"}
	case "upgrade(address,address)":
		return []string{"implementation", "proxy"}
	case "setRespectedGameType(address,uint32)":
		return []string{"disputeGameFactory", "gameType"}
	case "setImplementation(uint32,address,bytes)":
		return []string{"gameType", "implementation", "initData"}
	case "setGasLimit(uint64)":
		return []string{"gasLimit"}
	case "setEIP1559Params(uint32,uint32)":
		return []string{"denominator", "elasticity"}
	case "setBatcherHash(bytes32)":
		return []string{"batcherHash"}
	case "updateDynamicConfig((uint256,uint256),bool)":
		return []string{"config", "isEcotone"}
	case "disableModule(address,address)":
		return []string{"module", "prevModule"}
	case "setGuard(address)":
		return []string{"guard"}
	case "transferOwnership(address)":
		return []string{"newOwner"}
	default:
		panic(fmt.Sprintf("no expected argument names for %s", signature))
	}
}

func TestHistoryFixtureCoversEachNewSignatureOnce(t *testing.T) {
	seen := make(map[string]bool)
	for _, tt := range historySelectorFixture {
		if seen[tt.signature] {
			t.Fatalf("duplicate fixture signature %s", tt.signature)
		}
		seen[tt.signature] = true
		if tt.target == "" || !strings.HasPrefix(tt.target, "0x") {
			t.Fatalf("fixture target for %s is not an address: %q", tt.signature, tt.target)
		}
	}
	if len(seen) != 37 {
		t.Fatalf("fixture has %d signatures, want 37", len(seen))
	}
	if !strings.HasPrefix(historyFixtureSource, "https://github.com/") {
		t.Fatalf("fixture source is not a source URL: %q", historyFixtureSource)
	}
}

// historyRepresentativeCalldata is deliberately compact: all values are zero,
// except dynamic offsets, which makes every sample ABI-valid while keeping the
// expected method signature and decoded parameter layout deterministic.
func historyRepresentativeCalldata(signature string) string {
	words, dynamicAt := 0, -1
	switch signature {
	case "setRequired(uint256)", "execute(bytes)", "setOwner(address)", "migrateEth(address)", "setUnsafeBlockSigner(address)", "setRecommended(uint256)", "enableModule(address)", "changeThreshold(uint256)", "changeAdmin(address)", "setGasLimit(uint64)", "setBatcherHash(bytes32)", "setGuard(address)", "transferOwnership(address)":
		words = 1
	case "configureLivenessModule((uint256,address))", "setImplementation(uint32,address)", "setDeputy(address,bytes)", "setInitBond(uint32,uint256)", "initialize(address,address)", "setBytes32(bytes32,bytes32)", "setGasConfig(uint256,uint256)", "upgrade(address,address)", "setAddress(string,address)", "setRespectedGameType(address,uint32)", "setEIP1559Params(uint32,uint32)", "setAddress(bytes32,address)", "disableModule(address,address)":
		words = 2
	case "withdraw(uint256,uint256,address)", "createProxyWithNonce(address,bytes,uint256)", "upgradeAndCall(address,address,bytes)", "setImplementation(uint32,address,bytes)", "supply(uint256,uint256,address)":
		words = 3
	case "updateDynamicConfig((uint256,uint256),bool)":
		words = 3
	case "initialize(address,address,address,uint32)":
		words = 4
	case "unpause()", "pause()", "phase2()", "phase1()":
		words = 0
	default:
		panic(fmt.Sprintf("no representative layout for %s", signature))
	}
	switch signature {
	case "execute(bytes)":
		dynamicAt = 0
	case "setDeputy(address,bytes)", "setAddress(string,address)":
		dynamicAt = 1
	case "createProxyWithNonce(address,bytes,uint256)":
		dynamicAt = 1
	case "upgradeAndCall(address,address,bytes)", "setImplementation(uint32,address,bytes)":
		dynamicAt = 2
	}

	selector := crypto.Keccak256([]byte(signature))[:4]
	data := make([]byte, 4+32*words)
	copy(data, selector)
	if dynamicAt >= 0 {
		data[4+32*dynamicAt+31] = byte(32 * words)
		data = append(data, make([]byte, 32)...)
	}
	return "0x" + hex.EncodeToString(data)
}

func TestBuildKnownFunctionsRejectsConflictingSelectors(t *testing.T) {
	records := []KnownABIRecord{
		{Signature: "burn(uint256)", ParameterNames: []string{"amount"}, Source: "test", ABIJSON: `[{"inputs":[{"name":"amount","type":"uint256"}],"name":"burn","type":"function"}]`},
		{Signature: "collate_propagate_storage(bytes16)", ParameterNames: []string{"data"}, Source: "test", ABIJSON: `[{"inputs":[{"name":"data","type":"bytes16"}],"name":"collate_propagate_storage","type":"function"}]`},
	}
	_, err := BuildKnownFunctions(records)
	if err == nil || !strings.Contains(err.Error(), "conflicts") {
		t.Fatalf("BuildKnownFunctions() error = %v, want selector conflict", err)
	}
}
