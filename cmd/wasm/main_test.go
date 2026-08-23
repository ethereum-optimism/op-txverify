//go:build js && wasm

package main

import (
	"encoding/hex"
	"encoding/json"
	"math/big"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/ethereum-optimism/op-txverify/core"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	testSafe      = "0x847B5c174615B1B7fDF770882256e2D3E95b9D92"
	testRecipient = "0x1111111111111111111111111111111111111111"
	testLargeInt  = "9007199254740993"
)

func TestContractChecksExposeTypedArgumentsWithoutJSONNumbers(t *testing.T) {
	tupleCall := parsedCall(t, "updateDynamicConfig((uint256,uint256),bool)", testRecipient,
		struct {
			MaxSequencerDrift   *big.Int
			SequencerWindowSize *big.Int
		}{mustBigInt(t, testLargeInt), big.NewInt(1200)}, true)
	arrayCall := parsedCall(t, "propose(address[],uint256[],bytes[],string,uint8)", testRecipient,
		[]common.Address{common.HexToAddress(testRecipient)},
		[]*big.Int{mustBigInt(t, testLargeInt), big.NewInt(2)},
		[][]byte{{0xde, 0xad}}, "Upgrade", uint8(7))
	type call3 struct {
		Target       common.Address
		AllowFailure bool
		CallData     []byte
	}
	parentCall := parsedCall(t, core.Aggregate3Sig, core.Multicall3Delegatecall, []call3{
		{Target: common.HexToAddress(testRecipient), CallData: common.FromHex(tupleCall.Calldata)},
		{Target: common.HexToAddress(testRecipient), CallData: common.FromHex(arrayCall.Calldata)},
	})

	result := testResult(core.SafeTransaction{
		Safe: testSafe, SafeVersion: "1.4.1", Chain: 1, To: core.Multicall3Delegatecall,
		Value: mustBigInt(t, testLargeInt), Data: "0x1234", Operation: 1,
		SafeTxGas: 11, BaseGas: 12, GasPrice: 13,
		GasToken:       core.StripChainPrefix("eth:0x2222222222222222222222222222222222222222"),
		RefundReceiver: "0x3333333333333333333333333333333333333333", Nonce: 14,
	}, *parentCall)

	check := oneJSONCheck(t, result)
	if _, ok := check["decode"]; ok {
		t.Fatal("contract check still exposes rendered decode text")
	}
	if _, ok := check["params"]; ok {
		t.Fatal("contract check still exposes rendered parameter text")
	}
	assertNoJSONNumbers(t, check, "contractChecks[0]")

	call := requireMap(t, check["call"], "call")
	assertFields(t, call, map[string]any{
		"functionName": "aggregate3", "signature": core.Aggregate3Sig,
		"target": core.Multicall3Delegatecall, "targetLabel": "MULTICALL3 DELEGATECALL",
		"operation": "DELEGATECALL",
	})
	nested := requireSlice(t, call["calls"], "call.calls")
	if len(nested) != 2 {
		t.Fatalf("len(call.calls) = %d, want 2", len(nested))
	}

	tupleArgs := requireSlice(t, requireMap(t, nested[0], "call.calls[0]")["arguments"], "tuple arguments")
	config := requireMap(t, tupleArgs[0], "config argument")
	assertFields(t, config, map[string]any{"name": "config", "type": "(uint256,uint256)"})
	tupleValues := requireSlice(t, config["value"], "config value")
	assertFields(t, requireMap(t, tupleValues[0], "config.maxSequencerDrift"), map[string]any{
		"name": "maxSequencerDrift", "type": "uint256", "value": testLargeInt,
	})
	assertFields(t, requireMap(t, tupleValues[1], "config.sequencerWindowSize"), map[string]any{
		"name": "sequencerWindowSize", "type": "uint256", "value": "1200",
	})

	arrayArgs := requireSlice(t, requireMap(t, nested[1], "call.calls[1]")["arguments"], "array arguments")
	values := requireMap(t, arrayArgs[1], "values argument")
	assertFields(t, values, map[string]any{"name": "values", "type": "uint256[]"})
	if got := requireSlice(t, values["value"], "values value"); !reflect.DeepEqual(got, []any{testLargeInt, "2"}) {
		t.Fatalf("values = %#v, want exact decimal strings", got)
	}

	safeFields := requireMap(t, check["safeFields"], "safeFields")
	assertFields(t, safeFields, map[string]any{
		"to": core.Multicall3Delegatecall, "value": testLargeInt, "data": "0x1234",
		"operation": "1", "safeTxGas": "11", "baseGas": "12", "gasPrice": "13",
		"gasToken":       "0x2222222222222222222222222222222222222222",
		"refundReceiver": "0x3333333333333333333333333333333333333333", "nonce": "14",
	})
}

