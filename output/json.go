package output

import (
	"encoding/json"
	"fmt"
	"io"
)

// FormatJSON outputs the verification result as JSON
func FormatJSON(data interface{}, writer io.Writer) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("error formatting JSON: %w", err)
	}
	_, err = writer.Write(jsonData)
	return err
}
