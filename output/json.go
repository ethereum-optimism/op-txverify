package output

import (
	"encoding/json"
	"io"

	"github.com/ethereum-optimism/op-verify/core"
)

// FormatJSON outputs the verification result as JSON
func FormatJSON(result *core.VerificationResult, w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}
