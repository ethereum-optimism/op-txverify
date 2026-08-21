package output

import (
	"bytes"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum-optimism/op-txverify/core"
	"github.com/fatih/color"
)

func TestFormatHash(t *testing.T) {
	if got := formatHash("0xabcdef"); got != "0xABCDEF" {
		t.Fatalf("formatHash = %q, want 0xABCDEF", got)
	}
	if got := formatHash("abcdef"); got != "ABCDEF" {
		t.Fatalf("formatHash no prefix stays upper = %q", got)
	}
}

func TestFormatTerminal_ShowsHashedGasAndRefundFields(t *testing.T) {
	color.NoColor = true

	result := &core.VerificationResult{
		Transaction: core.SafeTransaction{
			Safe:           "0x847B5c174615B1B7fDF770882256e2D3E95b9D92",
			SafeVersion:    "1.3.0",
			Chain:          1,
			To:             "0x4200000000000000000000000000000000000016",
			Value:          big.NewInt(0),
			Data:           "0x",
			SafeTxGas:      111,
			BaseGas:        222,
			GasPrice:       333,
			GasToken:       "0x0000000000000000000000000000000000001234",
			RefundReceiver: "0x0000000000000000000000000000000000005678",
		},
	}

	var buf bytes.Buffer
	if err := FormatTerminal(result, &buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wants := []string{
		"Safe Tx Gas: 111",
		"Base Gas: 222",
		"Gas Price: 333",
		"Gas Token: 0x0000000000000000000000000000000000001234",
		"Refund Receiver: 0x0000000000000000000000000000000000005678",
		"THIS TRANSACTION PAYS A GAS REFUND FROM THE SAFE",
	}
	for _, want := range wants {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("output is missing %q", want)
		}
	}
}

func TestFormatTerminal_ShowsChildValueOperationAndRefund(t *testing.T) {
	color.NoColor = true

	child := core.SafeTransaction{
		Safe:           "0x847B5c174615B1B7fDF770882256e2D3E95b9D92",
		SafeVersion:    "1.3.0",
		Chain:          1,
		To:             "0x4200000000000000000000000000000000000016",
		Value:          big.NewInt(1_000_000_000_000_000_000),
		Data:           "0x",
		Operation:      1,
		GasPrice:       12345,
		GasToken:       "0x0000000000000000000000000000000000001234",
		RefundReceiver: "0x0000000000000000000000000000000000000000",
		Nonce:          9,
	}
	parent := child
	parent.Safe = "0x5a0Aae59D09fccBdDb6C6CcEB07B7279367C3d2A"
	parent.Value = big.NewInt(0)
	parent.Operation = 0
	parent.GasPrice = 0
	parent.GasToken = ""

	result := &core.VerificationResult{
		Transaction: parent,
		NestedResult: &core.VerificationResult{
			Transaction: child,
			ApproveHash: "0xbefcc37ec0dd42e4ebe3c6389c4929047b958910f3cf37376174d1f43b882e9f",
		},
	}

	var buf bytes.Buffer
	if err := FormatTerminal(result, &buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	parentSection, childSection, found := strings.Cut(out, "CHILD TRANSACTION SUMMARY")
	if !found {
		t.Fatal("output has no child transaction summary")
	}

	for _, want := range []string{
		"ETH Value: 1.00",
		"Operation: DELEGATECALL",
		"Gas Price: 12345",
		"Gas Token: 0x0000000000000000000000000000000000001234",
		"THIS TRANSACTION PAYS A GAS REFUND FROM THE SAFE",
	} {
		if !strings.Contains(childSection, want) {
			t.Errorf("child summary is missing %q", want)
		}
	}

	for _, want := range []string{"ETH Value: 0.00", "Operation: CALL", "Gas Price: 0"} {
		if !strings.Contains(parentSection, want) {
			t.Errorf("parent summary is missing %q", want)
		}
	}
	if strings.Contains(parentSection, "PAYS A GAS REFUND") {
		t.Error("parent summary warns about the child's gas refund")
	}
}