func TestContractChecksDescribeEmptyCalldata(t *testing.T) {
	for _, tc := range []struct {
		name         string
		value        string
		functionName string
		wantArgs     int
	}{
		{name: "native transfer", value: testLargeInt, functionName: "Send native ETH", wantArgs: 2},
		{name: "zero value", value: "0", functionName: "No calldata", wantArgs: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx := core.SafeTransaction{
				Safe: testSafe, SafeVersion: "1.4.1", Chain: 1, To: testRecipient,
				Value: mustBigInt(t, tc.value), Data: "0x", GasToken: common.Address{}.Hex(),
				RefundReceiver: common.Address{}.Hex(),
			}
			result := testResult(tx, core.CallData{Target: testRecipient, FunctionName: "unknown", RawData: "0x"})
			call := requireMap(t, oneJSONCheck(t, result)["call"], "call")
			assertFields(t, call, map[string]any{
				"functionName": tc.functionName, "target": testRecipient, "operation": "CALL",
			})
			args := requireSlice(t, call["arguments"], "call.arguments")
			if len(args) != tc.wantArgs {
				t.Fatalf("len(call.arguments) = %d, want %d", len(args), tc.wantArgs)
			}
			if tc.wantArgs != 0 {
				assertFields(t, requireMap(t, args[0], "recipient argument"), map[string]any{
					"name": "recipient", "type": "address", "value": testRecipient,
				})
				assertFields(t, requireMap(t, args[1], "value argument"), map[string]any{
					"name": "value", "type": "uint256", "value": testLargeInt,
				})
			}
		})
	}
}

func TestStructuredCallIncludesExactValueForKnownFunction(t *testing.T) {
	call := parsedCall(t, "setOwner(address)", testRecipient, common.HexToAddress(testSafe))
	view, err := structuredCall(*call, mustBigInt(t, testLargeInt), 0)
	if err != nil {
		t.Fatalf("structuredCall: %v", err)
	}
	assertFields(t, view, map[string]any{
		"functionName": "setOwner", "value": testLargeInt,
	})
}

func TestContractChecksDescribeUnknownCall(t *testing.T) {
	raw := "0xdeadbeef00000001"
	tx := core.SafeTransaction{
		Safe: testSafe, SafeVersion: "1.4.1", Chain: 1, To: testRecipient,
		Value: big.NewInt(0), Data: raw, GasToken: common.Address{}.Hex(), RefundReceiver: common.Address{}.Hex(),
	}
	result := testResult(tx, core.CallData{Target: testRecipient, FunctionName: "unknown", RawData: raw})
	call := requireMap(t, oneJSONCheck(t, result)["call"], "call")
	assertFields(t, call, map[string]any{
		"functionName": "Unknown function", "target": testRecipient, "operation": "CALL",
		"selector": "0xdeadbeef", "rawCalldata": raw,
	})
	if _, ok := call["signature"]; ok {
		t.Fatal("unknown call unexpectedly has a signature")
	}
	if args := requireSlice(t, call["arguments"], "call.arguments"); len(args) != 0 {
		t.Fatalf("len(call.arguments) = %d, want 0", len(args))
	}
}

