package core

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// ParseTransactionData parses the transaction data and identifies the function call
func ParseTransactionData(to string, data string, chainID uint64, options VerifyOptions) (*CallData, error) {
	// Remove 0x prefix if present
	cleanData := strings.TrimPrefix(data, "0x")

	// Normalize the target address
	normalizedTo := strings.ToLower(to)

	// Check if this is a known contract
	var contractInfo ContractInfo
	var isKnownContract bool
	targetName := ""

	if chainContracts, exists := KnownContracts[chainID]; exists {
		contractInfo, isKnownContract = chainContracts[normalizedTo]
		if isKnownContract {
			targetName = contractInfo.Name
		}
	}

	// If data is empty, return a simple transfer call
	if len(cleanData) == 0 {
		return &CallData{
			Target:       to,
			TargetName:   targetName,
			FunctionName: "transfer",
			ParsedData: map[string]interface{}{
				"value": "0",
			},
		}, nil
	}

	// Extract function selector (first 4 bytes)
	if len(cleanData) < 8 {
		return &CallData{
			Target:     to,
			TargetName: targetName,
			RawData:    data,
		}, nil
	}

	functionSelector := cleanData[:8]

	// Try to identify the function from known selectors
	functionInfo, isKnownFunction := KnownFunctions[functionSelector]

	if !isKnownFunction {
		// If we can't identify the function, return the raw data
		return &CallData{
			Target:       to,
			TargetName:   targetName,
			FunctionName: "unknown",
			RawData:      data,
		}, nil
	}

	// Parse the function arguments
	parsedArgs, err := parseArguments(functionInfo.ABI, "0x"+cleanData)
	if err != nil {
		// If we can't parse the arguments, return the raw data
		return &CallData{
			Target:       to,
			TargetName:   targetName,
			FunctionName: functionInfo.Name,
			RawData:      data,
		}, nil
	}

	// Check if this is a multicall contract and the function is a multicall function
	isMulticallContract := false
	if chainMulticalls, exists := MulticallAddresses[chainID]; exists {
		isMulticallContract = chainMulticalls[normalizedTo]
	}

	isMulticallFunction := functionInfo.Name == "multiSend" || functionInfo.Name == "aggregate3"

	if isMulticallContract && isMulticallFunction {
		// Parse subcalls - passing contract address, chain ID, and full function info
		subcalls, err := parseMulticall(normalizedTo, chainID, functionInfo, parsedArgs, options)
		if err != nil {
			return nil, err
		}

		// Return with subcalls
		return &CallData{
			Target:       to,
			TargetName:   targetName,
			FunctionName: functionInfo.Name,
			SubCalls:     subcalls,
		}, nil
	}

	// Regular function call
	return &CallData{
		Target:       to,
		TargetName:   targetName,
		FunctionName: functionInfo.Name,
		ParsedData:   parsedArgs,
	}, nil
}

// parseArguments decodes the function arguments from calldata
func parseArguments(method abi.Method, calldata string) (map[string]interface{}, error) {
	// Remove 0x prefix if present
	calldata = strings.TrimPrefix(calldata, "0x")

	// Convert hex string to bytes
	data, err := hex.DecodeString(calldata)
	if err != nil {
		return nil, err
	}

	// Ensure data is long enough to contain the method ID
	if len(data) < 4 {
		return nil, errors.New("calldata too short")
	}

	// Skip the method ID (first 4 bytes)
	inputData := data[4:]

	// Unpack the arguments
	args, err := method.Inputs.Unpack(inputData)
	if err != nil {
		return nil, err
	}

	// Convert to a map
	result := make(map[string]interface{})
	for i, arg := range args {
		name := method.Inputs[i].Name
		if name == "" {
			name = fmt.Sprintf("arg%d", i)
		}

		// Convert byte arrays to hex strings for better readability
		if byteArray, ok := arg.([]byte); ok {
			result[name] = "0x" + hex.EncodeToString(byteArray)
		} else {
			result[name] = arg
		}
	}

	return result, nil
}

// parseMulticall parses subcalls from a multicall function
func parseMulticall(contractAddress string, chainID uint64, functionInfo FunctionInfo, args map[string]interface{}, options VerifyOptions) ([]CallData, error) {
	var subcalls []CallData

	// First determine which contract we're dealing with
	normalizedAddress := strings.ToLower(contractAddress)

	// Handle Safe Multisend contracts
	if normalizedAddress == strings.ToLower(SafeMultisendAddress) || normalizedAddress == strings.ToLower(SafeMultisendCallOnly141) {
		// For Safe Multisend contract, check the function signature
		if functionInfo.Signature == SafeMultisendSig {
			// Parse multiSend calldata
			hexData, ok := args["transactions"].(string)
			if !ok || !strings.HasPrefix(hexData, "0x") {
				return nil, errors.New("invalid multiSend data format")
			}

			// Convert hex string to bytes
			data, err := hex.DecodeString(strings.TrimPrefix(hexData, "0x"))
			if err != nil {
				return nil, fmt.Errorf("invalid multiSend data: %v", err)
			}

			// multiSend data format: operation (1 byte) + to (20 bytes) + value (32 bytes) + dataLength (32 bytes) + data (variable)
			pos := 0
			for pos < len(data) {
				// Ensure we have enough data for the fixed-size fields
				if pos+85 > len(data) {
					break
				}

				// Extract operation
				// Can skip extracting the operation as it's always 0
				pos++

				// Extract to address
				to := common.BytesToAddress(data[pos : pos+20])
				pos += 20

				// Skip extracting value for now, can implement later if needed
				pos += 32

				// Extract data length
				dataLength := common.BytesToHash(data[pos : pos+32])
				pos += 32

				// Convert data length to int
				length := new(big.Int).SetBytes(dataLength[:]).Uint64()

				// Ensure we have enough data for the variable-size field
				if pos+int(length) > len(data) {
					break
				}

				// Extract call data
				callData := data[pos : pos+int(length)]
				pos += int(length)

				// Parse the subcall
				subcall, err := ParseTransactionData(to.Hex(), "0x"+hex.EncodeToString(callData), chainID, options)
				if err != nil {
					return nil, err
				}

				subcalls = append(subcalls, *subcall)
			}
		} else {
			return nil, fmt.Errorf("unsupported function %s for Safe Multisend contract", functionInfo.Signature)
		}
	} else if normalizedAddress == strings.ToLower(Multicall3Address) {
		// For Multicall3 contract, check the function signature
		if functionInfo.Signature == Aggregate3Sig {
			// Parse aggregate3 calldata
			calls, ok := args["calls"].([]struct {
				Target       common.Address
				AllowFailure bool
				CallData     string
			})
			if !ok {
				return nil, errors.New("invalid aggregate3 data format")
			}

			for _, call := range calls {
				subcall, err := ParseTransactionData(call.Target.Hex(), call.CallData, chainID, options)
				if err != nil {
					continue
				}

				subcalls = append(subcalls, *subcall)
			}
		} else {
			return nil, fmt.Errorf("unsupported function %s for Multicall3 contract", functionInfo.Signature)
		}
	} else {
		// No generic parsing - return an error for unknown contracts
		return nil, fmt.Errorf("unsupported multicall contract: %s", contractAddress)
	}

	return subcalls, nil
}
