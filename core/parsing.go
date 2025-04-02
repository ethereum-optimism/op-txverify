package core

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"reflect"
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
	contractInfo, isKnownContract := GetKnownContract(normalizedTo, chainID)
	targetName := ""
	if isKnownContract {
		targetName = contractInfo.Name
	}

	// If data is empty, return a simple transfer call
	if len(cleanData) == 0 {
		return &CallData{
			Target:       to,
			TargetName:   targetName,
			FunctionName: "unknown",
			RawData:      "0x" + data,
		}, nil
	}

	// Extract function selector (first 4 bytes)
	if len(cleanData) < 8 {
		return &CallData{
			Target:       to,
			TargetName:   targetName,
			FunctionName: "unknown",
			RawData:      data,
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

	// If this is a token function on a known contract where we have decimals, adjust the amount
	if isKnownContract && TokenFunctions[functionInfo.Name] && contractInfo.Decimals > 0 && parsedArgs["amount"] != nil {
		parsedArgs["amount"] = ParseDecimals(parsedArgs["amount"].(*big.Int), contractInfo.Decimals)
	}

	// Check if any of the arguments are known contracts
	for key, value := range parsedArgs {
		if value, ok := value.(common.Address); ok {
			contractInfo, isKnownContract := GetKnownContract(value.Hex(), chainID)
			if isKnownContract {
				parsedArgs[key] = fmt.Sprintf("%s (%s 🔍)", value, contractInfo.Name)
			}
		}
	}

	// Check if this is a multicall contract and the function is a multicall function
	isMulticallContract := false
	if chainMulticalls, exists := MulticallAddresses[chainID]; exists {
		isMulticallContract = chainMulticalls[normalizedTo]
	}

	isMulticallFunction := functionInfo.Name == "multiSend" || functionInfo.Name == "aggregate3" || functionInfo.Name == "aggregate3Value"

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

// ParseDecimals parses the amount and returns it as a human-readable string
// with the correct number of decimals and comma grouping in the integer portion.
// If the decimal portion is all zeros, it will show only 2 decimal places.
func ParseDecimals(amount *big.Int, decimals int) string {
	amountStr := amount.String()

	// Pad with leading zeros for decimal portion if needed
	if len(amountStr) <= decimals {
		amountStr = strings.Repeat("0", decimals-len(amountStr)+1) + amountStr
	}
	decimalPos := len(amountStr) - decimals

	// Split integer and fractional parts
	intPart := amountStr[:decimalPos]
	fracPart := ""
	if decimals > 0 {
		fracPart = amountStr[decimalPos:]
	}

	// Insert commas into integer part
	intPartWithCommas := addCommas(intPart)

	// If we have a fractional part
	if decimals > 0 {
		// Check if all digits in fracPart are zeros
		allZeros := true
		for _, c := range fracPart {
			if c != '0' {
				allZeros = false
				break
			}
		}

		if allZeros {
			// If all zeros, just show 2 decimal places
			if len(fracPart) >= 2 {
				return intPartWithCommas + "." + fracPart[:2]
			}
			return intPartWithCommas + "." + fracPart + strings.Repeat("0", 2-len(fracPart))
		} else {
			// Trim trailing zeros
			for len(fracPart) > 0 && fracPart[len(fracPart)-1] == '0' {
				fracPart = fracPart[:len(fracPart)-1]
			}
			if len(fracPart) > 0 {
				return intPartWithCommas + "." + fracPart
			}
		}
	}

	return intPartWithCommas
}

// addCommas inserts commas every three digits starting from the right.
func addCommas(s string) string {
	// Special case for empty or very short string
	if len(s) <= 3 {
		return s
	}

	var result []byte
	count := 0

	// Append digits in reverse order, inserting commas every 3 digits
	for i := len(s) - 1; i >= 0; i-- {
		result = append(result, s[i])
		count++
		if count%3 == 0 && i != 0 {
			result = append(result, ',')
		}
	}

	// Reverse the result back to normal order
	for i := 0; i < len(result)/2; i++ {
		result[i], result[len(result)-1-i] = result[len(result)-1-i], result[i]
	}

	return string(result)
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

		// Handle arrays by creating indexed entries in the map
		if reflect.TypeOf(arg).Kind() == reflect.Slice &&
			reflect.TypeOf(arg).Elem().Kind() != reflect.Uint8 {
			// It's an array but not a byte array
			arrayValue := reflect.ValueOf(arg)

			// Add the full array under its original name
			result[name] = arg

			// Also add individual elements with indexed names for easier access
			for j := 0; j < arrayValue.Len(); j++ {
				indexedName := fmt.Sprintf("%s[%d]", name, j)
				element := arrayValue.Index(j).Interface()

				// Apply the same formatting we do for regular elements
				if byteArray, ok := element.([]byte); ok {
					result[indexedName] = "0x" + hex.EncodeToString(byteArray)
				} else {
					result[indexedName] = element
				}
			}
		} else {
			// Convert byte arrays and fixed-size uint8 arrays to hex strings for better readability
			if byteArray, ok := arg.([]byte); ok {
				result[name] = "0x" + hex.EncodeToString(byteArray)
			} else {
				// Check for fixed-size uint8 arrays using type string
				argType := fmt.Sprintf("%T", arg)
				if strings.HasPrefix(argType, "[") && strings.HasSuffix(argType, "]uint8") {
					// Convert the fixed-size array to a byte slice
					bytes := make([]byte, 0)

					// Get string representation without brackets
					str := fmt.Sprintf("%v", arg)
					str = strings.Trim(str, "[]")

					// Split by space (Go's array string representation uses spaces)
					for _, numStr := range strings.Fields(str) {
						var val uint8
						fmt.Sscanf(numStr, "%d", &val)
						bytes = append(bytes, val)
					}

					result[name] = "0x" + hex.EncodeToString(bytes)
				} else {
					result[name] = arg
				}
			}
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
	} else if normalizedAddress == strings.ToLower(Multicall3Address) || normalizedAddress == strings.ToLower(Multicall3Delegatecall) {
		// For Multicall3 contract, check the function signature
		if functionInfo.Signature == Aggregate3Sig {
			// Define the struct type for the calls
			type Call3 struct {
				Target       common.Address
				AllowFailure bool
				CallData     []byte
			}

			// Get and validate the raw calls data
			rawCalls, ok := args["calls"]
			if !ok {
				return nil, errors.New("missing calls in aggregate3 data")
			}

			// Use reflection to convert the data to our expected type
			rawCallsValue := reflect.ValueOf(rawCalls)
			if rawCallsValue.Kind() != reflect.Slice {
				return nil, errors.New("aggregate3 calls must be a slice")
			}

			// Create the result slice with the correct capacity
			calls := make([]Call3, rawCallsValue.Len())

			// Process each call in the slice
			for i := 0; i < rawCallsValue.Len(); i++ {
				callValue := rawCallsValue.Index(i)

				// Extract the required fields using case-insensitive matching
				targetField := callValue.FieldByNameFunc(func(name string) bool {
					return strings.EqualFold(name, "target")
				})
				if !targetField.IsValid() {
					return nil, errors.New("missing target field in call")
				}

				allowFailureField := callValue.FieldByNameFunc(func(name string) bool {
					return strings.EqualFold(name, "allowfailure")
				})
				if !allowFailureField.IsValid() {
					return nil, errors.New("missing allowFailure field in call")
				}

				callDataField := callValue.FieldByNameFunc(func(name string) bool {
					return strings.EqualFold(name, "calldata")
				})
				if !callDataField.IsValid() {
					return nil, errors.New("missing callData field in call")
				}

				// Populate the result struct
				calls[i] = Call3{
					Target:       targetField.Interface().(common.Address),
					AllowFailure: allowFailureField.Interface().(bool),
					CallData:     callDataField.Interface().([]byte),
				}
			}

			// Parse aggregate3 calldata
			for _, call := range calls {
				subcall, err := ParseTransactionData(call.Target.Hex(), "0x"+hex.EncodeToString(call.CallData), chainID, options)
				if err != nil {
					continue
				}

				subcalls = append(subcalls, *subcall)
			}
		} else if functionInfo.Signature == Aggregate3ValueSig {
			// Define the struct type for the calls
			type Call3 struct {
				Target       common.Address
				AllowFailure bool
				Value        *big.Int
				CallData     []byte
			}

			// Get and validate the raw calls data
			rawCalls, ok := args["calls"]
			if !ok {
				return nil, errors.New("missing calls in aggregate3 data")
			}

			// Use reflection to convert the data to our expected type
			rawCallsValue := reflect.ValueOf(rawCalls)
			if rawCallsValue.Kind() != reflect.Slice {
				return nil, errors.New("aggregate3 calls must be a slice")
			}

			// Create the result slice with the correct capacity
			calls := make([]Call3, rawCallsValue.Len())

			// Process each call in the slice
			for i := 0; i < rawCallsValue.Len(); i++ {
				callValue := rawCallsValue.Index(i)

				// Extract the required fields using case-insensitive matching
				targetField := callValue.FieldByNameFunc(func(name string) bool {
					return strings.EqualFold(name, "target")
				})
				if !targetField.IsValid() {
					return nil, errors.New("missing target field in call")
				}

				allowFailureField := callValue.FieldByNameFunc(func(name string) bool {
					return strings.EqualFold(name, "allowfailure")
				})
				if !allowFailureField.IsValid() {
					return nil, errors.New("missing allowFailure field in call")
				}

				valueField := callValue.FieldByNameFunc(func(name string) bool {
					return strings.EqualFold(name, "value")
				})
				if !valueField.IsValid() {
					return nil, errors.New("missing value field in call")
				}

				callDataField := callValue.FieldByNameFunc(func(name string) bool {
					return strings.EqualFold(name, "calldata")
				})
				if !callDataField.IsValid() {
					return nil, errors.New("missing callData field in call")
				}

				// Populate the result struct
				calls[i] = Call3{
					Target:       targetField.Interface().(common.Address),
					AllowFailure: allowFailureField.Interface().(bool),
					Value:        valueField.Interface().(*big.Int),
					CallData:     callDataField.Interface().([]byte),
				}
			}

			// Parse aggregate3 calldata
			for _, call := range calls {
				subcall, err := ParseTransactionData(call.Target.Hex(), "0x"+hex.EncodeToString(call.CallData), chainID, options)
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
