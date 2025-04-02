package core

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

const (
	// EIP-712 constants
	DomainSeparatorTypehash = "0x47e79534a245952e8b16893a336b85a3d9ea9fa8c573f3d803afb92a79469218"
	SafeTxTypehash          = "0xbb8310d486368db6bd6f849402fdd73ad53d316b5a4b2644ad6efe0f941286d8"

	// Function signatures needed for parseMulticall
	SafeMultisendSig   = "multiSend(bytes)"
	Aggregate3Sig      = "aggregate3((address,bool,bytes)[])"
	Aggregate3ValueSig = "aggregate3Value((address,bool,uint256,bytes)[])"
)

// Known contract addresses
const (
	SafeMultisendAddress     = "0xA1dabEF33b3B82c7814B6D82A79e50F4AC44102B"
	SafeMultisendCallOnly141 = "0x9641d764fc13c8B624c04430C7356C1C7C8102e2"
	Multicall3Address        = "0xcA11bde05977b3631167028862bE2a173976CA11"
	Multicall3Delegatecall   = "0x93dc480940585D9961bfcEab58124fFD3d60f76a"
	USDCMainnetAddress       = "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"
	OPTokenAddress           = "0x4200000000000000000000000000000000000042"
	SuperfluidOP             = "0x1828Bff08BD244F7990edDCd9B19cc654b33cDB4"
	OptimismGovernor         = "0xcDF27F107725988f2261Ce2256bDfCdE8B382B10"
	OPGrants1                = "0x2501c477D0A35545a387Aa4A3EEe4292A9a8B3F0"
	OPGrants2                = "0x19793c7824Be70ec58BB673CA42D2779d12581BE"
	ProxyAdminOwner          = "0x5a0Aae59D09fccBdDb6C6CcEB07B7279367C3d2A"
)

// Functions on ERC20 tokens that require decimal adjustment
var TokenFunctions = map[string]bool{
	"transfer":          true,
	"transferFrom":      true,
	"approve":           true,
	"increaseAllowance": true,
	"decreaseAllowance": true,
}

// ContractInfo stores information about a known contract
type ContractInfo struct {
	Name     string
	Decimals int
}

// FunctionInfo stores information about a known function
type FunctionInfo struct {
	Name      string
	Signature string
	ABI       abi.Method
}

// ChainID constants for supported networks
const (
	MainnetChainID   = 1
	OPMainnetChainID = 10
	SepoliaChainID   = 11155111
	OPSepoliaChainID = 11155420
)

// ChainNames maps chain IDs to their names
var ChainNames = map[uint64]string{
	MainnetChainID:   "Ethereum",
	OPMainnetChainID: "OP Mainnet",
	SepoliaChainID:   "Sepolia",
	OPSepoliaChainID: "OP Sepolia",
}

// KnownContracts maps chain IDs to a map of addresses to contract info
var KnownContracts = map[uint64]map[string]ContractInfo{
	MainnetChainID: {
		strings.ToLower(SafeMultisendAddress):   {Name: "GNOSIS SAFE MULTISEND", Decimals: 0},
		strings.ToLower(Multicall3Address):      {Name: "MULTICALL3", Decimals: 0},
		strings.ToLower(Multicall3Delegatecall): {Name: "MULTICALL3 DELEGATECALL", Decimals: 0},
		strings.ToLower(USDCMainnetAddress):     {Name: "USDC", Decimals: 6},
		strings.ToLower(ProxyAdminOwner):        {Name: "SUPERCHAIN PROXY ADMIN OWNER", Decimals: 0},
	},
	OPMainnetChainID: {
		strings.ToLower(SafeMultisendAddress):     {Name: "GNOSIS SAFE MULTISEND", Decimals: 0},
		strings.ToLower(SafeMultisendCallOnly141): {Name: "GNOSIS SAFE MULTISEND", Decimals: 0},
		strings.ToLower(Multicall3Address):        {Name: "MULTICALL3", Decimals: 0},
		strings.ToLower(Multicall3Delegatecall):   {Name: "MULTICALL3 DELEGATECALL", Decimals: 0},
		strings.ToLower(OPTokenAddress):           {Name: "OP TOKEN", Decimals: 18},
		strings.ToLower(SuperfluidOP):             {Name: "SUPERFLUID OP", Decimals: 18},
		strings.ToLower(OptimismGovernor):         {Name: "OPTIMISM GOVERNOR", Decimals: 0},
		strings.ToLower(OPGrants1):                {Name: "OP GRANTS 1 (3F0)", Decimals: 0},
		strings.ToLower(OPGrants2):                {Name: "OP GRANTS 2 (1BE)", Decimals: 0},
	},
	SepoliaChainID: {
		strings.ToLower(SafeMultisendAddress):   {Name: "GNOSIS SAFE MULTISEND", Decimals: 0},
		strings.ToLower(Multicall3Address):      {Name: "MULTICALL3", Decimals: 0},
		strings.ToLower(Multicall3Delegatecall): {Name: "MULTICALL3 DELEGATECALL", Decimals: 0},
	},
	OPSepoliaChainID: {
		strings.ToLower(SafeMultisendAddress):   {Name: "GNOSIS SAFE MULTISEND", Decimals: 0},
		strings.ToLower(Multicall3Address):      {Name: "MULTICALL3", Decimals: 0},
		strings.ToLower(Multicall3Delegatecall): {Name: "MULTICALL3 DELEGATECALL", Decimals: 0},
	},
}

