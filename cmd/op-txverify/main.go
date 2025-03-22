package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ethereum-optimism/op-txverify/core"
	"github.com/ethereum-optimism/op-txverify/output"
	cli "github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "op-txverify",
		Usage: "Verify and generate Optimism transactions",
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
						Name:     "network",
						Aliases:  []string{"n"},
						Usage:    "Network name: ethereum, op, base (required)",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "safe",
						Aliases:  []string{"a"},
						Usage:    "Safe address (required)",
						Required: true,
					},
					&cli.Uint64Flag{
						Name:     "nonce",
						Usage:    "Transaction nonce (required)",
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
				Action: onlineAction,
			},
			{
				Name:  "download",
				Usage: "Generate a transaction JSON file",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "network",
						Aliases:  []string{"n"},
						Usage:    "Network name: ethereum, op, base (required)",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "safe",
						Aliases:  []string{"a"},
						Usage:    "Safe address (required)",
						Required: true,
					},
					&cli.Uint64Flag{
						Name:     "nonce",
						Usage:    "Transaction nonce (required)",
						Required: true,
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

	// Set verification options
	options := core.VerifyOptions{
		Verbose: verbose,
	}

	// Verify the transaction
	result, err := core.VerifyTransaction(tx, options)
	if err != nil {
		return fmt.Errorf("error verifying transaction: %w", err)
	}

	// Output the result in the requested format
	switch outputFormat {
	case "json":
		output.FormatJSON(result, os.Stdout)
	case "terminal":
		output.FormatTerminal(result, os.Stdout)
	default:
		return fmt.Errorf("unknown output format: %s", outputFormat)
	}

	return nil
}

func onlineAction(c *cli.Context) error {
	network := c.String("network")
	address := c.String("safe")
	nonce := c.Uint64("nonce")
	outputFormat := c.String("output")
	verbose := c.Bool("verbose")

	// Validate network
	if network != "ethereum" && network != "op" && network != "base" {
		return fmt.Errorf("invalid network: %s (must be ethereum, op, or base)", network)
	}

	// Strip the chain prefix if present
	address = core.StripChainPrefix(address)

	// Generate the transaction
	tx, err := core.GenerateTransaction(network, address, nonce)
	if err != nil {
		return err
	}

	// Set verification options
	options := core.VerifyOptions{
		Verbose: verbose,
	}

	// Verify the generated transaction
	result, err := core.VerifyTransaction(*tx, options)
	if err != nil {
		return fmt.Errorf("error verifying transaction: %w", err)
	}

	// Output the result in the requested format
	switch outputFormat {
	case "json":
		output.FormatJSON(result, os.Stdout)
	case "terminal":
		output.FormatTerminal(result, os.Stdout)
	default:
		return fmt.Errorf("unknown output format: %s", outputFormat)
	}

	return nil
}

func downloadAction(c *cli.Context) error {
	network := c.String("network")
	address := c.String("safe")
	nonce := c.Uint64("nonce")
	outputFile := c.String("output")

	// Validate network
	if network != "ethereum" && network != "op" && network != "base" {
		return fmt.Errorf("invalid network: %s (must be ethereum, op, or base)", network)
	}

	// Generate the transaction JSON
	tx, err := core.GenerateTransaction(network, address, nonce)
	if err != nil {
		return fmt.Errorf("error generating transaction: %w", err)
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

	// Set verification options
	options := core.VerifyOptions{
		Verbose: verbose,
	}

	// Verify the transaction
	result, err := core.VerifyTransaction(tx, options)
	if err != nil {
		return fmt.Errorf("error verifying transaction: %w", err)
	}

	// Output the result in the requested format
	switch outputFormat {
	case "json":
		output.FormatJSON(result, os.Stdout)
	case "terminal":
		output.FormatTerminal(result, os.Stdout)
	default:
		return fmt.Errorf("unknown output format: %s", outputFormat)
	}

	return nil
}
