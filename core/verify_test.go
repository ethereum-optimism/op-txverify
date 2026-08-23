package core

import (
	"encoding/hex"
	"encoding/json"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func testTransaction() SafeTransaction {
	return SafeTransaction{
		Safe:           "0x847B5c174615B1B7fDF770882256e2D3E95b9D92",
		SafeVersion:    "1.3.0",
		Chain:          1,
		To:             "0x4200000000000000000000000000000000000016",
		Value:          big.NewInt(0),
		Data:           "0x",
		GasToken:       "0x0000000000000000000000000000000000000000",
		RefundReceiver: "0x0000000000000000000000000000000000000000",
		Nonce:          1,
	}
}

// subcall is one call inside the multicalls superchain-ops wraps approvals in.
type subcall struct {
	target common.Address
	value  *big.Int
	data   []byte
}

func approveHashData(hash string) []byte {
	return common.FromHex("0xd4d9bdcd" + strings.TrimPrefix(hash, "0x"))
}

// packCall builds calldata for a function this tool knows how to decode.
func packCall(t *testing.T, name string, args ...interface{}) string {
	t.Helper()
	for selector, function := range KnownFunctions {
		if function.Name != name {
			continue
		}
		packed, err := function.ABI.Inputs.Pack(args...)
		if err != nil {
			t.Fatalf("failed to pack %s: %v", name, err)
		}
		return "0x" + selector + hex.EncodeToString(packed)
	}
	t.Fatalf("no known function named %s", name)
	return ""
}

func aggregate3Data(t *testing.T, calls []subcall) string {
	t.Helper()
	type call3 struct {
		Target       common.Address
		AllowFailure bool
		CallData     []byte
	}
	packed := make([]call3, len(calls))
	for i, call := range calls {
		packed[i] = call3{Target: call.target, CallData: call.data}
	}
	return packCall(t, "aggregate3", packed)
}

func aggregate3ValueData(t *testing.T, calls []subcall) string {
	t.Helper()
	type call3Value struct {
		Target       common.Address
		AllowFailure bool
		Value        *big.Int
		CallData     []byte
	}
	packed := make([]call3Value, len(calls))
	for i, call := range calls {
		value := call.value
		if value == nil {
			value = big.NewInt(0)
		}
		packed[i] = call3Value{Target: call.target, Value: value, CallData: call.data}
	}
	return packCall(t, "aggregate3Value", packed)
}

// multiSendData encodes the operation, target, value, length and data records multiSend
// concatenates into a single bytes argument.
func multiSendData(t *testing.T, calls []subcall) string {
	return multiSendDataWithOperation(t, 0, calls)
}

func multiSendDataWithOperation(t *testing.T, operation byte, calls []subcall) string {
	t.Helper()
	var packed []byte
	for _, call := range calls {
		value := call.value
		if value == nil {
			value = big.NewInt(0)
		}
		packed = append(packed, operation)
		packed = append(packed, call.target.Bytes()...)
		packed = append(packed, common.BigToHash(value).Bytes()...)
		packed = append(packed, common.BigToHash(big.NewInt(int64(len(call.data)))).Bytes()...)
		packed = append(packed, call.data...)
	}
	return multiSendPayload(t, packed)
}

func multiSendPayload(t *testing.T, payload []byte) string {
	t.Helper()
	return packCall(t, "multiSend", payload)
}

func TestParseTransactionDataRejectsInvalidMultiSendOperation(t *testing.T) {
	data := multiSendDataWithOperation(t, 2, []subcall{{
		target: common.HexToAddress("0x1111111111111111111111111111111111111111"),
		data:   approveHashData("0x" + strings.Repeat("1", 64)),
	}})

	_, err := ParseTransactionData(SafeMultisendAddress, data, MainnetChainID, VerifyOptions{})
	if err == nil || !strings.Contains(err.Error(), "multiSend operation 2") {
		t.Fatalf("ParseTransactionData error = %v, want invalid multiSend operation 2", err)
	}
}

func TestParseTransactionDataRejectsTruncatedMultiSendRecords(t *testing.T) {
	target := common.HexToAddress("0x1111111111111111111111111111111111111111")
	truncatedBody := append([]byte{0}, target.Bytes()...)
	truncatedBody = append(truncatedBody, common.BigToHash(big.NewInt(0)).Bytes()...)
	truncatedBody = append(truncatedBody, common.BigToHash(big.NewInt(1)).Bytes()...)

	for _, tc := range []struct {
		name    string
		payload []byte
		want    string
	}{
		{"incomplete header", make([]byte, 84), "truncated multiSend header"},
		{"declared body overrun", truncatedBody, "truncated multiSend body"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseTransactionData(SafeMultisendAddress, multiSendPayload(t, tc.payload), MainnetChainID, VerifyOptions{})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("ParseTransactionData error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestParseTransactionDataRejectsMalformedMultiSendInsideAggregate3(t *testing.T) {
	malformedMultiSend := multiSendPayload(t, make([]byte, 84))
	child := subcall{
		target: common.HexToAddress(SafeMultisendAddress),
		data:   common.FromHex(malformedMultiSend),
	}

	for _, tc := range []struct {
		name string
		data string
	}{
		{"aggregate3", aggregate3Data(t, []subcall{child})},
		{"aggregate3Value", aggregate3ValueData(t, []subcall{child})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseTransactionData(Multicall3Address, tc.data, MainnetChainID, VerifyOptions{})
			if err == nil || !strings.Contains(err.Error(), "truncated multiSend header") {
				t.Fatalf("ParseTransactionData error = %v, want nested multiSend rejection", err)
			}
		})
	}
}

func TestParseTransactionData_EmptyCalldataRemainsValueAgnostic(t *testing.T) {
	call, err := ParseTransactionData(testTransaction().To, "0x", MainnetChainID, VerifyOptions{})
	if err != nil {
		t.Fatalf("ParseTransactionData: %v", err)
	}
	if call.FunctionName != "unknown" {
		t.Fatalf("FunctionName = %q, want value-agnostic unknown", call.FunctionName)
	}
}

func TestVerifyTransaction_NormalizesEmptyCalldataWithTransactionValue(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		value    *big.Int
		function string
	}{
		{"native transfer", "", big.NewInt(1), "Send native ETH"},
		{"zero-value call", "0x", big.NewInt(0), "No calldata"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tx := testTransaction()
			tx.Data = tc.data
			tx.Value = tc.value
			wantHash, err := CalculateApproveHash(tx)
			if err != nil {
				t.Fatalf("CalculateApproveHash: %v", err)
			}

			result, err := VerifyTransaction(tx, VerifyOptions{})
			if err != nil {
				t.Fatalf("VerifyTransaction: %v", err)
			}
			if result.Call.FunctionName != tc.function {
				t.Fatalf("FunctionName = %q, want %q", result.Call.FunctionName, tc.function)
			}
			if result.Call.Target != tx.To || result.Call.RawData != "0x" {
				t.Fatalf("call = %+v, want target %s and raw calldata 0x", result.Call, tx.To)
			}
			if result.ApproveHash != wantHash {
				t.Fatalf("ApproveHash = %s, want unchanged hash %s", result.ApproveHash, wantHash)
			}
		})
	}
}

func TestVerifyTransaction_NormalizesEmptyCalldataForNestedChild(t *testing.T) {
	child := testTransaction()
	child.Data = "0x"
	child.Value = big.NewInt(1)
	childHash, err := CalculateApproveHash(child)
	if err != nil {
		t.Fatalf("CalculateApproveHash: %v", err)
	}

	tx := child
	tx.Nested = &Nested{
		Safe:        ProxyAdminOwner,
		SafeVersion: "1.3.0",
		Nonce:       7,
		To:          child.Safe,
		Data:        "0x" + hex.EncodeToString(approveHashData(childHash)),
	}
	result, err := VerifyTransaction(tx, VerifyOptions{})
	if err != nil {
		t.Fatalf("VerifyTransaction: %v", err)
	}
	childCall := result.NestedResult.Call
	if childCall.FunctionName != "Send native ETH" {
		t.Fatalf("child FunctionName = %q, want Send native ETH", childCall.FunctionName)
	}
	if childCall.Target != child.To || childCall.RawData != "0x" {
		t.Fatalf("child call = %+v, want target %s and raw calldata 0x", childCall, child.To)
	}
}

func TestVerifyTransaction_NormalizesValueBearingEmptyCalldataSubcalls(t *testing.T) {
	recipient := common.HexToAddress("0x1111111111111111111111111111111111111111")
	calls := []subcall{
		{target: recipient, value: big.NewInt(1)},
		{target: recipient, value: big.NewInt(0)},
	}

	for _, tc := range []struct {
		name   string
		target string
		data   string
	}{
		{"multiSend", SafeMultisendAddress, multiSendData(t, calls)},
		{"aggregate3Value", Multicall3Address, aggregate3ValueData(t, calls)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx := testTransaction()
			tx.To = tc.target
			tx.Data = tc.data
			wantHash, err := CalculateApproveHash(tx)
			if err != nil {
				t.Fatalf("CalculateApproveHash: %v", err)
			}

			result, err := VerifyTransaction(tx, VerifyOptions{})
			if err != nil {
				t.Fatalf("VerifyTransaction: %v", err)
			}
			if result.ApproveHash != wantHash {
				t.Fatalf("ApproveHash = %s, want %s", result.ApproveHash, wantHash)
			}
			if len(result.Call.SubCalls) != 2 {
				t.Fatalf("len(SubCalls) = %d, want 2", len(result.Call.SubCalls))
			}
			for i, want := range []string{"Send native ETH", "No calldata"} {
				call := result.Call.SubCalls[i]
				if call.FunctionName != want || call.Target != recipient.Hex() || call.RawData != "0x" {
					t.Fatalf("SubCalls[%d] = %+v, want %s to %s with raw calldata 0x", i, call, want, recipient.Hex())
				}
			}
		})
	}
}

