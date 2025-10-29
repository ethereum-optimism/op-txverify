package output

import (
	"fmt"
	"io"
	"math/big"
	"reflect"
	"sort"
	"strings"

	"encoding/hex"

	"github.com/ethereum-optimism/op-txverify/core"
	"github.com/ethereum/go-ethereum/common"
	"github.com/fatih/color"
)

// FormatTerminal outputs the verification result in a human-readable format to the provided writer.
// It displays transaction details, nested transactions, call data, and verification instructions
// in a color-coded terminal-friendly format.
func FormatTerminal(result *core.VerificationResult, w io.Writer) error {
	// Set up colors for consistent formatting
	heading := color.New(color.FgCyan, color.Bold).SprintFunc()
	divider := color.New(color.FgCyan).SprintFunc()
	label := color.New(color.FgMagenta).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	warning := color.New(color.FgYellow, color.Bold).SprintFunc()
	important := color.New(color.FgRed, color.Bold).SprintFunc()

	// Print basic transaction details
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, heading("TRANSACTION SUMMARY"))
	fmt.Fprintln(w, divider("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))

	// Extract transaction details
	tx := result.Transaction

	// Display verified Safe if we can
	safeContractInfo, isKnownSafeContract := core.GetKnownContract(tx.Safe, uint64(tx.Chain))
	safeDisplay := tx.Safe
	if isKnownSafeContract {
		safeDisplay = fmt.Sprintf("%s (%s 🔍)", tx.Safe, safeContractInfo.Name)
	}

	// Display verified Target if we can
	targetContractInfo, isKnownTargetContract := core.GetKnownContract(tx.To, uint64(tx.Chain))
	targetDisplay := tx.To
	if isKnownTargetContract {
		targetDisplay = fmt.Sprintf("%s (%s 🔍)", tx.To, targetContractInfo.Name)
	}

	// Parse out the operation being performed
	var operation string
	if tx.Operation == 0 {
		operation = "CALL"
	} else if tx.Operation == 1 {
		operation = "DELEGATECALL"
	} else {
		operation = "UNKNOWN OPERATION ❌"
	}

	// Parse out the value being sent
	value := core.ParseDecimals(tx.Value, 18)

	// Parse out network
	network, isKnownNetwork := core.ChainNames[uint64(tx.Chain)]
	chainDisplay := fmt.Sprintf("%d", int(tx.Chain))
	if isKnownNetwork {
		chainDisplay = fmt.Sprintf("%d (%s 🔍)", int(tx.Chain), network)
	}

	fmt.Fprintf(w, "%s: %s\n", bold("Safe"), safeDisplay)
	fmt.Fprintf(w, "%s: %s\n", bold("Chain ID"), chainDisplay)
	fmt.Fprintf(w, "%s: %s\n", bold("Target"), targetDisplay)
	fmt.Fprintf(w, "%s: %s\n", bold("ETH Value"), value)
	fmt.Fprintf(w, "%s: %d\n", bold("Nonce"), tx.Nonce)
	fmt.Fprintf(w, "%s: %s\n", bold("Operation"), operation)
	fmt.Fprintln(w, "")

	// Check if this is a nested transaction
	if result.NestedResult != nil {
		fmt.Fprintln(w, warning("⚠️  WARNING: CHILD TRANSACTION DETECTED  ⚠️"))
		fmt.Fprintln(w, divider("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))
		fmt.Fprintln(w, bold("This transaction is approving the execution of a transaction in a child Safe."))
		fmt.Fprintln(w, "")

		fmt.Fprintln(w, "")
		fmt.Fprintln(w, important("⬇️  START OF CHILD TRANSACTION DETAILS  ⬇️"))
		fmt.Fprintln(w, "")

		nestedTx := result.NestedResult.Transaction

		// Display nested Safe if we can
		nestedSafeInfo, isKnownNestedSafe := core.GetKnownContract(nestedTx.Safe, uint64(nestedTx.Chain))
		nestedSafeDisplay := nestedTx.Safe
		if isKnownNestedSafe {
			nestedSafeDisplay = fmt.Sprintf("%s (%s 🔍)", nestedTx.Safe, nestedSafeInfo.Name)
		}

		fmt.Fprintln(w, heading("CHILD TRANSACTION SUMMARY"))
		fmt.Fprintln(w, divider("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))
		fmt.Fprintf(w, "%s: %s\n", bold("Child Safe"), nestedSafeDisplay)
		fmt.Fprintf(w, "%s: %d\n", bold("Child Nonce"), nestedTx.Nonce)
		fmt.Fprintf(w, "%s: %s\n", bold("Child Hash"), result.NestedResult.ApproveHash)
		fmt.Fprintln(w, "")

		// Use the existing function to print the child call details
		printCallDetails(w, result.NestedResult.Call, 0, heading, divider, label, yellow, bold)

		// Add a divider after the child details
		fmt.Fprintln(w, important("⬆️   END OF CHILD TRANSACTION DETAILS   ⬆️"))
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "")
	}

	// Print call details (of the outer transaction in case of nested)
	printCallDetails(w, result.Call, 0, heading, divider, label, yellow, bold)

	// Print hashes
	fmt.Fprintln(w, heading("HASHES"))
	fmt.Fprintln(w, divider("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))
	fmt.Fprintf(w, "%s:  %s\n", label(bold("Domain Hash")), formatHash(result.DomainHash))
	fmt.Fprintf(w, "%s: %s\n", label(bold("Message Hash")), formatHash(result.MessageHash))
	fmt.Fprintf(w, "%s: %s\n", label(bold("Safe Tx Hash")), formatHash(result.ApproveHash))
	fmt.Fprintln(w, "")

	// Print verification instructions
	fmt.Fprintln(w, heading("VERIFICATION INSTRUCTIONS"))
	fmt.Fprintln(w, divider("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))
	fmt.Fprintf(w, "%s\n", bold("1. Transaction details should EXACTLY MATCH what you expect to see."))
	fmt.Fprintf(w, "%s\n", bold("2. Domain and message hashes should EXACTLY MATCH other machines."))
	fmt.Fprintf(w, "%s\n", bold("3. Your hardware wallet should show you the EXACT SAME HASHES."))
	fmt.Fprintf(w, "%s\n", bold("4. WHEN IN DOUBT, ASK FOR HELP."))
	fmt.Fprintln(w, "")

	return nil
}

// printCallDetails recursively prints the details of a call and any subcalls.
// Parameters:
// - w: writer to output to
// - call: the call data to print
// - depth: current recursion depth (0 for main call, increments for subcalls)
// - heading, divider, label, yellow, bold: formatting functions for consistent styling
func printCallDetails(w io.Writer, call core.CallData, depth int, heading, divider, label, yellow, bold func(a ...interface{}) string) {
	// Determine heading based on depth
	if depth == 0 {
		fmt.Fprintln(w, heading("FUNCTION CALL DETAILS"))
	} else {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "%s\n", heading(fmt.Sprintf("SUBCALL DETAILS (SUBCALL #%d)", depth)))
	}
	fmt.Fprintln(w, divider("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))

	// Print target and function name
	targetDisplay := call.Target
	if call.TargetName != "" {
		targetDisplay = fmt.Sprintf("%s (%s 🔍)", call.Target, call.TargetName)
	}
	fmt.Fprintf(w, "%s: %s\n", label("Target"), targetDisplay)
	fmt.Fprintf(w, "%s: %s\n", label("Function"), call.FunctionName)

	// If there's raw data, print it
	if call.RawData != "" {
		fmt.Fprintf(w, "%s: %s\n\n", label("Calldata"), call.RawData)
		return
	}

	// If there's parsed data, print it
	if call.ParsedData != nil {
		// Check if ParsedData is a map
		if parsedMap, ok := call.ParsedData.(map[string]interface{}); ok {
			// Get keys and sort them
			keys := make([]string, 0, len(parsedMap))
			for key := range parsedMap {
				keys = append(keys, key)
			}
			// Sort the keys alphabetically
			sort.Strings(keys)

			// Print values using sorted keys
			for _, key := range keys {
				// Skip printing array elements with index notation if we already printed the full array
				if strings.Contains(key, "[") && strings.HasSuffix(key, "]") {
					// Extract the base name before the bracket
					baseName := key[:strings.Index(key, "[")]
					if contains(keys, baseName) {
						// Skip individual elements since we'll format the array better
						continue
					}
				}

				value := parsedMap[key]
				prettyPrintValue(w, key, value, yellow, "", 0)
			}
		} else {
			// If it's not a map, just print the value
			fmt.Fprintf(w, "%v\n", formatSimpleValue(call.ParsedData))
		}
		fmt.Fprintln(w, "")
	}

	// If there are subcalls, print them recursively
	if len(call.SubCalls) > 0 {
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, heading("THIS TRANSACTION INCLUDES MULTIPLE CONTRACT INTERACTIONS"))
		fmt.Fprintln(w, divider("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))
		fmt.Fprintf(w, "%s: %d\n", bold("Number of subcalls"), len(call.SubCalls))

		// Process each subcall
		for i, subcall := range call.SubCalls {
			// Increment depth for subcalls
			printCallDetails(w, subcall, depth+i+1, heading, divider, label, yellow, bold)
		}
	}
}

// contains checks if a string slice contains a given value.
// Returns true if the value is found, false otherwise.
func contains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

// prettyPrintValue recursively formats and prints a value with proper indentation.
// Parameters:
// - w: writer to output to
// - key: name of the field/property
// - value: the value to print
// - keyColor: formatting function for the key
// - indent: current indentation string
// - depth: current recursion depth to prevent infinite recursion
func prettyPrintValue(w io.Writer, key string, value interface{}, keyColor func(a ...interface{}) string, indent string, depth int) {
	// Prevent excessive recursion
	if depth > 5 {
		fmt.Fprintf(w, "%s%s: [complex nested structure]\n", indent, keyColor(key))
		return
	}

	// Handle nil values
	if value == nil {
		fmt.Fprintf(w, "%s%s: nil\n", indent, keyColor(key))
		return
	}

	valueType := reflect.TypeOf(value)
	valueKind := valueType.Kind()

	// Simple types can be printed directly
	if valueKind != reflect.Array && valueKind != reflect.Slice &&
		valueKind != reflect.Map && valueKind != reflect.Struct {
		fmt.Fprintf(w, "%s%s: %v\n", indent, keyColor(key), formatSimpleValue(value))
		return
	}

	// For complex types, use specialized formatting functions
	if valueKind == reflect.Slice || valueKind == reflect.Array {
		prettyPrintArray(w, key, value, keyColor, indent, depth)
	} else if valueKind == reflect.Map {
		prettyPrintMap(w, key, value, keyColor, indent, depth)
	} else if valueKind == reflect.Struct {
		prettyPrintStructObj(w, key, value, keyColor, indent, depth)
	}
}

// prettyPrintArray formats and prints an array or slice with appropriate formatting.
// For byte arrays, it uses hex encoding. For small arrays of simple types, it uses
// inline formatting. For larger or complex arrays, it formats items vertically.
func prettyPrintArray(w io.Writer, key string, arr interface{}, keyColor func(a ...interface{}) string, indent string, depth int) {
	arrValue := reflect.ValueOf(arr)
	arrLen := arrValue.Len()

	// For empty arrays
	if arrLen == 0 {
		fmt.Fprintf(w, "%s%s: []\n", indent, keyColor(key))
		return
	}

	// For byte arrays/slices, print as hex
	if arrValue.Type().Elem().Kind() == reflect.Uint8 {
		// Convert to []byte
		byteArr := make([]byte, arrLen)
		for i := 0; i < arrLen; i++ {
			byteArr[i] = uint8(arrValue.Index(i).Uint())
		}
		fmt.Fprintf(w, "%s%s: 0x%s\n", indent, keyColor(key), hex.EncodeToString(byteArr))
		return
	}

	// For arrays with fewer than 5 simple elements, print inline
	if arrLen < 5 {
		allSimple := true
		for i := 0; i < arrLen; i++ {
			itemKind := arrValue.Index(i).Kind()
			if itemKind == reflect.Array || itemKind == reflect.Slice ||
				itemKind == reflect.Map || itemKind == reflect.Struct {
				allSimple = false
				break
			}
		}

		if allSimple {
			fmt.Fprintf(w, "%s%s: [", indent, keyColor(key))
			for i := 0; i < arrLen; i++ {
				if i > 0 {
					fmt.Fprint(w, ", ")
				}
				fmt.Fprintf(w, "%v", formatSimpleValue(arrValue.Index(i).Interface()))
			}
			fmt.Fprintln(w, "]")
			return
		}
	}

	// For larger or complex arrays, print items vertically
	fmt.Fprintf(w, "%s%s: [\n", indent, keyColor(key))
	for i := 0; i < arrLen; i++ {
		item := arrValue.Index(i).Interface()
		itemKind := arrValue.Index(i).Kind()

		if itemKind == reflect.Struct || itemKind == reflect.Map {
			// For complex items, recursively print them
			fmt.Fprintf(w, "%s  Item #%d:\n", indent, i)
			if itemKind == reflect.Struct {
				prettyPrintStructObj(w, "", item, keyColor, indent+"    ", depth+1)
			} else {
				prettyPrintMap(w, "", item, keyColor, indent+"    ", depth+1)
			}
		} else {
			// For simple items
			fmt.Fprintf(w, "%s  Item #%d: %v\n", indent, i, formatSimpleValue(item))
		}
	}
	fmt.Fprintf(w, "%s]\n", indent)
}

// prettyPrintMap formats and prints a map with keys sorted alphabetically.
func prettyPrintMap(w io.Writer, key string, m interface{}, keyColor func(a ...interface{}) string, indent string, depth int) {
	mapValue := reflect.ValueOf(m)

	// For empty maps
	if mapValue.Len() == 0 {
		fmt.Fprintf(w, "%s%s: {}\n", indent, keyColor(key))
		return
	}

	if key != "" {
		fmt.Fprintf(w, "%s%s: {\n", indent, keyColor(key))
	} else {
		fmt.Fprintf(w, "%s{\n", indent)
	}

	// Get and sort keys
	mapKeys := mapValue.MapKeys()
	sort.Slice(mapKeys, func(i, j int) bool {
		return fmt.Sprintf("%v", mapKeys[i]) < fmt.Sprintf("%v", mapKeys[j])
	})

	// Print each key-value pair
	for _, k := range mapKeys {
		mapKey := fmt.Sprintf("%v", k)
		mapItem := mapValue.MapIndex(k).Interface()
		prettyPrintValue(w, mapKey, mapItem, keyColor, indent+"  ", depth+1)
	}

	fmt.Fprintf(w, "%s}\n", indent)
}

// prettyPrintStructObj formats and prints a struct with its fields.
func prettyPrintStructObj(w io.Writer, key string, s interface{}, keyColor func(a ...interface{}) string, indent string, depth int) {
	structValue := reflect.ValueOf(s)
	structType := structValue.Type()

	if key != "" {
		fmt.Fprintf(w, "%s%s: {\n", indent, keyColor(key))
	} else {
		fmt.Fprintf(w, "%s{\n", indent)
	}

	// Print each field
	for i := 0; i < structValue.NumField(); i++ {
		field := structType.Field(i)
		fieldValue := structValue.Field(i).Interface()

		// Skip unexported fields
		if field.PkgPath != "" {
			continue
		}

		prettyPrintValue(w, field.Name, fieldValue, keyColor, indent+"  ", depth+1)
	}

	fmt.Fprintf(w, "%s}\n", indent)
}

// formatSimpleValue handles special formatting for various types like byte arrays,
// addresses, and other common Ethereum-specific types.
// Returns a properly formatted representation of the value.
func formatSimpleValue(value interface{}) interface{} {
	if value == nil {
		return "nil"
	}

	// Special handling for common types
	switch v := value.(type) {
	case []byte:
		return "0x" + hex.EncodeToString(v)
	case common.Address:
		return v.Hex()
	case fmt.Stringer:
		return v.String()
	case *big.Int:
		return v.String()
	default:
		return v
	}
}

// formatHash ensures the '0x' prefix is lowercase and the rest of the hash is uppercase.
func formatHash(hash string) string {
	if strings.HasPrefix(hash, "0x") {
		return "0x" + strings.ToUpper(hash[2:])
	}
	// Fallback if no "0x" prefix
	return strings.ToUpper(hash)
}
