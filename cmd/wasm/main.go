//go:build js && wasm

// Command wasm exposes op-txverify's Safe transaction verification to a browser page as a single
// globalThis.txvVerify(txJSON) function. It performs no I/O: the page owns any network access.
package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"regexp"
	"strconv"
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
// Each entry carries its own structured call and Safe fields. This side holds exact big.Int values
// where JSON.parse would round above 2^53-1, and self-contained entries mean the page cannot pair
// one transaction's hashes with another's fields, which differ for a nested transaction.
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
		call, err := structuredCall(c.result.Call, tx.Value, tx.Operation)
		if err != nil {
			return nil, fmt.Errorf("failed to present %s check: %w", c.label, err)
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
			"call":        call,
			"safeFields":  structuredSafeFields(tx),
		})
	}
	return checks, nil
}

var addressPattern = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)

func structuredCall(call core.CallData, value *big.Int, operation int) (map[string]any, error) {
	view := map[string]any{
		"target":    call.Target,
		"operation": operationName(operation),
		"arguments": []any{},
	}
	if call.TargetName != "" {
		view["targetLabel"] = call.TargetName
	}

	calldata := call.Calldata
	if calldata == "" {
		calldata = call.RawData
	}
	cleanData := strings.TrimPrefix(calldata, "0x")
	if cleanData == "" {
		if value != nil && value.Sign() > 0 {
			view["functionName"] = "Send native ETH"
			view["arguments"] = []any{
				argument("recipient", "address", call.Target),
				argument("value", "uint256", value.String()),
			}
		} else {
			view["functionName"] = "No calldata"
		}
		return addSubcalls(view, call.SubCalls)
	}

	info, decoded, err := decodeCallArguments(cleanData)
	if err != nil {
		return nil, err
	}
	if info == nil {
		view["functionName"] = "Unknown function"
		selector := cleanData
		if len(selector) > 8 {
			selector = selector[:8]
		}
		view["selector"] = "0x" + strings.ToLower(selector)
		view["rawCalldata"] = "0x" + cleanData
		return addSubcalls(view, call.SubCalls)
	}

	view["functionName"] = info.Name
	view["signature"] = info.Signature
	view["arguments"] = decoded
	return addSubcalls(view, call.SubCalls)
}

func decodeCallArguments(cleanData string) (*core.FunctionInfo, []any, error) {
	if len(cleanData) < 8 {
		return nil, nil, nil
	}
	info, ok := core.KnownFunctions[strings.ToLower(cleanData[:8])]
	if !ok {
		return nil, nil, nil
	}
	data, err := hex.DecodeString(cleanData[8:])
	if err != nil {
		return nil, nil, nil
	}
	values, err := info.ABI.Inputs.Unpack(data)
	if err != nil {
		return nil, nil, nil
	}
	arguments := make([]any, len(values))
	for i, value := range values {
		converted, err := exactABIValue(info.ABI.Inputs[i].Type, value)
		if err != nil {
			return nil, nil, fmt.Errorf("convert %s argument %s: %w", info.Signature, info.ABI.Inputs[i].Name, err)
		}
		name := info.ABI.Inputs[i].Name
		if name == "" {
			name = fmt.Sprintf("arg%d", i)
		}
		arguments[i] = argument(name, info.ABI.Inputs[i].Type.String(), converted)
	}
	return &info, arguments, nil
}