func TestVerifyTransaction_PreservesNonzeroValueOnKnownSubcalls(t *testing.T) {
	recipient := common.HexToAddress("0x1111111111111111111111111111111111111111")
	knownCall := approveHashData("0x" + strings.Repeat("1", 64))

	for _, tc := range []struct {
		name   string
		target string
		data   string
	}{
		{"multiSend", SafeMultisendAddress, multiSendData(t, []subcall{{target: recipient, value: big.NewInt(7), data: knownCall}})},
		{"aggregate3Value", Multicall3Address, aggregate3ValueData(t, []subcall{{target: recipient, value: big.NewInt(7), data: knownCall}})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx := testTransaction()
			tx.To = tc.target
			tx.Data = tc.data
			result, err := VerifyTransaction(tx, VerifyOptions{})
			if err != nil {
				t.Fatalf("VerifyTransaction: %v", err)
			}
			child := result.Call.SubCalls[0]
			if child.FunctionName != "approveHash" || child.Value == nil || child.Value.Cmp(big.NewInt(7)) != 0 {
				t.Fatalf("child = %+v, want approveHash carrying 7 wei", child)
			}
		})
	}
}

func TestVerifyTransaction_RejectsNilValue(t *testing.T) {
	tx := testTransaction()
	tx.Value = nil

	_, err := VerifyTransaction(tx, VerifyOptions{})
	if err == nil || !strings.Contains(err.Error(), "value is required") {
		t.Fatalf("VerifyTransaction error = %v, want missing value rejection", err)
	}
}

