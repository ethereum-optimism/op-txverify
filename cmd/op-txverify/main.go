package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/ethereum-optimism/op-txverify/core"
	"github.com/ethereum-optimism/op-txverify/output"
	cli "github.com/urfave/cli/v2"
)

// Version can be set at build time with -ldflags "-X main.Version=x.y.z"
var Version = "dev"

// Commit can be set at build time with -ldflags "-X main.Commit=abc123"
var Commit = "unknown"

func main() {
	// Build version string with commit info if available
	version := Version
	if Commit != "unknown" && Commit != "" {
		commitDisplay := Commit
		if len(Commit) > 8 {
			commitDisplay = Commit[:8] // Show first 8 chars of commit
		}
		version = fmt.Sprintf("%s (commit: %s)", Version, commitDisplay)
	}

	app := &cli.App{
		Name:    "op-txverify",
		Usage:   "Verify and generate Optimism transactions",
		Version: version,
		Commands: []*cli.Command{
			{
				Name:  "offline",
				Usage: "Verify a transaction file",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "tx",
						Aliases:  []string{"t"},
						Usage:    "Path to transaction file (required)",
						Required: true,
					},
					&cli.StringFlag{
						Name:    "output",
						Aliases: []string{"o"},
						Usage:   "Output format: terminal, json",
						Value:   "terminal",
					},
					&cli.BoolFlag{
						Name:    "verbose",
						Aliases: []string{"v"},
						Usage:   "Show verbose output",
					},
				},
				Action: offlineAction,
			},
			{
				Name:  "online",
				Usage: "Generate and verify a transaction in one step",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "tx",
						Aliases: []string{"t"},
						Usage:   "Safe transaction hash or link (alternative to --network/--safe/--nonce)",
					},
					&cli.StringFlag{
						Name:    "network",
						Aliases: []string{"n"},
						Usage:   "Network name: ethereum, op, base (required unless --tx is used)",
					},
					&cli.StringFlag{
						Name:    "safe",
						Aliases: []string{"a"},
						Usage:   "Safe address (required unless --tx is used)",
					},
					&cli.Uint64Flag{
						Name:  "nonce",
						Usage: "Transaction nonce (required unless --tx is used)",
					},
					&cli.StringFlag{
						Name:    "output",
						Aliases: []string{"o"},
						Usage:   "Output format: terminal, json",
						Value:   "terminal",
					},
					&cli.BoolFlag{
						Name:    "verbose",
						Aliases: []string{"v"},
						Usage:   "Show verbose output",
					},
				},
				Action: onlineAction,
			},
			{
				Name:  "download",
				Usage: "Generate a transaction JSON file",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "tx",
						Aliases: []string{"t"},
						Usage:   "Safe transaction hash or link (alternative to --network/--safe/--nonce)",
					},
					&cli.StringFlag{
						Name:    "network",
						Aliases: []string{"n"},
						Usage:   "Network name: ethereum, op, base (required unless --tx is used)",
					},
					&cli.StringFlag{
						Name:    "safe",
						Aliases: []string{"a"},
						Usage:   "Safe address (required unless --tx is used)",
					},
					&cli.Uint64Flag{
						Name:  "nonce",
						Usage: "Transaction nonce (required unless --tx is used)",
					},
					&cli.StringFlag{
						Name:    "output",
						Aliases: []string{"o"},
						Usage:   "Output file path (defaults to stdout if not specified)",
					},
				},
				Action: downloadAction,
			},
			{
				Name:  "qr",
				Usage: "Scan a transaction QR code using your camera",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "device",
						Aliases: []string{"d"},
						Usage:   "Camera device to use (defaults to system default)",
						Value:   "",
					},
					&cli.StringFlag{
						Name:    "output",
						Aliases: []string{"o"},
						Usage:   "Output format: terminal, json",
						Value:   "terminal",
					},
					&cli.BoolFlag{
						Name:    "verbose",
						Aliases: []string{"v"},
						Usage:   "Show verbose output",
					},
				},
				Action: qrAction,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func offlineAction(c *cli.Context) error {
	txFile := c.String("tx")
	outputFormat := c.String("output")
	verbose := c.Bool("verbose")

	// Read the transaction file
	data, err := os.ReadFile(txFile)
	if err != nil {
		return fmt.Errorf("failed to read transaction file: %w", err)
	}

	// Parse the transaction
	var tx core.SafeTransaction
	if err := json.Unmarshal(data, &tx); err != nil {
		return fmt.Errorf("failed to parse transaction: %w", err)
	}

	return verifyTransactionAndFormat(tx, verbose, outputFormat, os.Stdout)
}

