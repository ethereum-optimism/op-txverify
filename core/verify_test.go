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
		packed[i] = call3Value{Target: call.target, Value: big.NewInt(0), CallData: call.data}
	}
	return packCall(t, "aggregate3Value", packed)
}

// multiSendData encodes the operation, target, value, length and data records multiSend
// concatenates into a single bytes argument.
func multiSendData(t *testing.T, calls []subcall) string {
	t.Helper()
	var packed []byte
	for _, call := range calls {
		packed = append(packed, 0)
		packed = append(packed, call.target.Bytes()...)
		packed = append(packed, common.BigToHash(big.NewInt(0)).Bytes()...)
		packed = append(packed, common.BigToHash(big.NewInt(int64(len(call.data)))).Bytes()...)
		packed = append(packed, call.data...)
	}
	return packCall(t, "multiSend", packed)
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