func TestVerifyTransaction_RejectsSilentlyCoercedFields(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*SafeTransaction)
	}{
		{"over-long to", func(tx *SafeTransaction) {
			tx.To = "0x4200000000000000000000000000000000000016dEadBeefdEadBeefdEadBeefdEadBeefdEadBeef"
		}},
		{"over-long gas token", func(tx *SafeTransaction) {
			tx.GasToken = "0x0000000000000000000000000000000000000000dEadBeefdEadBeefdEadBeefdEadBeefdEadBeef"
		}},
		{"short refund receiver", func(tx *SafeTransaction) {
			tx.RefundReceiver = "0x0"
		}},
		{"chain-prefixed gas token", func(tx *SafeTransaction) {
			tx.GasToken = "oeth:0x0000000000000000000000000000000000000000"
		}},
		{"invalid hex data", func(tx *SafeTransaction) {
			tx.Data = "0xnonsense"
		}},
		{"odd-length hex data", func(tx *SafeTransaction) {
			tx.Data = "0xabc"
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tx := testTransaction()
			tc.mutate(&tx)

			result, err := VerifyTransaction(tx, VerifyOptions{})
			if err == nil {
				t.Fatalf("expected rejection, got message hash %s", result.MessageHash)
			}
			t.Log(err)
		})
	}
}

