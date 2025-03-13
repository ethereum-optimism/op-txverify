package output

import (
	"fmt"
	"io"
	"math/big"
	"sort"

	"github.com/ethereum-optimism/op-verify/core"
	"github.com/fatih/color"
)

// FormatTerminal outputs the verification result in a human-readable format
func FormatTerminal(result *core.VerificationResult, w io.Writer) error {
	// Set up colors
	heading := color.New(color.FgCyan, color.Bold).SprintFunc()
	divider := color.New(color.FgCyan).SprintFunc()
	label := color.New(color.FgMagenta).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	// Print basic transaction details
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, heading("BASIC TRANSACTION DETAILS"))
	fmt.Fprintln(w, divider("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))

	// Extract transaction details
	tx := result.Transaction

	// Display verified Safe if we can
	safeContractInfo, isKnownSafeContract := core.GetKnownContract(tx.Safe, uint64(tx.Chain))
	safeDisplay := tx.Safe
	if isKnownSafeContract {
		safeDisplay = fmt.Sprintf("%s (%s ✅)", tx.Safe, safeContractInfo.Name)
	}

	// Display verified Target if we can
	targetContractInfo, isKnownTargetContract := core.GetKnownContract(tx.To, uint64(tx.Chain))
	targetDisplay := tx.To
	if isKnownTargetContract {
		targetDisplay = fmt.Sprintf("%s (%s ✅)", tx.To, targetContractInfo.Name)
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
	value := core.ParseDecimals(big.NewInt(int64(tx.Value)), 18)

	// Parse out network
	network, isKnownNetwork := core.ChainNames[uint64(tx.Chain)]
	chainDisplay := fmt.Sprintf("%d", int(tx.Chain))
	if isKnownNetwork {
		chainDisplay = fmt.Sprintf("%d (%s ✅)", int(tx.Chain), network)
	}

	fmt.Fprintf(w, "%s: %s\n", bold("Safe"), safeDisplay)
	fmt.Fprintf(w, "%s: %s\n", bold("Chain ID"), chainDisplay)
	fmt.Fprintf(w, "%s: %s\n", bold("Target"), targetDisplay)
	fmt.Fprintf(w, "%s: %s\n", bold("ETH Value"), value)
	fmt.Fprintf(w, "%s: %d\n", bold("Nonce"), tx.Nonce)
	fmt.Fprintf(w, "%s: %s\n", bold("Operation"), operation)
	fmt.Fprintln(w, "")

	// Print call details
	printCallDetails(w, result.Call, 0, heading, divider, label, yellow, bold)

	// Print hashes
	fmt.Fprintln(w, heading("HASHES"))
	fmt.Fprintln(w, divider("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))
	fmt.Fprintf(w, "%s: %s\n", label(bold("Domain Hash")), result.DomainHash)
	fmt.Fprintf(w, "%s: %s\n", label(bold("Message Hash")), result.MessageHash)
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

// Helper function to print call details recursively
func printCallDetails(w io.Writer, call core.CallData, depth int, heading, divider, label, yellow, bold func(a ...interface{}) string) {
	// Determine heading based on depth
	if depth == 0 {
		fmt.Fprintln(w, heading("TRANSACTION DETAILS"))
	} else {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "%s\n", heading(fmt.Sprintf("SUBCALL DETAILS (SUBCALL #%d)", depth)))
	}
	fmt.Fprintln(w, divider("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))

	// Print target and function name
	targetDisplay := call.Target
	if call.TargetName != "" {
		targetDisplay = fmt.Sprintf("%s (%s ✅)", call.Target, call.TargetName)
	}
	fmt.Fprintf(w, "%s: %s\n", label("Target"), targetDisplay)
	fmt.Fprintf(w, "%s: %s\n\n", label("Function"), call.FunctionName)

	// If there's raw data, print it
	if call.RawData != "" {
		fmt.Fprintln(w, label(bold("RAW DATA:")))
		fmt.Fprintln(w, label("────────────────────────────────────────────────────────"))
		fmt.Fprintln(w, call.RawData)
		fmt.Fprintln(w, "")
		return
	}

	// If there's parsed data, print it
	if call.ParsedData != nil {
		// Make the Parsed Data heading purple (using the label color) but still bold
		fmt.Fprintln(w, label(bold("PARSED DATA")))
		fmt.Fprintln(w, label("────────────────────────────────────────────────────────"))

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
				fmt.Fprintf(w, "%s: %v\n", yellow(key), parsedMap[key])
			}
		} else {
			// If it's not a map, just print the value
			fmt.Fprintf(w, "%v\n", call.ParsedData)
		}
		fmt.Fprintln(w, "")
	}

	// If there are subcalls, print them recursively
	if len(call.SubCalls) > 0 {
		fmt.Fprintln(w, heading("THIS TRANSACTION INCLUDES NESTED SUBCALLS"))
		fmt.Fprintln(w, divider("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))
		fmt.Fprintf(w, "%s: %d\n\n", bold("Number of subcalls"), len(call.SubCalls))

		// Process each subcall
		for i, subcall := range call.SubCalls {
			// Increment depth for subcalls
			printCallDetails(w, subcall, depth+i+1, heading, divider, label, yellow, bold)
		}
	}
}
