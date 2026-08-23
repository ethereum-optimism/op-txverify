package core

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

type historyFixture struct {
	Signature  string                     `json:"signature"`
	Chain      uint64                     `json:"chain"`
	Safe       string                     `json:"safe"`
	Target     string                     `json:"target"`
	SafeTxHash string                     `json:"safeTxHash"`
	Nonce      int                        `json:"nonce"`
	SourceURL  string                     `json:"sourceURL"`
	Calldata   string                     `json:"calldata"`
	Expected   map[string]json.RawMessage `json:"expected"`
}

func TestHistoryFixturesDecodeLiteralCalldata(t *testing.T) {
	data, err := os.ReadFile("testdata/history_calls.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []historyFixture
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	if len(fixtures) != 37 {
		t.Fatalf("fixture count = %d, want 37", len(fixtures))
	}

	seen := make(map[string]bool, len(fixtures))
	for _, fixture := range fixtures {
		t.Run(fixture.Signature, func(t *testing.T) {
			if seen[fixture.Signature] {
				t.Fatalf("duplicate fixture signature %s", fixture.Signature)
			}
			seen[fixture.Signature] = true
			if fixture.Chain == 0 || fixture.Safe == "" || fixture.Target == "" || fixture.Nonce < 0 || len(fixture.SafeTxHash) != 66 || !strings.HasPrefix(fixture.SourceURL, "https://safe-transaction-") {
				t.Fatalf("incomplete provenance: %+v", fixture)
			}

			call, err := ParseTransactionData(fixture.Target, fixture.Calldata, fixture.Chain, VerifyOptions{})
			if err != nil {
				t.Fatalf("ParseTransactionData() error = %v", err)
			}
			selector := strings.TrimPrefix(fixture.Calldata, "0x")[:8]
			function, ok := KnownFunctions[selector]
			if !ok || function.Signature != fixture.Signature || call.FunctionName == "unknown" {
				t.Fatalf("selector %s decoded as %#v, want %s", selector, function, fixture.Signature)
			}
			actual, ok := call.ParsedData.(map[string]interface{})
			if !ok {
				t.Fatalf("parsed data type = %T, want map", call.ParsedData)
			}
			actualJSON, err := json.Marshal(actual)
			if err != nil {
				t.Fatal(err)
			}
			var actualFields map[string]json.RawMessage
			if err := json.Unmarshal(actualJSON, &actualFields); err != nil {
				t.Fatal(err)
			}
			for name, want := range fixture.Expected {
				got, ok := actualFields[name]
				if !ok || compactJSON(got) != compactJSON(want) {
					t.Fatalf("decoded %s = %s, want %s", name, got, want)
				}
			}
		})
	}
}

func TestHistoryRecordProvenanceIsCommitted(t *testing.T) {
	for _, record := range historyABIRecords {
		if record.Signature == "withdraw(uint256,uint256,address)" {
			continue
		}
		if record.Source != "Safe history fixture: core/testdata/history_calls.json" {
			t.Fatalf("%s source = %q, want committed history fixture", record.Signature, record.Source)
		}
	}
}

func compactJSON(raw json.RawMessage) string {
	var value interface{}
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return "invalid JSON"
	}
	compact, err := json.Marshal(value)
	if err != nil {
		return "invalid JSON"
	}
	return string(compact)
}

func TestBuildKnownFunctionsRejectsConflictingSelectors(t *testing.T) {
	records := []KnownABIRecord{
		{Signature: "burn(uint256)", ParameterNames: []string{"amount"}, Source: "test", ABIJSON: `[{"inputs":[{"name":"amount","type":"uint256"}],"name":"burn","type":"function"}]`},
		{Signature: "collate_propagate_storage(bytes16)", ParameterNames: []string{"data"}, Source: "test", ABIJSON: `[{"inputs":[{"name":"data","type":"bytes16"}],"name":"collate_propagate_storage","type":"function"}]`},
	}
	_, err := BuildKnownFunctions(records)
	if err == nil || !strings.Contains(err.Error(), "conflicts") {
		t.Fatalf("BuildKnownFunctions() error = %v, want selector conflict", err)
	}
}

func TestEtherFiSpokeLabels(t *testing.T) {
	for address, want := range map[string]string{
		"0xdffcC3536D932eb51Df51a7F5FA407c4270d5308": "ETHERFI SPOKE (PROXY)",
		"0xA1f75D801633a1941cae6670352d627884dC3b68": "ETHERFI SPOKE (IMPLEMENTATION)",
	} {
		info, ok := GetKnownContract(address, OPMainnetChainID)
		if !ok || info.Name != want || info.Decimals != 0 {
			t.Fatalf("GetKnownContract(%s) = %+v, %t; want %q with 0 decimals", address, info, ok, want)
		}
	}
}