func TestVerifyTransaction_AcceptsOmittedRefundFields(t *testing.T) {
	// download writes what the Safe API returns, which omits the refund fields when they
	// are unset, and the hash reads an omitted one as the zero address.
	const payload = `{"safe":"0x847B5c174615B1B7fDF770882256e2D3E95b9D92","safe_version":"1.3.0",` +
		`"chain":1,"to":"0x4200000000000000000000000000000000000016","value":0,"data":"0x",` +
		`"gas_token":null,"nonce":1}`

	var tx SafeTransaction
	if err := json.Unmarshal([]byte(payload), &tx); err != nil {
		t.Fatalf("failed to parse payload: %v", err)
	}

	result, err := VerifyTransaction(tx, VerifyOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want, err := CalculateApproveHash(testTransaction())
	if err != nil {
		t.Fatalf("failed to calculate hash: %v", err)
	}
	if result.ApproveHash != want {
		t.Fatalf("hash = %s, want the zero-address hash %s", result.ApproveHash, want)
	}
}

func TestVerifyTransaction_AcceptsMatchingChildHash(t *testing.T) {
	child := testTransaction()
	childHash, err := CalculateApproveHash(child)
	if err != nil {
		t.Fatalf("failed to calculate child hash: %v", err)
	}
	approvals := []subcall{{target: common.HexToAddress(child.Safe), data: approveHashData(childHash)}}

	tests := []struct {
		name string
		to   string
		data string
	}{
		{"direct approveHash", child.Safe, "0x" + hex.EncodeToString(approveHashData(childHash))},
		{"aggregate3", Multicall3Address, aggregate3Data(t, approvals)},
		{"aggregate3Value", Multicall3Address, aggregate3ValueData(t, approvals)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tx := child
			tx.Nested = &Nested{
				Safe:        ProxyAdminOwner,
				SafeVersion: "1.3.0",
				Nonce:       7,
				To:          tc.to,
				Data:        tc.data,
			}

			result, err := VerifyTransaction(tx, VerifyOptions{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.NestedResult.ApproveHash != childHash {
				t.Fatalf("child hash = %s, want %s", result.NestedResult.ApproveHash, childHash)
			}
		})
	}
}

func TestVerifyTransaction_RejectsUnrelatedChildHash(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{"approves a different hash", "0xd4d9bdcd" + strings.Repeat("0", 64)},
		{"does not approve anything", "0x"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tx := testTransaction()
			tx.Nested = &Nested{
				Safe:        ProxyAdminOwner,
				SafeVersion: "1.3.0",
				Nonce:       7,
				To:          tx.Safe,
				Data:        tc.data,
			}

			result, err := VerifyTransaction(tx, VerifyOptions{})
			if err == nil {
				t.Fatalf("expected rejection, got message hash %s", result.MessageHash)
			}
			t.Log(err)
		})
	}
}

