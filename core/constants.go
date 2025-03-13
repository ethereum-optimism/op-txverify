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
	SafeMultisendSig = "multiSend(bytes transactions)"
	Aggregate3Sig    = "aggregate3((address target,bool allowFailure,bytes callData)[] calls)"
)

// Known contract addresses
const (
	SafeMultisendAddress     = "0xA1dabEF33b3B82c7814B6D82A79e50F4AC44102B"
	SafeMultisendCallOnly141 = "0x9641d764fc13c8B624c04430C7356C1C7C8102e2"
	Multicall3Address        = "0xcA11bde05977b3631167028862bE2a173976CA11"
	USDCMainnetAddress       = "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"
	OPTokenAddress           = "0x4200000000000000000000000000000000000042"
	SuperfluidOP             = "0x1828Bff08BD244F7990edDCd9B19cc654b33cDB4"
	OptimismGovernor         = "0xcDF27F107725988f2261Ce2256bDfCdE8B382B10"
	OPGrants1                = "0x2501c477D0A35545a387Aa4A3EEe4292A9a8B3F0"
	OPGrants2                = "0x19793c7824Be70ec58BB673CA42D2779d12581BE"
)

// Known function signatures
var KnownSignatures = []string{
	"transfer(address to,uint256 amount)",
	"transferFrom(address from,address to,uint256 amount)",
	"approve(address spender,uint256 amount)",
	"increaseAllowance(address spender,uint256 amount)",
	"decreaseAllowance(address spender,uint256 amount)",
	"approveHash(bytes32 hashToApprove)",
	"aggregate3((address target,bool allowFailure,bytes callData)[] calls)",
	"multiSend(bytes transactions)",
	"callAgreement(address agreementClass, bytes callData, bytes userData)",
	"createVestingScheduleFromAmountAndDuration(address superToken, address receiver, uint256 totalAmount, uint32 totalDuration, uint32 startDate, uint32 cliffPeriod, uint32 claimPeriod)",
	"propose(address[] targets, uint256[] values, bytes[] calldatas, string description, uint8 proposalType)",
}

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
)

// ChainNames maps chain IDs to their names
var ChainNames = map[uint64]string{
	MainnetChainID:   "Ethereum",
	OPMainnetChainID: "OP Mainnet",
}

// KnownContracts maps chain IDs to a map of addresses to contract info
var KnownContracts = map[uint64]map[string]ContractInfo{
	MainnetChainID: {
		strings.ToLower(SafeMultisendAddress): {Name: "GNOSIS SAFE MULTISEND", Decimals: 0},
		strings.ToLower(Multicall3Address):    {Name: "MULTICALL3", Decimals: 0},
		strings.ToLower(USDCMainnetAddress):   {Name: "USDC", Decimals: 6},
	},
	OPMainnetChainID: {
		strings.ToLower(SafeMultisendAddress):     {Name: "GNOSIS SAFE MULTISEND CALL ONLY", Decimals: 0},
		strings.ToLower(SafeMultisendCallOnly141): {Name: "GNOSIS SAFE MULTISEND CALL ONLY", Decimals: 0},
		strings.ToLower(Multicall3Address):        {Name: "MULTICALL3", Decimals: 0},
		strings.ToLower(OPTokenAddress):           {Name: "OP TOKEN", Decimals: 18},
		strings.ToLower(SuperfluidOP):             {Name: "SUPERFLUID OP", Decimals: 18},
		strings.ToLower(OptimismGovernor):         {Name: "OPTIMISM GOVERNOR", Decimals: 0},
		strings.ToLower(OPGrants1):                {Name: "OP GRANTS 1 (3F0)", Decimals: 0},
		strings.ToLower(OPGrants2):                {Name: "OP GRANTS 2 (1BE)", Decimals: 0},
	},
}

// MulticallAddresses maps chain IDs to a set of addresses known to be multicall contracts
var MulticallAddresses = map[uint64]map[string]bool{
	MainnetChainID: {
		strings.ToLower(SafeMultisendAddress): true,
		strings.ToLower(Multicall3Address):    true,
	},
	OPMainnetChainID: {
		strings.ToLower(SafeMultisendAddress):     true,
		strings.ToLower(SafeMultisendCallOnly141): true,
		strings.ToLower(Multicall3Address):        true,
	},
}

// KnownFunctions maps function selectors to function info
var KnownFunctions = make(map[string]FunctionInfo)

// Initialize known functions
func init() {
	// Parse each signature and add to KnownFunctions
	for _, sig := range KnownSignatures {
		funcInfo, err := ParseFunctionSignature(sig)
		if err != nil {
			// Log error but continue
			fmt.Printf("Error parsing function signature %s: %v\n", sig, err)
			continue
		}

		// Add to known functions map using the selector hex
		selector := funcInfo.ABI.ID
		selectorHex := hex.EncodeToString(selector)
		KnownFunctions[selectorHex] = funcInfo
	}
}

// ParseFunctionSignature converts a function signature string to FunctionInfo
func ParseFunctionSignature(sig string) (FunctionInfo, error) {
	// Extract function name from signature
	funcName := strings.Split(sig, "(")[0]

	// Create a minimal ABI with just this function
	parsedABI, err := abi.JSON(strings.NewReader(fmt.Sprintf(`[{"name":"%s","type":"function","inputs":%s}]`,
		funcName,
		getInputsJSON(sig))))

	if err != nil {
		return FunctionInfo{}, fmt.Errorf("error parsing ABI for %s: %v", sig, err)
	}

	// Get the method from the parsed ABI
	method, exist := parsedABI.Methods[funcName]
	if !exist {
		return FunctionInfo{}, fmt.Errorf("method %s not found in parsed ABI", funcName)
	}

	return FunctionInfo{
		Name:      funcName,
		Signature: sig,
		ABI:       method,
	}, nil
}

// Helper function to generate JSON for function inputs
func getInputsJSON(signature string) string {
	// Extract the parameters part from the signature
	paramsStr := strings.TrimSuffix(strings.Split(signature, "(")[1], ")")

	// Split by comma to get individual parameters
	if paramsStr == "" {
		return "[]"
	}

	params := strings.Split(paramsStr, ",")
	inputs := make([]string, len(params))

	for i, param := range params {
		// Split the parameter into type and name
		parts := strings.Split(strings.TrimSpace(param), " ")
		paramType := parts[0]
		paramName := ""

		// If there's a name specified, use it; otherwise use arg{i}
		if len(parts) > 1 {
			paramName = parts[1]
		} else {
			paramName = fmt.Sprintf("arg%d", i)
		}

		inputs[i] = fmt.Sprintf(`{"name":"%s","type":"%s"}`, paramName, paramType)
	}

	return "[" + strings.Join(inputs, ",") + "]"
}

func GetKnownContract(address string, chainID uint64) (ContractInfo, bool) {
	normalizedAddr := strings.ToLower(address)
	if chainContracts, exists := KnownContracts[chainID]; exists {
		contractInfo, isKnown := chainContracts[normalizedAddr]
		return contractInfo, isKnown
	}
	return ContractInfo{}, false
}
