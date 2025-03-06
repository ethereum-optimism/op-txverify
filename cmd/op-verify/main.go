package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ethereum-optimism/op-verify/core"
	"github.com/ethereum-optimism/op-verify/output"
)

func main() {
	// Parse command line flags
	txFile := flag.String("tx", "", "Path to transaction file (required)")
	outputFormat := flag.String("output", "terminal", "Output format: terminal, json")
	verbose := flag.Bool("verbose", false, "Show verbose output")

	flag.Parse()

	// Check required flags
	if *txFile == "" {
		fmt.Println("Error: transaction file is required")
		flag.Usage()
		os.Exit(1)
	}

	// Set verification options
	options := core.VerifyOptions{
		Verbose: *verbose,
	}

	// Verify the transaction
	result, err := core.VerifyTransactionFile(*txFile, options)
	if err != nil {
		fmt.Printf("Error verifying transaction: %v\n", err)
		os.Exit(1)
	}

	// Output the result in the requested format
	switch *outputFormat {
	case "json":
		output.FormatJSON(result, os.Stdout)
	case "terminal":
		output.FormatTerminal(result, os.Stdout)
	default:
		fmt.Printf("Unknown output format: %s\n", *outputFormat)
		os.Exit(1)
	}
}