func TestVerifyTransaction_RejectsUnaccountedApproveHash(t *testing.T) {
	child := testTransaction()
	childHash, err := CalculateApproveHash(child)
	if err != nil {
		t.Fatalf("failed to calculate child hash: %v", err)
	}

	childSafe := common.HexToAddress(child.Safe)
	decoy := common.HexToAddress("0x000000000000000000000000000000000000dEaD")
	attackerHash := "0x" + strings.Repeat("ab", 32)

	tests := []struct {
		name  string
		calls []subcall
	}{
		{"decoy subcall carries the displayed hash", []subcall{
			{target: decoy, data: approveHashData(childHash)},
			{target: childSafe, data: approveHashData(attackerHash)},
		}},
		{"extra approval the output cannot show", []subcall{
			{target: childSafe, data: approveHashData(childHash)},
			{target: decoy, data: approveHashData(attackerHash)},
		}},
		{"approval aimed at another Safe", []subcall{
			{target: decoy, data: approveHashData(childHash)},
		}},
		{"undecodable approveHash", []subcall{
			{target: childSafe, data: approveHashData(childHash)},
			{target: childSafe, data: common.FromHex("0xd4d9bdcd")},
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tx := child
			tx.Nested = &Nested{
				Safe:        ProxyAdminOwner,
				SafeVersion: "1.3.0",
				Nonce:       7,
				To:          Multicall3Address,
				Data:        aggregate3Data(t, tc.calls),
			}

			result, err := VerifyTransaction(tx, VerifyOptions{})
			if err == nil {
				t.Fatalf("expected rejection, got child hash %s", result.NestedResult.ApproveHash)
			}
			t.Log(err)
		})
	}
}

func TestVerifyTransaction_ParentDoesNotInheritChildRefundFields(t *testing.T) {
	child := testTransaction()
	child.SafeTxGas = 111
	child.BaseGas = 222
	child.GasPrice = 12345
	child.GasToken = "0x000000000000000000000000000000000000bEEF"
	childHash, err := CalculateApproveHash(child)
	if err != nil {
		t.Fatalf("failed to calculate child hash: %v", err)
	}

	tx := child
	tx.Nested = &Nested{
		Safe:        ProxyAdminOwner,
		SafeVersion: "1.3.0",
		Nonce:       7,
		To:          child.Safe,
		Data:        "0x" + hex.EncodeToString(approveHashData(childHash)),
	}

	result, err := VerifyTransaction(tx, VerifyOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	parent := result.Transaction
	if parent.SafeTxGas != 0 || parent.BaseGas != 0 || parent.GasPrice != 0 || parent.GasToken != "" {
		t.Errorf("parent inherited the child's gas fields: %+v", parent)
	}
	if nested := result.NestedResult.Transaction; nested.GasPrice != 12345 || nested.GasToken != child.GasToken {
		t.Errorf("child lost its own gas fields: %+v", nested)
	}
}

func TestCollectApproveHashCalls_UnwrapsMulticalls(t *testing.T) {
	childSafe := common.HexToAddress(ProxyAdminOwner)
	hash := "0x493ad64b8f788ed9808c7bf527a10a017d9f263bb7889868ce18b451d685762d"
	calls := []subcall{{target: childSafe, data: approveHashData(hash)}}

	tests := []struct {
		name   string
		target string
		data   string
	}{
		{"aggregate3", Multicall3Address, aggregate3Data(t, calls)},
		{"aggregate3Value", Multicall3Address, aggregate3ValueData(t, calls)},
		{"multiSend", SafeMultisendCallOnly141, multiSendData(t, calls)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			call, err := ParseTransactionData(tc.target, tc.data, MainnetChainID, VerifyOptions{})
			if err != nil {
				t.Fatalf("failed to parse %s: %v", tc.name, err)
			}

			got := collectApproveHashCalls(*call)
			want := []approveHashCall{{target: childSafe.Hex(), hash: hash}}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("collectApproveHashCalls = %+v, want %+v", got, want)
			}
		})
	}
}

