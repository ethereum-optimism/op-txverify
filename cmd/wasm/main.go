//go:build js && wasm

// Command wasm exposes op-txverify's Safe transaction verification to a browser page as a single
// globalThis.txvVerify(txJSON) function. It performs no I/O: the page owns any network access.
package main

import (
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"syscall/js"

	"github.com/ethereum-optimism/op-txverify/core"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// encodeTransactionDataABI is the Safe getter returning the EIP-712 preimage
// 0x1901 || domainHash || messageHash. The page eth_calls it so the hashes shown to the signer are
// corroborated by the Safe itself rather than by this module alone.
const encodeTransactionDataABI = `[{"inputs":[{"name":"to","type":"address"},{"name":"value","type":"uint256"},{"name":"data","type":"bytes"},{"name":"operation","type":"uint8"},{"name":"safeTxGas","type":"uint256"},{"name":"baseGas","type":"uint256"},{"name":"gasPrice","type":"uint256"},{"name":"gasToken","type":"address"},{"name":"refundReceiver","type":"address"},{"name":"_nonce","type":"uint256"}],"name":"encodeTransactionData","outputs":[{"name":"","type":"bytes"}],"stateMutability":"view","type":"function"}]`

func main() {
	js.Global().Set("txvVerify", js.FuncOf(txvVerify))
	<-make(chan struct{})
}

// txvVerify returns {result, contractChecks} or {error}. `result` is the JSON encoding of
// core.VerificationResult as a string, not an object: decoding it here would round `value` through
// a float64 and lose wei precision above 2^53-1, so the page parses it however it needs to.
func txvVerify(_ js.Value, args []js.Value) (out any) {
	// A panic would tear down the whole module and leave the page blank, so surface it as an error.
	defer func() {
		if r := recover(); r != nil {
			out = map[string]any{"error": fmt.Sprintf("panic: %v", r)}
		}
	}()

	if len(args) != 1 || args[0].Type() != js.TypeString {
		return map[string]any{"error": "txvVerify expects one argument: the transaction JSON as a string"}
	}

	var tx core.SafeTransaction
	if err := json.Unmarshal([]byte(args[0].String()), &tx); err != nil {
		return map[string]any{"error": fmt.Sprintf("invalid transaction JSON: %v", err)}
	}

	// VerifyTransaction, not the Calculate* functions: only it performs the nested swap that yields
	// the hash pair the hardware wallet displays. Hashing a nested transaction's top-level fields
	// would produce the inner transaction's hashes instead.
	result, err := core.VerifyTransaction(tx, core.VerifyOptions{})
	if err != nil {
		return map[string]any{"error": err.Error()}
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		return map[string]any{"error": fmt.Sprintf("failed to encode result: %v", err)}
	}

	checks, err := contractChecks(result)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}

	return map[string]any{"result": string(encoded), "contractChecks": checks}
}

// contractChecks lists the eth_calls the page should make. "ledger" is the pair the signer compares
// against the device; "inner" is the nested transaction being approved, when there is one.
//
// Each entry carries its own rendered decode and parameters: this side holds exact big.Int values
// where JSON.parse would round above 2^53-1, and self-contained entries mean the page cannot pair
// one transaction's hashes with another's parameters, which for a nested transaction differ.
func contractChecks(result *core.VerificationResult) ([]any, error) {
	checks := make([]any, 0, 2)
	for _, c := range []struct {
		label  string
		result *core.VerificationResult
	}{
		{"ledger", result},
		{"inner", result.NestedResult},
	} {
		if c.result == nil {
			continue
		}
		tx := c.result.Transaction
		target := core.StripChainPrefix(tx.Safe)
		if !addressPattern.MatchString(target) {
			return nil, fmt.Errorf("%s check: %q is not a 20-byte address", c.label, target)
		}
		calldata, err := encodeTransactionDataCalldata(tx)
		if err != nil {
			return nil, fmt.Errorf("failed to encode %s check: %w", c.label, err)
		}
		checks = append(checks, map[string]any{
			"label":       c.label,
			"target":      target,
			"safeVersion": tx.SafeVersion,
			"calldata":    calldata,
			"domainHash":  c.result.DomainHash,
			"messageHash": c.result.MessageHash,
			// keccak256(0x1901 || domainHash || messageHash), so the page can hold this result to
			// the hash it asked for. See core.SafeTransaction.SafeTxHash.
			"approveHash": c.result.ApproveHash,
			"decode":      renderCall(c.result.Call, 0),
			"params":      renderParams(tx),
		})
	}
	return checks, nil
}

var addressPattern = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)

func renderCall(call core.CallData, depth int) string {
	pad := strings.Repeat("  ", depth)
	name := call.Target
	if call.TargetName != "" {
		name = fmt.Sprintf("%s (%s)", call.Target, call.TargetName)
	}
	function := call.FunctionName
	if function == "" {
		function = "(raw)"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s%s -> %s", pad, function, name)
	if call.IsDelegateCall {
		b.WriteString("  [DELEGATECALL]")
	}
	if call.ParsedData != nil {
		fmt.Fprintf(&b, "\n%s  %v", pad, call.ParsedData)
	}
	for _, sub := range call.SubCalls {
		fmt.Fprintf(&b, "\n%s", renderCall(sub, depth+1))
	}
	return b.String()
}

// renderParams prints the exact arguments this check's eth_call uses, so a signer re-reading
// encodeTransactionData on a block explorer enters the same values.
func renderParams(tx core.SafeTransaction) string {
	value := "0"
	if tx.Value != nil {
		value = tx.Value.String()
	}
	return strings.Join([]string{
		fmt.Sprintf("to:             %s", core.StripChainPrefix(tx.To)),
		fmt.Sprintf("value:          %s", value),
		fmt.Sprintf("operation:      %d", tx.Operation),
		fmt.Sprintf("safeTxGas:      %d", tx.SafeTxGas),
		fmt.Sprintf("baseGas:        %d", tx.BaseGas),
		fmt.Sprintf("gasPrice:       %d", tx.GasPrice),
		fmt.Sprintf("gasToken:       %s", tx.GasToken),
		fmt.Sprintf("refundReceiver: %s", tx.RefundReceiver),
		fmt.Sprintf("nonce:          %d", tx.Nonce),
		fmt.Sprintf("data:           %s", tx.Data),
	}, "\n")
}

func encodeTransactionDataCalldata(tx core.SafeTransaction) (string, error) {
	parsed, err := abi.JSON(strings.NewReader(encodeTransactionDataABI))
	if err != nil {
		return "", err
	}

	value := tx.Value
	if value == nil {
		value = big.NewInt(0)
	}

	packed, err := parsed.Pack("encodeTransactionData",
		common.HexToAddress(core.StripChainPrefix(tx.To)),
		value,
		common.FromHex(tx.Data),
		uint8(tx.Operation),
		big.NewInt(int64(tx.SafeTxGas)),
		big.NewInt(int64(tx.BaseGas)),
		big.NewInt(int64(tx.GasPrice)),
		common.HexToAddress(tx.GasToken),
		common.HexToAddress(tx.RefundReceiver),
		big.NewInt(int64(tx.Nonce)),
	)
	if err != nil {
		return "", err
	}
	return hexutil.Encode(packed), nil
}
