package core

import (
	"fmt"
	"math/big"
	"strings"
)

// VerificationResult represents the complete output of the verification process
type VerificationResult struct {
	Transaction  SafeTransaction     `json:"transaction"`
	DomainHash   string              `json:"domainHash"`
	MessageHash  string              `json:"messageHash"`
	ApproveHash  string              `json:"approveHash"`
	Call         CallData            `json:"call"`
	NestedResult *VerificationResult `json:"nestedResult,omitempty"`
}

// Nested represents the data about nested approve hash transactions
type Nested struct {
	Safe        string `json:"safe"`
	SafeVersion string `json:"safe_version"`
	Nonce       int    `json:"nonce"`
	Data        string `json:"data"`
	Operation   int    `json:"operation"`
	To          string `json:"to"`
}

// SafeTransaction represents a Gnosis Safe transaction
type SafeTransaction struct {
	Safe           string   `json:"safe"`
	SafeVersion    string   `json:"safe_version"`
	Chain          int      `json:"chain"`
	To             string   `json:"to"`
	Value          *big.Int `json:"value"`
	Data           string   `json:"data"`
	Operation      int      `json:"operation"`
	SafeTxGas      int      `json:"safe_tx_gas"`
	BaseGas        int      `json:"base_gas"`
	GasPrice       int      `json:"gas_price"`
	GasToken       string   `json:"gas_token"`
	RefundReceiver string   `json:"refund_receiver"`
	Nonce          int      `json:"nonce"`
	Nested         *Nested  `json:"nested,omitempty"`
	Call           CallData `json:"call"`
}

// CallData represents a function call with parsed arguments
type CallData struct {
	Target         string      `json:"target"`
	TargetName     string      `json:"targetName,omitempty"`
	FunctionName   string      `json:"functionName"`
	FunctionData   string      `json:"functionData,omitempty"`
	RawData        string      `json:"rawData,omitempty"`
	ParsedData     interface{} `json:"parsedData,omitempty"`
	SubCalls       []CallData  `json:"subCalls,omitempty"`
	IsDelegateCall bool        `json:"isDelegateCall,omitempty"`
}

// VerifyOptions contains configuration options for verification
type VerifyOptions struct {
	Verbose bool
}

// VerifyTransaction verifies a Safe transaction
func VerifyTransaction(tx SafeTransaction, options VerifyOptions) (*VerificationResult, error) {
	// Check if this is a nested transaction
	var nestedResult *VerificationResult
	if tx.Nested != nil {
		// Verify the inner transaction first
		var err error
		nestedResult, err = verifyTransactionInternal(tx, options)
		if err != nil {
			return nil, fmt.Errorf("failed to verify nested transaction: %w", err)
		}

		// Manipulate the transaction to generate the outer result
		tx.To = tx.Nested.To
		tx.Safe = tx.Nested.Safe
		tx.Nonce = tx.Nested.Nonce
		tx.Operation = tx.Nested.Operation
		tx.Value = big.NewInt(0)
		tx.Data = tx.Nested.Data
	}

	// Verify the main transaction
	result, err := verifyTransactionInternal(tx, options)
	if err != nil {
		return nil, err
	}

	// Attach nested result if it exists
	result.NestedResult = nestedResult

	return result, nil
}

// verifyTransactionInternal contains the core verification logic
func verifyTransactionInternal(tx SafeTransaction, options VerifyOptions) (*VerificationResult, error) {
	// Strip chain prefix from the target address (e.g., "oeth:", "eth:")
	tx.To = StripChainPrefix(tx.To)
	tx.Safe = StripChainPrefix(tx.Safe)

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

	approveHash, err := CalculateApproveHash(tx)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate approve hash: %w", err)
	}

	// Create the verification result
	result := &VerificationResult{
		Transaction: tx,
		DomainHash:  domainHash,
		MessageHash: messageHash,
		ApproveHash: approveHash,
		Call:        *call,
	}

	return result, nil
}

// stripChainPrefix removes chain prefixes like "oeth:", "eth:", etc. from addresses
func StripChainPrefix(address string) string {
	if idx := strings.Index(address, ":"); idx != -1 {
		return address[idx+1:]
	}
	return address
}
