package core

import (
	"encoding/hex"
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
	Safe           string `json:"safe"`
	SafeVersion    string `json:"safe_version"`
	Nonce          int    `json:"nonce"`
	Data           string `json:"data"`
	Operation      int    `json:"operation"`
	To             string `json:"to"`
	SafeTxGas      int    `json:"safe_tx_gas"`
	BaseGas        int    `json:"base_gas"`
	GasPrice       int    `json:"gas_price"`
	GasToken       string `json:"gas_token"`
	RefundReceiver string `json:"refund_receiver"`
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
		tx.SafeVersion = tx.Nested.SafeVersion

		// The parent has its own gas and refund configuration; inheriting the child's would
		// hash and display a refund the parent transaction does not pay.
		tx.SafeTxGas = tx.Nested.SafeTxGas
		tx.BaseGas = tx.Nested.BaseGas
		tx.GasPrice = tx.Nested.GasPrice
		tx.GasToken = tx.Nested.GasToken
		tx.RefundReceiver = tx.Nested.RefundReceiver
	}

	// Verify the main transaction
	result, err := verifyTransactionInternal(tx, options)
	if err != nil {
		return nil, err
	}

	if nestedResult != nil {
		if err := checkApprovesChild(result.Call, nestedResult); err != nil {
			return nil, err
		}
	}

	// Attach nested result if it exists
	result.NestedResult = nestedResult

	return result, nil
}

// approveHashCall is one approveHash call found in a parent transaction's call tree.
type approveHashCall struct {
	target string
	hash   string
}

// checkApprovesChild confirms every approveHash call in the parent transaction is the one
// child approval we display, matched on target Safe as well as hash. Matching the hash
// alone lets a decoy subcall carry the hash we print while the call that reaches the child
// Safe approves another, and an extra approval is one this output has no room to show.
func checkApprovesChild(parentCall CallData, child *VerificationResult) error {
	approvals := collectApproveHashCalls(parentCall)
	if len(approvals) == 0 {
		return fmt.Errorf("parent transaction does not call approveHash, so it cannot approve child transaction %s", child.ApproveHash)
	}
	for _, approval := range approvals {
		if !strings.EqualFold(approval.target, child.Transaction.Safe) || !strings.EqualFold(approval.hash, child.ApproveHash) {
			return fmt.Errorf("parent transaction calls approveHash(%q) on %s, but the only child transaction shown is %s on %s",
				approval.hash, approval.target, child.ApproveHash, child.Transaction.Safe)
		}
	}
	return nil
}

// collectApproveHashCalls returns every approveHash call in a call tree, including the
// multicall subcalls superchain-ops wraps them in.
func collectApproveHashCalls(call CallData) []approveHashCall {
	var calls []approveHashCall
	if call.FunctionName == "approveHash" {
		// A call whose argument we could not decode is still recorded, so undecodable
		// calldata cannot smuggle an approval past the check above.
		hash := ""
		if parsed, ok := call.ParsedData.(map[string]interface{}); ok {
			hash, _ = parsed["hashToApprove"].(string)
		}
		calls = append(calls, approveHashCall{target: call.Target, hash: hash})
	}
	for _, subcall := range call.SubCalls {
		calls = append(calls, collectApproveHashCalls(subcall)...)
	}
	return calls
}

// verifyTransactionInternal contains the core verification logic
func verifyTransactionInternal(tx SafeTransaction, options VerifyOptions) (*VerificationResult, error) {
	// Strip chain prefix from the target address (e.g., "oeth:", "eth:")
	tx.To = StripChainPrefix(tx.To)
	tx.Safe = StripChainPrefix(tx.Safe)

	if err := tx.validate(); err != nil {
		return nil, err
	}

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

// validate rejects values the EIP-712 hash would otherwise silently coerce: HexToAddress
// keeps only the last 20 bytes of an over-long address and FromHex decodes invalid or
// odd-length calldata to something shorter, either of which hashes as a transaction other
// than the one we display.
func (tx SafeTransaction) validate() error {
	addresses := []struct {
		field, value string
		optional     bool
	}{
		{"safe", tx.Safe, false},
		{"to", tx.To, false},
		// The Safe API omits the refund fields when they are unset, and the hash reads an
		// absent one as the zero address.
		{"gas_token", tx.GasToken, true},
		{"refund_receiver", tx.RefundReceiver, true},
	}
	for _, address := range addresses {
		if address.optional && address.value == "" {
			continue
		}
		if err := validateAddress(address.field, address.value); err != nil {
			return err
		}
	}
	return validateHex("data", tx.Data)
}

func validateAddress(field, value string) error {
	if len(strings.TrimPrefix(value, "0x")) != 40 {
		return fmt.Errorf("%s must be a 20-byte hex address, got %q", field, value)
	}
	return validateHex(field, value)
}

func validateHex(field, value string) error {
	if _, err := hex.DecodeString(strings.TrimPrefix(value, "0x")); err != nil {
		return fmt.Errorf("%s must be hex, got %q: %w", field, value, err)
	}
	return nil
}

// stripChainPrefix removes chain prefixes like "oeth:", "eth:", etc. from addresses
func StripChainPrefix(address string) string {
	if idx := strings.Index(address, ":"); idx != -1 {
		return address[idx+1:]
	}
	return address
}
