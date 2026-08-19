package core

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
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

// UnmarshalJSON accepts `value` as a JSON number or a decimal string. The Safe Transaction Service
// returns a string, and passing it through unquoted is the only way to keep precision above 2^53-1
// wei, but `*big.Int` alone rejects the quoted form. Both are accepted so payloads written before
// this change still parse.
func (t *SafeTransaction) UnmarshalJSON(data []byte) error {
	type alias SafeTransaction // sheds the method set, so this does not recurse
	aux := struct {
		Value json.RawMessage `json:"value"`
		*alias
	}{alias: (*alias)(t)}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	raw := string(aux.Value)
	if raw == "" || raw == "null" {
		// Zero rather than nil: a nil Value panics when the struct hash is ABI-packed.
		t.Value = big.NewInt(0)
		return nil
	}
	if raw[0] == '"' {
		if err := json.Unmarshal(aux.Value, &raw); err != nil {
			return fmt.Errorf("invalid value: %w", err)
		}
	}

	value, err := ParseWei(raw)
	if err != nil {
		return err
	}
	t.Value = value

	// The remaining uint256 fields are decoded into int, so encoding/json already rejects anything
	// beyond int64. Negatives and an out-of-range operation still get through, and both would be
	// ABI-packed into unsigned types without complaint - producing a hash for values other than the
	// ones displayed.
	if t.Operation != 0 && t.Operation != 1 {
		return fmt.Errorf("invalid operation %d: Safe allows only 0 (CALL) or 1 (DELEGATECALL)", t.Operation)
	}
	for name, field := range map[string]int{
		"safe_tx_gas": t.SafeTxGas,
		"base_gas":    t.BaseGas,
		"gas_price":   t.GasPrice,
		"nonce":       t.Nonce,
	} {
		if field < 0 {
			return fmt.Errorf("invalid %s %d: must not be negative", name, field)
		}
	}
	return nil
}

// ParseWei parses a base-10 wei amount, rejecting anything outside the uint256 range. The range
// check matters because abi.Arguments.Pack reduces modulo 2^256 rather than erroring, so a negative
// or oversized value would print one number in the decode and hash a different one.
func ParseWei(raw string) (*big.Int, error) {
	value, ok := new(big.Int).SetString(raw, 10)
	if !ok {
		return nil, fmt.Errorf("invalid value %q: not a base-10 integer", raw)
	}
	if value.Sign() < 0 || value.BitLen() > 256 {
		return nil, fmt.Errorf("invalid value %q: outside the uint256 range", raw)
	}
	return value, nil
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
		if err := checkNestedGasFields(tx); err != nil {
			return nil, err
		}

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

// checkNestedGasFields rejects a nested transaction whose gas fields are non-zero. Nested carries
// no gas fields of its own, so the outer hash is computed from the inner transaction's — Go and the
// Safe contract would then agree with each other and both disagree with the hardware wallet, the one
// mismatch nothing downstream can catch. Every superchain-ops nested task uses zeros, so this exists
// to make that assumption loud rather than silent.
func checkNestedGasFields(tx SafeTransaction) error {
	switch {
	case tx.SafeTxGas != 0:
		return fmt.Errorf("nested transaction requires safe_tx_gas to be 0, got %d", tx.SafeTxGas)
	case tx.BaseGas != 0:
		return fmt.Errorf("nested transaction requires base_gas to be 0, got %d", tx.BaseGas)
	case tx.GasPrice != 0:
		return fmt.Errorf("nested transaction requires gas_price to be 0, got %d", tx.GasPrice)
	case common.HexToAddress(tx.GasToken) != (common.Address{}):
		return fmt.Errorf("nested transaction requires gas_token to be the zero address, got %s", tx.GasToken)
	case common.HexToAddress(tx.RefundReceiver) != (common.Address{}):
		return fmt.Errorf("nested transaction requires refund_receiver to be the zero address, got %s", tx.RefundReceiver)
	}
	return nil
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