func TestContractChecksGiveShortUnknownCallsAnIdentity(t *testing.T) {
	for _, raw := range []string{"0x12", "0x1234", "0x123456"} {
		t.Run(raw, func(t *testing.T) {
			tx := core.SafeTransaction{
				Safe: testSafe, SafeVersion: "1.4.1", Chain: 1, To: testRecipient,
				Value: big.NewInt(0), Data: raw, GasToken: common.Address{}.Hex(), RefundReceiver: common.Address{}.Hex(),
			}
			result := testResult(tx, core.CallData{Target: testRecipient, FunctionName: "unknown", RawData: raw})
			call := requireMap(t, oneJSONCheck(t, result)["call"], "call")
			assertFields(t, call, map[string]any{
				"functionName": "Unknown function", "selector": raw, "rawCalldata": raw,
			})
		})
	}
}

func TestContractChecksTreatMalformedKnownCallAsUnknown(t *testing.T) {
	raw := "0x13af4035" // setOwner(address) selector with its address argument missing.
	tx := core.SafeTransaction{
		Safe: testSafe, SafeVersion: "1.4.1", Chain: 1, To: testRecipient,
		Value: big.NewInt(0), Data: raw, GasToken: common.Address{}.Hex(), RefundReceiver: common.Address{}.Hex(),
	}
	parsed, err := core.ParseTransactionData(tx.To, tx.Data, core.MainnetChainID, core.VerifyOptions{})
	if err != nil {
		t.Fatalf("ParseTransactionData: %v", err)
	}
	call := requireMap(t, oneJSONCheck(t, testResult(tx, *parsed))["call"], "call")
	assertFields(t, call, map[string]any{
		"functionName": "Unknown function", "target": testRecipient, "operation": "CALL",
		"selector": "0x13af4035", "rawCalldata": raw,
	})
	if _, ok := call["signature"]; ok {
		t.Fatal("malformed known call unexpectedly has a trusted signature")
	}
	if args := requireSlice(t, call["arguments"], "call.arguments"); len(args) != 0 {
		t.Fatalf("len(call.arguments) = %d, want 0", len(args))
	}
}

func TestContractChecksDoNotLeakPresentationMetadataIntoRawResult(t *testing.T) {
	known := parsedCall(t, "setOwner(address)", testRecipient, common.HexToAddress(testSafe))
	tx := core.SafeTransaction{
		Safe: testSafe, SafeVersion: "1.4.1", Chain: 1, To: testRecipient,
		Value: big.NewInt(0), Data: known.Calldata, GasToken: common.Address{}.Hex(), RefundReceiver: common.Address{}.Hex(),
	}
	result, err := core.VerifyTransaction(tx, core.VerifyOptions{})
	if err != nil {
		t.Fatalf("VerifyTransaction: %v", err)
	}
	before, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result before presentation: %v", err)
	}
	if strings.Contains(string(before), `"functionData"`) || strings.Contains(string(before), `"calldata"`) {
		t.Fatalf("raw result leaked presentation metadata: %s", before)
	}
	if _, err := contractChecks(result); err != nil {
		t.Fatalf("contractChecks: %v", err)
	}
	after, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result after presentation: %v", err)
	}
	if string(after) != string(before) {
		t.Fatalf("presentation mutated raw result\nbefore: %s\nafter:  %s", before, after)
	}
}