// MulticallAddresses maps chain IDs to a set of addresses known to be multicall contracts
var MulticallAddresses = map[uint64]map[string]bool{
	MainnetChainID: {
		strings.ToLower(SafeMultisendAddress):   true,
		strings.ToLower(Multicall3Address):      true,
		strings.ToLower(Multicall3Delegatecall): true,
	},
	OPMainnetChainID: {
		strings.ToLower(SafeMultisendAddress):     true,
		strings.ToLower(SafeMultisendCallOnly141): true,
		strings.ToLower(Multicall3Address):        true,
		strings.ToLower(Multicall3Delegatecall):   true,
	},
	SepoliaChainID: {
		strings.ToLower(SafeMultisendAddress):   true,
		strings.ToLower(Multicall3Address):      true,
		strings.ToLower(Multicall3Delegatecall): true,
	},
	OPSepoliaChainID: {
		strings.ToLower(SafeMultisendAddress):   true,
		strings.ToLower(Multicall3Address):      true,
		strings.ToLower(Multicall3Delegatecall): true,
	},
}

// KnownFunctions maps function selectors to function info
var KnownFunctions = make(map[string]FunctionInfo)

// KnownABIJSON contains the JSON ABIs for common functions
var KnownABIJSON = []string{
	`[{"inputs":[{"name":"to","type":"address"},{"name":"amount","type":"uint256"}],"name":"transfer","type":"function"}]`,
	`[{"inputs":[{"name":"from","type":"address"},{"name":"to","type":"address"},{"name":"amount","type":"uint256"}],"name":"transferFrom","type":"function"}]`,
	`[{"inputs":[{"name":"spender","type":"address"},{"name":"amount","type":"uint256"}],"name":"approve","type":"function"}]`,
	`[{"inputs":[{"name":"spender","type":"address"},{"name":"amount","type":"uint256"}],"name":"increaseAllowance","type":"function"}]`,
	`[{"inputs":[{"name":"spender","type":"address"},{"name":"amount","type":"uint256"}],"name":"decreaseAllowance","type":"function"}]`,
	`[{"inputs":[{"name":"hashToApprove","type":"bytes32"}],"name":"approveHash","type":"function"}]`,
	`[{"inputs":[{"name":"calls","type":"tuple[]","components":[{"name":"target","type":"address"},{"name":"allowFailure","type":"bool"},{"name":"callData","type":"bytes"}]}],"name":"aggregate3","type":"function"}]`,
	`[{"inputs":[{"name":"calls","type":"tuple[]","components":[{"name":"target","type":"address"},{"name":"allowFailure","type":"bool"},{"name":"value","type":"uint256"},{"name":"callData","type":"bytes"}]}],"name":"aggregate3Value","type":"function"}]`,
	`[{"inputs":[{"name":"transactions","type":"bytes"}],"name":"multiSend","type":"function"}]`,
	`[{"inputs":[{"name":"agreementClass","type":"address"},{"name":"callData","type":"bytes"},{"name":"userData","type":"bytes"}],"name":"callAgreement","type":"function"}]`,
	`[{"inputs":[{"name":"superToken","type":"address"},{"name":"receiver","type":"address"},{"name":"totalAmount","type":"uint256"},{"name":"totalDuration","type":"uint32"},{"name":"startDate","type":"uint32"},{"name":"cliffPeriod","type":"uint32"},{"name":"claimPeriod","type":"uint32"}],"name":"createVestingScheduleFromAmountAndDuration","type":"function"}]`,
	`[{"inputs":[{"name":"targets","type":"address[]"},{"name":"values","type":"uint256[]"},{"name":"calldatas","type":"bytes[]"},{"name":"description","type":"string"},{"name":"proposalType","type":"uint8"}],"name":"propose","type":"function"}]`,
	`[{"inputs":[{"name":"opChainConfigs","type":"tuple[]","components":[{"name":"systemConfigProxy","type":"address"},{"name":"proxyAdmin","type":"address"},{"name":"absolutePrestate","type":"bytes32"}]}],"name":"upgrade","type":"function"}]`,
	`[{"inputs":[{"components":[{"name":"schema","type":"bytes32"},{"components":[{"name":"recipient","type":"address"},{"name":"expirationTime","type":"uint64"},{"name":"revocable","type":"bool"},{"name":"refUID","type":"bytes32"},{"name":"data","type":"bytes"},{"name":"value","type":"uint256"}],"name":"data","type":"tuple"}],"name":"request","type":"tuple"}],"name":"attest","type":"function"}]`,
	`[{"inputs":[{"name":"owner","type":"address"},{"name":"threshold","type":"uint256"}],"name":"addOwnerWithThreshold","type":"function"}]`,
	`[{"inputs":[{"name":"prevOwner","type":"address"},{"name":"owner","type":"address"},{"name":"threshold","type":"uint256"}],"name":"removeOwner","type":"function"}]`,
}

// Initialize known functions
func init() {
	// Parse each ABI JSON and add to KnownFunctions
	for _, abiJSON := range KnownABIJSON {
		parsedABI, err := abi.JSON(strings.NewReader(abiJSON))
		if err != nil {
			// Log error but continue
			fmt.Printf("Error parsing ABI JSON: %v\n", err)
			continue
		}

		// There should be only one method in each ABI
		for name, method := range parsedABI.Methods {
			// Add to known functions map using the selector hex
			selector := hex.EncodeToString(method.ID[:])

			KnownFunctions[selector] = FunctionInfo{
				Name:      name,
				Signature: method.Sig,
				ABI:       method,
			}
		}
	}
}

func GetKnownContract(address string, chainID uint64) (ContractInfo, bool) {
	normalizedAddr := strings.ToLower(address)
	if chainContracts, exists := KnownContracts[chainID]; exists {
		contractInfo, isKnown := chainContracts[normalizedAddr]
		return contractInfo, isKnown
	}
	return ContractInfo{}, false
}