func onlineAction(c *cli.Context) error {
	txInput := c.String("tx")
	network := c.String("network")
	address := c.String("safe")
	nonce := c.Uint64("nonce")
	outputFormat := c.String("output")
	verbose := c.Bool("verbose")

	tx, metadata, err := getTxFromFlags(txInput, network, address, nonce, c.IsSet("nonce"))
	if err != nil {
		return err
	}

	if verbose && metadata != nil {
		fmt.Fprintf(os.Stderr, "Found transaction on %s (chain %d)\n", metadata.Network, metadata.ChainID)
		fmt.Fprintf(os.Stderr, "Safe: %s\n", metadata.SafeAddress)
		fmt.Fprintf(os.Stderr, "Nonce: %d\n", metadata.Nonce)
	}

	return verifyTransactionAndFormat(*tx, verbose, outputFormat, os.Stdout)
}

func downloadAction(c *cli.Context) error {
	txInput := c.String("tx")
	network := c.String("network")
	address := c.String("safe")
	nonce := c.Uint64("nonce")
	outputFile := c.String("output")

	tx, _, err := getTxFromFlags(txInput, network, address, nonce, c.IsSet("nonce"))
	if err != nil {
		return err
	}

	// Output the transaction JSON
	if outputFile != "" {
		file, err := os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("error creating output file: %w", err)
		}
		defer file.Close()
		return output.FormatJSON(tx, file)
	}

	// Output to stdout if no file specified
	return output.FormatJSON(tx, os.Stdout)
}

func qrAction(c *cli.Context) error {
	deviceID := c.String("device")
	outputFormat := c.String("output")
	verbose := c.Bool("verbose")

	// Scan QR code from camera
	data, err := core.ScanQRCode(deviceID)
	if err != nil {
		return fmt.Errorf("failed to scan QR code: %w", err)
	}

	// Parse the transaction
	var tx core.SafeTransaction
	if err := json.Unmarshal([]byte(data), &tx); err != nil {
		return fmt.Errorf("failed to parse transaction from QR code: %w", err)
	}

	return verifyTransactionAndFormat(tx, verbose, outputFormat, os.Stdout)
}

// getTxFromFlags resolves a SafeTransaction from either a tx hash/link or network/safe/nonce flags.
func getTxFromFlags(txInput, network, address string, nonce uint64, nonceProvided bool) (*core.SafeTransaction, *core.TransactionMetadata, error) {
	if txInput != "" {
		// Extract transaction hash from URL or use directly
		txHash, err := core.ExtractTransactionHash(txInput)
		if err != nil {
			return nil, nil, fmt.Errorf("error parsing transaction input: %w", err)
		}

		// Fetch transaction data from API
		metadata, err := core.FetchTransactionByHash(txHash)
		if err != nil {
			return nil, nil, fmt.Errorf("error fetching transaction: %w", err)
		}

		return metadata.Transaction, metadata, nil
	}

	// Validate required parameters
	if network == "" || address == "" || (nonce == 0 && !nonceProvided) {
		return nil, nil, fmt.Errorf("either --tx or all of (--network, --safe, --nonce) must be provided")
	}

	// Validate network (keep aligned with CLI help text)
	if network != "ethereum" && network != "op" && network != "base" {
		return nil, nil, fmt.Errorf("invalid network: %s (must be ethereum, op, or base)", network)
	}

	// Strip the chain prefix if present
	address = core.StripChainPrefix(address)

	// Generate the transaction
	tx, err := core.GenerateTransaction(network, address, nonce)
	if err != nil {
		return nil, nil, err
	}
	return tx, nil, nil
}

// verifyTransactionAndFormat verifies a transaction and writes it using the requested formatter.
func verifyTransactionAndFormat(tx core.SafeTransaction, verbose bool, outputFormat string, w io.Writer) error {
	options := core.VerifyOptions{Verbose: verbose}
	result, err := core.VerifyTransaction(tx, options)
	if err != nil {
		return fmt.Errorf("error verifying transaction: %w", err)
	}
	return writeVerificationOutput(result, outputFormat, w)
}

// writeVerificationOutput writes a VerificationResult to the writer using the selected output format.
func writeVerificationOutput(result *core.VerificationResult, outputFormat string, w io.Writer) error {
	switch outputFormat {
	case "json":
		return output.FormatJSON(result, w)
	case "terminal":
		return output.FormatTerminal(result, w)
	default:
		return fmt.Errorf("unknown output format: %s", outputFormat)
	}
}
