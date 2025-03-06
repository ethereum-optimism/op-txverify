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

	// ABI type strings
	DomainSeparatorSig = "bytes32,uint256,address"
	SafeTxSig          = "bytes32,address,uint256,bytes32,uint8,uint256,uint256,uint256,address,address,uint256"

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
		// Extract function name from signature
		funcName := strings.Split(sig, "(")[0]

		// Create a minimal ABI with just this function
		parsedABI, err := abi.JSON(strings.NewReader(fmt.Sprintf(`[{"name":"%s","type":"function","inputs":%s}]`,
			funcName,
			getInputsJSON(sig))))

		if err != nil {
			// Log error but continue
			fmt.Printf("Error parsing ABI for %s: %v\n", sig, err)
			continue
		}

		// Get the method from the parsed ABI
		method, exist := parsedABI.Methods[funcName]
		if !exist {
			continue
		}

		// Calculate the function selector
		selector := method.ID
		selectorHex := hex.EncodeToString(selector)

		// Add to known functions map
		KnownFunctions[selectorHex] = FunctionInfo{
			Name:      funcName,
			Signature: sig,
			ABI:       method,
		}
	}
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
