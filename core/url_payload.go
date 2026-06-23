package core

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// DecodeTransactionURL decodes an op-txverify URL payload. It accepts the
// legacy uncompressed tx parameter and the compressed txz parameter emitted by
// superchain-ops.
func DecodeTransactionURL(rawURL string) (*SafeTransaction, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}

	query := parsed.Query()
	var payload []byte
	if compressedParam := query.Get("txz"); compressedParam != "" {
		compressed, err := decodeBase64Payload(compressedParam)
		if err != nil {
			return nil, fmt.Errorf("invalid base64 txz parameter: %w", err)
		}
		payload, err = FastLZDecompress(compressed)
		if err != nil {
			return nil, fmt.Errorf("invalid compressed txz parameter: %w", err)
		}
	} else {
		txParam := query.Get("tx")
		if txParam == "" {
			return nil, fmt.Errorf("tx or txz parameter not found in url")
		}
		payload, err = decodeBase64Payload(txParam)
		if err != nil {
			return nil, fmt.Errorf("invalid base64 tx parameter: %w", err)
		}
	}

	var tx SafeTransaction
	if err := json.Unmarshal(payload, &tx); err != nil {
		return nil, fmt.Errorf("failed to parse transaction from url: %w", err)
	}
	return &tx, nil
}

func decodeBase64Payload(payload string) ([]byte, error) {
	stdPayload := strings.ReplaceAll(payload, " ", "+")
	attempts := []struct {
		encoding *base64.Encoding
		value    string
	}{
		{base64.RawURLEncoding, payload},
		{base64.URLEncoding, payload},
		{base64.StdEncoding, stdPayload},
		{base64.RawStdEncoding, stdPayload},
	}

	var lastErr error
	for _, attempt := range attempts {
		decoded, err := attempt.encoding.DecodeString(attempt.value)
		if err == nil {
			return decoded, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

// FastLZDecompress decompresses the FastLZ level-1 format produced by Solady's
// LibZip.flzCompress.
func FastLZDecompress(data []byte) ([]byte, error) {
	out := make([]byte, 0, len(data)*4)
	for i := 0; i < len(data); {
		control := data[i]
		tag := int(control >> 5)
		if tag == 0 {
			length := int(control) + 1
			i++
			if i+length > len(data) {
				return nil, fmt.Errorf("literal run exceeds input length")
			}
			out = append(out, data[i:i+length]...)
			i += length
			continue
		}

		var length int
		var distance int
		if tag < 7 {
			if i+1 >= len(data) {
				return nil, fmt.Errorf("truncated short match")
			}
			length = tag + 2
			distance = (int(control&0x1f) << 8) | int(data[i+1])
			i += 2
		} else {
			if i+2 >= len(data) {
				return nil, fmt.Errorf("truncated long match")
			}
			length = int(data[i+1]) + 9
			distance = (int(control&0x1f) << 8) | int(data[i+2])
			i += 3
		}

		ref := len(out) - distance - 1
		if ref < 0 {
			return nil, fmt.Errorf("invalid match distance")
		}
		for j := 0; j < length; j++ {
			if ref+j >= len(out) {
				return nil, fmt.Errorf("match exceeds output length")
			}
			out = append(out, out[ref+j])
		}
	}
	return out, nil
}