func TestSafeTransactionValueUnmarshal(t *testing.T) {
	maxUint256, _ := new(big.Int).SetString("115792089237316195423570985008687907853269984665640564039457584007913129639935", 10)

	testCases := []struct {
		name     string
		json     string
		expected *big.Int
	}{
		// A string above 2^53-1 wei is the case the writer page used to corrupt via parseInt.
		{"decimal string", `{"value":"1000000000000000000"}`, big.NewInt(1000000000000000000)},
		{"bare number", `{"value":1000000000000000000}`, big.NewInt(1000000000000000000)},
		{"max uint256 string", `{"value":"115792089237316195423570985008687907853269984665640564039457584007913129639935"}`, maxUint256},
		{"zero string", `{"value":"0"}`, big.NewInt(0)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var tx SafeTransaction
			if err := json.Unmarshal([]byte(tc.json), &tx); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}
			if tx.Value == nil || tx.Value.Cmp(tc.expected) != 0 {
				t.Errorf("Value mismatch. Got %v, want %s", tx.Value, tc.expected)
			}
		})
	}

	// Offline transaction files and compressed URL payloads omit value for zero-value transactions.
	// It must land on zero, not nil: a nil Value panics when the struct hash is ABI-packed.
	for _, payload := range []string{`{}`, `{"value":null}`} {
		var tx SafeTransaction
		if err := json.Unmarshal([]byte(payload), &tx); err != nil {
			t.Errorf("Failed to unmarshal %s: %v", payload, err)
			continue
		}
		if tx.Value == nil || tx.Value.Sign() != 0 {
			t.Errorf("Expected zero for %s, got %v", payload, tx.Value)
		}
	}

	// A missing or null value must not skip the checks below it. Omitted values are supported for
	// offline and compressed-URL payloads, so an early return there would be a reachable bypass.
	bypassAttempts := []string{
		`{"operation":2}`,
		`{"value":null,"nonce":-1}`,
		`{"nonce":-1}`,
		`{"value":null,"operation":2}`,
		`{"value":null,"safe_tx_gas":-1}`,
		`{"value":null,"base_gas":-1}`,
		`{"value":null,"gas_price":-1}`,
	}
	for _, payload := range bypassAttempts {
		var tx SafeTransaction
		if err := json.Unmarshal([]byte(payload), &tx); err == nil {
			t.Errorf("Expected an error for %s, got operation=%d nonce=%d", payload, tx.Operation, tx.Nonce)
		}
	}

	// Garbage and out-of-range values must fail loudly. abi.Pack reduces modulo 2^256 rather than
	// erroring, so a negative or oversized value would print one number and hash another.
	rejected := []string{
		`{"value":"not-a-number"}`,
		`{"value":"0x1234"}`,
		`{"value":true}`,
		`{"value":-1}`,
		`{"value":"-1"}`,
		`{"value":"115792089237316195423570985008687907853269984665640564039457584007913129639936"}`,
	}
	for _, payload := range rejected {
		var tx SafeTransaction
		if err := json.Unmarshal([]byte(payload), &tx); err == nil {
			t.Errorf("Expected an error for %s, got %v", payload, tx.Value)
		}
	}
}