func TestExactABIValueCoversKnownShapes(t *testing.T) {
	bytes32 := [32]byte{0xde, 0xad}
	function := [24]byte{0xbe, 0xef}
	tupleType := mustABIType(t, "tuple", []abi.ArgumentMarshaling{
		{Name: "amount", Type: "uint256"},
		{Name: "recipient", Type: "address"},
	})
	nestedTupleArrayType := mustABIType(t, "tuple[]", []abi.ArgumentMarshaling{
		{Name: "config", Type: "tuple", Components: []abi.ArgumentMarshaling{
			{Name: "delay", Type: "uint64"},
			{Name: "enabled", Type: "bool"},
		}},
		{Name: "payloads", Type: "bytes[]"},
	})
	type tupleValue struct {
		Amount    *big.Int
		Recipient common.Address
	}
	type nestedConfig struct {
		Delay   uint64
		Enabled bool
	}
	type nestedTupleValue struct {
		Config   nestedConfig
		Payloads [][]byte
	}

	tests := []struct {
		name  string
		typ   abi.Type
		value any
		want  any
	}{
		{"uint8", mustABIType(t, "uint8", nil), uint8(7), "7"},
		{"uint32", mustABIType(t, "uint32", nil), uint32(32), "32"},
		{"uint64", mustABIType(t, "uint64", nil), uint64(64), "64"},
		{"uint256", mustABIType(t, "uint256", nil), mustBigInt(t, testLargeInt), testLargeInt},
		{"int8", mustABIType(t, "int8", nil), int8(-7), "-7"},
		{"int32", mustABIType(t, "int32", nil), int32(-32), "-32"},
		{"int64", mustABIType(t, "int64", nil), int64(-64), "-64"},
		{"int256", mustABIType(t, "int256", nil), big.NewInt(-256), "-256"},
		{"address", mustABIType(t, "address", nil), common.HexToAddress(testRecipient), testRecipient},
		{"bool", mustABIType(t, "bool", nil), true, true},
		{"bytes", mustABIType(t, "bytes", nil), []byte{0xde, 0xad}, "0xdead"},
		{"bytes32", mustABIType(t, "bytes32", nil), bytes32, "0xdead" + strings.Repeat("00", 30)},
		{"string", mustABIType(t, "string", nil), "hello", "hello"},
		{"function", mustABIType(t, "function", nil), function, "0xbeef" + strings.Repeat("00", 22)},
		{"slice", mustABIType(t, "address[]", nil), []common.Address{common.HexToAddress(testRecipient)}, []any{testRecipient}},
		{"array", mustABIType(t, "uint32[2]", nil), [2]uint32{1, 2}, []any{"1", "2"}},
		{"tuple", tupleType, tupleValue{mustBigInt(t, testLargeInt), common.HexToAddress(testRecipient)}, []any{
			map[string]any{"name": "amount", "type": "uint256", "value": testLargeInt},
			map[string]any{"name": "recipient", "type": "address", "value": testRecipient},
		}},
		{"nested tuple slice", nestedTupleArrayType, []nestedTupleValue{{
			Config: nestedConfig{Delay: 12, Enabled: true}, Payloads: [][]byte{{0xab}, {0xcd}},
		}}, []any{[]any{
			map[string]any{"name": "config", "type": "(uint64,bool)", "value": []any{
				map[string]any{"name": "delay", "type": "uint64", "value": "12"},
				map[string]any{"name": "enabled", "type": "bool", "value": true},
			}},
			map[string]any{"name": "payloads", "type": "bytes[]", "value": []any{"0xab", "0xcd"}},
		}}},
	}

	covered := make(map[string]bool)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := exactABIValue(tc.typ, tc.value)
			if err != nil {
				t.Fatalf("exactABIValue: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("exactABIValue(%s) = %#v, want %#v", tc.typ.String(), got, tc.want)
			}
		})
		collectABIShapes(tc.typ, covered)
	}

	for _, function := range core.KnownFunctions {
		for _, input := range function.ABI.Inputs {
			assertABIShapeCovered(t, input.Type, covered, function.Signature)
		}
	}

	if _, err := exactABIValue(abi.Type{T: abi.HashTy}, [32]byte{}); err == nil {
		t.Fatal("exactABIValue accepted unsupported HashTy")
	}
}

func parsedCall(t *testing.T, signature, target string, values ...any) *core.CallData {
	t.Helper()
	selector := hex.EncodeToString(crypto.Keccak256([]byte(signature))[:4])
	info, ok := core.KnownFunctions[selector]
	if !ok {
		t.Fatalf("known function %s is missing", signature)
	}
	args, err := info.ABI.Inputs.Pack(values...)
	if err != nil {
		t.Fatalf("pack %s: %v", signature, err)
	}
	calldata := "0x" + selector + hex.EncodeToString(args)
	call, err := core.ParseTransactionData(target, calldata, core.MainnetChainID, core.VerifyOptions{})
	if err != nil {
		t.Fatalf("parse %s: %v", signature, err)
	}
	return call
}

