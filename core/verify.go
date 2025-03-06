package core

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// VerificationResult represents the complete output of the verification process
type VerificationResult struct {
	Transaction SafeTransaction `json:"transaction"`
	DomainHash  string          `json:"domainHash"`
	MessageHash string          `json:"messageHash"`
	Call        CallData        `json:"call"`
}

// SafeTransaction represents a Gnosis Safe transaction
type SafeTransaction struct {
	Safe           string   `json:"safe"`
	Chain          int      `json:"chain"`
	To             string   `json:"to"`
	Value          string   `json:"value"`
	Data           string   `json:"data"`
	Operation      int      `json:"operation"`
	SafeTxGas      int      `json:"safe_tx_gas"`
	BaseGas        int      `json:"base_gas"`
	GasPrice       int      `json:"gas_price"`
	GasToken       string   `json:"gas_token"`
	RefundReceiver string   `json:"refund_receiver"`
	Nonce          int      `json:"nonce"`
	Call           CallData `json:"call"`
}

// CallData represents a function call with parsed arguments
type CallData struct {
	Target       string      `json:"target"`
	TargetName   string      `json:"targetName,omitempty"`
	FunctionName string      `json:"functionName"`
	FunctionData string      `json:"functionData,omitempty"`
	RawData      string      `json:"rawData,omitempty"`
	ParsedData   interface{} `json:"parsedData,omitempty"`
	SubCalls     []CallData  `json:"subCalls,omitempty"`
}

// VerifyOptions contains configuration options for verification
type VerifyOptions struct {
	Verbose bool
}

// VerifyTransactionFile verifies a Safe transaction from a file path
func VerifyTransactionFile(filePath string, options VerifyOptions) (*VerificationResult, error) {
	// Read the transaction file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read transaction file: %w", err)
	}

	return VerifyTransaction(data, options)
}

// VerifyTransaction verifies a Safe transaction from JSON data
func VerifyTransaction(txData []byte, options VerifyOptions) (*VerificationResult, error) {
	// Parse the transaction
	var tx SafeTransaction
	if err := json.Unmarshal(txData, &tx); err != nil {
		return nil, fmt.Errorf("failed to parse transaction: %w", err)
	}

	// Strip chain prefix from the target address (e.g., "oeth:", "eth:")
	tx.To = stripChainPrefix(tx.To)
	tx.Safe = stripChainPrefix(tx.Safe)

	// Parse the transaction data
	call, err := ParseTransactionData(tx.To, tx.Data, uint64(tx.Chain), options)
	if err != nil {
		return nil, fmt.Errorf("failed to parse transaction data: %w", err)
	}
	tx.Call = *call

	// Calculate the domain and message hashes
	domainHash, err := CalculateDomainHash(tx)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate domain hash: %w", err)
	}

	messageHash, err := CalculateMessageHash(tx)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate message hash: %w", err)
	}

	// Create the verification result
	result := &VerificationResult{
		Transaction: tx,
		DomainHash:  domainHash,
		MessageHash: messageHash,
		Call:        *call,
	}

	return result, nil
}

// stripChainPrefix removes chain prefixes like "oeth:", "eth:", etc. from addresses
func stripChainPrefix(address string) string {
	// Check if the address contains a colon
	if idx := strings.Index(address, ":"); idx != -1 {
		// Return everything after the colon
		return address[idx+1:]
	}
	return address
}