func TestVerifyTransaction_BindsFieldsToTheRequestedHash(t *testing.T) {
	tx := testTransaction()
	hash, err := CalculateApproveHash(tx)
	if err != nil {
		t.Fatalf("failed to calculate hash: %v", err)
	}

	t.Run("accepts the hash of the fields", func(t *testing.T) {
		bound := tx
		bound.SafeTxHash = hash
		if _, err := VerifyTransaction(bound, VerifyOptions{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	// Every one of these is a field the hash reads, so altering it yields a self-consistent
	// transaction whose hash is no longer the one that was asked for.
	tampered := []struct {
		name  string
		apply func(*SafeTransaction)
	}{
		{"to", func(tx *SafeTransaction) { tx.To = OPL1StandardBridge }},
		{"value", func(tx *SafeTransaction) { tx.Value = big.NewInt(1) }},
		{"data", func(tx *SafeTransaction) { tx.Data = "0xdeadbeef" }},
		{"operation", func(tx *SafeTransaction) { tx.Operation = 1 }},
		{"nonce", func(tx *SafeTransaction) { tx.Nonce = 2 }},
		{"safe_tx_gas", func(tx *SafeTransaction) { tx.SafeTxGas = 1 }},
		{"base_gas", func(tx *SafeTransaction) { tx.BaseGas = 1 }},
		{"gas_price", func(tx *SafeTransaction) { tx.GasPrice = 1 }},
		{"gas_token", func(tx *SafeTransaction) { tx.GasToken = OPL1StandardBridge }},
		{"refund_receiver", func(tx *SafeTransaction) { tx.RefundReceiver = OPL1StandardBridge }},
		{"chain", func(tx *SafeTransaction) { tx.Chain = 10 }},
		{"safe", func(tx *SafeTransaction) { tx.Safe = ProxyAdminOwner }},
		{"safe_version", func(tx *SafeTransaction) { tx.SafeVersion = "1.1.1" }},
	}
	for _, tc := range tampered {
		t.Run("rejects a tampered "+tc.name, func(t *testing.T) {
			bound := tx
			bound.SafeTxHash = hash
			tc.apply(&bound)

			result, err := VerifyTransaction(bound, VerifyOptions{})
			if err == nil {
				t.Fatalf("expected rejection, got hash %s", result.ApproveHash)
			}
			if !strings.Contains(err.Error(), "not to the requested") {
				t.Fatalf("error does not name the binding: %v", err)
			}
		})
	}

	t.Run("rejects a malformed hash", func(t *testing.T) {
		bound := tx
		bound.SafeTxHash = "0xnope"
		if _, err := VerifyTransaction(bound, VerifyOptions{}); err == nil {
			t.Fatal("expected rejection")
		}
	})

	t.Run("an unbound transaction still verifies", func(t *testing.T) {
		if _, err := VerifyTransaction(tx, VerifyOptions{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

// The nested case has two hashes, and each half must answer to its own: the parent's is the hash
// the signer asked about, the child's is what the parent's calldata approves.
func TestVerifyTransaction_BindsBothHalvesOfANestedTransaction(t *testing.T) {
	child := testTransaction()
	childHash, err := CalculateApproveHash(child)
	if err != nil {
		t.Fatalf("failed to calculate child hash: %v", err)
	}

	nested := func() SafeTransaction {
		tx := child
		tx.SafeTxHash = childHash
		tx.Nested = &Nested{
			Safe:        ProxyAdminOwner,
			SafeVersion: "1.3.0",
			Nonce:       7,
			To:          child.Safe,
			Data:        "0x" + hex.EncodeToString(approveHashData(childHash)),
		}
		return tx
	}

	bound := nested()
	parent := bound
	parent.To = parent.Nested.To
	parent.Safe = parent.Nested.Safe
	parent.Nonce = parent.Nested.Nonce
	parent.Value = big.NewInt(0)
	parent.Data = parent.Nested.Data
	parentHash, err := CalculateApproveHash(parent)
	if err != nil {
		t.Fatalf("failed to calculate parent hash: %v", err)
	}

	t.Run("accepts both hashes", func(t *testing.T) {
		tx := nested()
		tx.Nested.SafeTxHash = parentHash
		result, err := VerifyTransaction(tx, VerifyOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ApproveHash != parentHash || result.NestedResult.ApproveHash != childHash {
			t.Fatalf("hashes = %s / %s, want %s / %s",
				result.ApproveHash, result.NestedResult.ApproveHash, parentHash, childHash)
		}
	})

	t.Run("rejects a parent hash the parent fields do not produce", func(t *testing.T) {
		tx := nested()
		tx.Nested.SafeTxHash = childHash
		if _, err := VerifyTransaction(tx, VerifyOptions{}); err == nil {
			t.Fatal("expected rejection")
		}
	})

	t.Run("rejects a child hash the child fields do not produce", func(t *testing.T) {
		tx := nested()
		tx.Nested.SafeTxHash = parentHash
		tx.SafeTxHash = parentHash
		if _, err := VerifyTransaction(tx, VerifyOptions{}); err == nil {
			t.Fatal("expected rejection")
		}
	})
}