func mustABIType(t *testing.T, typeName string, components []abi.ArgumentMarshaling) abi.Type {
	t.Helper()
	typ, err := abi.NewType(typeName, "", components)
	if err != nil {
		t.Fatalf("abi.NewType(%q): %v", typeName, err)
	}
	return typ
}

func collectABIShapes(typ abi.Type, covered map[string]bool) {
	covered[abiShape(typ)] = true
	if typ.Elem != nil {
		collectABIShapes(*typ.Elem, covered)
	}
	for _, elem := range typ.TupleElems {
		collectABIShapes(*elem, covered)
	}
}

func assertABIShapeCovered(t *testing.T, typ abi.Type, covered map[string]bool, signature string) {
	t.Helper()
	if !covered[abiShape(typ)] {
		t.Errorf("%s uses uncovered ABI shape %s", signature, abiShape(typ))
	}
	if typ.Elem != nil {
		assertABIShapeCovered(t, *typ.Elem, covered, signature)
	}
	for _, elem := range typ.TupleElems {
		assertABIShapeCovered(t, *elem, covered, signature)
	}
}

func abiShape(typ abi.Type) string {
	switch typ.T {
	case abi.IntTy:
		return "int" + strconv.Itoa(typ.Size)
	case abi.UintTy:
		return "uint" + strconv.Itoa(typ.Size)
	case abi.BoolTy:
		return "bool"
	case abi.StringTy:
		return "string"
	case abi.SliceTy:
		return "slice"
	case abi.ArrayTy:
		return "array"
	case abi.TupleTy:
		return "tuple"
	case abi.AddressTy:
		return "address"
	case abi.FixedBytesTy:
		return "bytes" + strconv.Itoa(typ.Size)
	case abi.BytesTy:
		return "bytes"
	case abi.FunctionTy:
		return "function"
	default:
		return "unsupported:" + strconv.Itoa(int(typ.T))
	}
}

func testResult(tx core.SafeTransaction, call core.CallData) *core.VerificationResult {
	tx.Call = call
	return &core.VerificationResult{
		Transaction: tx, DomainHash: "0x" + strings.Repeat("1", 64),
		MessageHash: "0x" + strings.Repeat("2", 64), ApproveHash: "0x" + strings.Repeat("3", 64),
		Call: call,
	}
}

func oneJSONCheck(t *testing.T, result *core.VerificationResult) map[string]any {
	t.Helper()
	checks, err := contractChecks(result)
	if err != nil {
		t.Fatalf("contractChecks: %v", err)
	}
	if len(checks) != 1 {
		t.Fatalf("len(contractChecks) = %d, want 1", len(checks))
	}
	raw, err := json.Marshal(checks[0])
	if err != nil {
		t.Fatalf("marshal check: %v", err)
	}
	var check map[string]any
	if err := json.Unmarshal(raw, &check); err != nil {
		t.Fatalf("unmarshal check: %v", err)
	}
	return check
}

func mustBigInt(t *testing.T, value string) *big.Int {
	t.Helper()
	result, ok := new(big.Int).SetString(value, 10)
	if !ok {
		t.Fatalf("invalid test integer %q", value)
	}
	return result
}

func requireMap(t *testing.T, value any, path string) map[string]any {
	t.Helper()
	result, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("%s = %T, want object", path, value)
	}
	return result
}

func requireSlice(t *testing.T, value any, path string) []any {
	t.Helper()
	result, ok := value.([]any)
	if !ok {
		t.Fatalf("%s = %T, want array", path, value)
	}
	return result
}

func assertFields(t *testing.T, object map[string]any, want map[string]any) {
	t.Helper()
	for field, expected := range want {
		if got := object[field]; got != expected {
			t.Errorf("%s = %#v, want %#v", field, got, expected)
		}
	}
}

func assertNoJSONNumbers(t *testing.T, value any, path string) {
	t.Helper()
	switch value := value.(type) {
	case float64:
		t.Errorf("%s crossed the JSON boundary as number %v", path, value)
	case []any:
		for _, item := range value {
			assertNoJSONNumbers(t, item, path+"[]")
		}
	case map[string]any:
		for field, item := range value {
			assertNoJSONNumbers(t, item, path+"."+field)
		}
	}
}