func exactABIValue(typ abi.Type, value any) (any, error) {
	switch typ.T {
	case abi.IntTy, abi.UintTy:
		switch value := value.(type) {
		case *big.Int:
			return value.String(), nil
		case big.Int:
			return value.String(), nil
		}
		rv := reflect.ValueOf(value)
		if rv.IsValid() {
			switch rv.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				return strconv.FormatInt(rv.Int(), 10), nil
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				return strconv.FormatUint(rv.Uint(), 10), nil
			}
		}
		return nil, fmt.Errorf("integer has Go type %T", value)
	case abi.BoolTy:
		result, ok := value.(bool)
		if !ok {
			return nil, fmt.Errorf("bool has Go type %T", value)
		}
		return result, nil
	case abi.StringTy:
		result, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("string has Go type %T", value)
		}
		return result, nil
	case abi.AddressTy:
		result, ok := value.(common.Address)
		if !ok {
			return nil, fmt.Errorf("address has Go type %T", value)
		}
		return result.Hex(), nil
	case abi.BytesTy, abi.FixedBytesTy, abi.FunctionTy:
		return exactBytes(value)
	case abi.SliceTy, abi.ArrayTy:
		rv := reflect.ValueOf(value)
		if rv.Kind() != reflect.Array && rv.Kind() != reflect.Slice {
			return nil, fmt.Errorf("array has Go type %T", value)
		}
		items := make([]any, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			converted, err := exactABIValue(*typ.Elem, rv.Index(i).Interface())
			if err != nil {
				return nil, fmt.Errorf("element %d: %w", i, err)
			}
			items[i] = converted
		}
		return items, nil
	case abi.TupleTy:
		rv := reflect.ValueOf(value)
		if rv.Kind() == reflect.Pointer {
			rv = rv.Elem()
		}
		if rv.Kind() != reflect.Struct || rv.NumField() != len(typ.TupleElems) {
			return nil, fmt.Errorf("tuple has Go type %T", value)
		}
		fields := make([]any, len(typ.TupleElems))
		for i, elem := range typ.TupleElems {
			converted, err := exactABIValue(*elem, rv.Field(i).Interface())
			if err != nil {
				return nil, fmt.Errorf("tuple field %d: %w", i, err)
			}
			name := ""
			if i < len(typ.TupleRawNames) {
				name = typ.TupleRawNames[i]
			}
			if name == "" {
				name = fmt.Sprintf("field%d", i)
			}
			fields[i] = argument(name, elem.String(), converted)
		}
		return fields, nil
	default:
		return nil, fmt.Errorf("unsupported ABI type %s", typ.String())
	}
}

func exactBytes(value any) (string, error) {
	rv := reflect.ValueOf(value)
	if rv.Kind() != reflect.Array && rv.Kind() != reflect.Slice {
		return "", fmt.Errorf("bytes have Go type %T", value)
	}
	data := make([]byte, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		if rv.Index(i).Kind() != reflect.Uint8 {
			return "", fmt.Errorf("byte %d has Go type %s", i, rv.Index(i).Kind())
		}
		data[i] = byte(rv.Index(i).Uint())
	}
	return hexutil.Encode(data), nil
}

func addSubcalls(view map[string]any, subcalls []core.CallData) (map[string]any, error) {
	if len(subcalls) == 0 {
		return view, nil
	}
	calls := make([]any, len(subcalls))
	for i, subcall := range subcalls {
		operation := 0
		if subcall.IsDelegateCall {
			operation = 1
		}
		call, err := structuredCall(subcall, subcall.Value, operation)
		if err != nil {
			return nil, fmt.Errorf("subcall %d: %w", i, err)
		}
		calls[i] = call
	}
	view["calls"] = calls
	return view, nil
}

func argument(name, typ string, value any) map[string]any {
	return map[string]any{"name": name, "type": typ, "value": value}
}

func operationName(operation int) string {
	if operation == 1 {
		return "DELEGATECALL"
	}
	return "CALL"
}

func structuredSafeFields(tx core.SafeTransaction) map[string]any {
	value := "0"
	if tx.Value != nil {
		value = tx.Value.String()
	}
	return map[string]any{
		"to":             core.StripChainPrefix(tx.To),
		"value":          value,
		"data":           tx.Data,
		"operation":      strconv.Itoa(tx.Operation),
		"safeTxGas":      strconv.Itoa(tx.SafeTxGas),
		"baseGas":        strconv.Itoa(tx.BaseGas),
		"gasPrice":       strconv.Itoa(tx.GasPrice),
		"gasToken":       tx.GasToken,
		"refundReceiver": tx.RefundReceiver,
		"nonce":          strconv.Itoa(tx.Nonce),
	}
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
