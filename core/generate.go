package core

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

// SafeAPITransaction is one queued transaction as returned by the Safe Transaction Service.
type SafeAPITransaction struct {
	SafeTxHash     string      `json:"safeTxHash"`
	IsExecuted     bool        `json:"isExecuted"`
	To             string      `json:"to"`
	Value          string      `json:"value"`
	Data           string      `json:"data"`
	Operation      int         `json:"operation"`
	SafeTxGas      int         `json:"safeTxGas"`
	BaseGas        int         `json:"baseGas"`
	GasPrice       string      `json:"gasPrice"`
	GasToken       string      `json:"gasToken"`
	RefundReceiver string      `json:"refundReceiver"`
	DataDecoded    interface{} `json:"dataDecoded"`
}

// APIResponse represents the response from the Safe API
type APIResponse struct {
	Count   int                  `json:"count"`
	Results []SafeAPITransaction `json:"results"`
}

// selectQueuedTransaction picks the single transaction queued at a nonce. A Safe can hold several
// competing proposals at one nonce and anyone able to queue can add one, so returning the first
// result would silently verify a transaction the signer never intended.
func selectQueuedTransaction(resp APIResponse, safeAddress string, nonce uint64) (*SafeAPITransaction, error) {
	if resp.Count == 0 || len(resp.Results) == 0 {
		return nil, fmt.Errorf("no transaction found for safe %s with nonce %d", safeAddress, nonce)
	}

	if resp.Count > 1 || len(resp.Results) > 1 {
		candidates := make([]string, 0, len(resp.Results))
		for _, result := range resp.Results {
			state := "pending"
			if result.IsExecuted {
				state = "executed"
			}
			candidates = append(candidates, fmt.Sprintf("%s (%s)", result.SafeTxHash, state))
		}
		// There is no --safe-tx-hash flag, so point at the paths that do take a hash directly.
		return nil, fmt.Errorf(
			"safe %s has %d transactions at nonce %d: %s; "+
				"a nonce lookup cannot tell which one you mean - verify it by hash instead, "+
				"via op-txverify.optimism.io or `op-txverify offline --tx <json>`",
			safeAddress, resp.Count, nonce, strings.Join(candidates, ", "),
		)
	}

	return &resp.Results[0], nil
}

// SafeInfoResponse represents the response from the Safe info API
type SafeInfoResponse struct {
	Version string `json:"version"`
}

// fetchSafeVersion fetches the Safe version for a given safe address
func fetchSafeVersion(apiURL, safeAddress string) (string, error) {
	endpoint := fmt.Sprintf("%s/api/v1/safes/%s/", apiURL, safeAddress)

	resp, err := http.Get(endpoint)
	if err != nil {
		return "", fmt.Errorf("error fetching safe version: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch safe version: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading safe info response body: %w", err)
	}

	var safeInfo SafeInfoResponse
	if err := json.Unmarshal(body, &safeInfo); err != nil {
		return "", fmt.Errorf("error parsing safe info response: %w", err)
	}

	return safeInfo.Version, nil
}

// GenerateTransaction fetches transaction data from the Safe API and returns a SafeTransaction
func GenerateTransaction(network string, safeAddress string, nonce uint64) (*SafeTransaction, error) {
	// Get network info
	apiURL, chainID, err := getNetworkInfo(network)
	if err != nil {
		return nil, err
	}

	return generateTransaction(apiURL, chainID, safeAddress, nonce)
}

// generateTransaction is GenerateTransaction with the Safe API base URL supplied, so tests
// can serve the responses it assembles a transaction from.
func generateTransaction(apiURL string, chainID uint64, safeAddress string, nonce uint64) (*SafeTransaction, error) {
	// Normalize safe address
	safeAddress = common.HexToAddress(safeAddress).Hex()

	// Fetch Safe version
	safeVersion, err := fetchSafeVersion(apiURL, safeAddress)
	if err != nil {
		return nil, fmt.Errorf("error fetching safe version: %w", err)
	}

	// Construct API endpoint
	endpoint := fmt.Sprintf("%s/api/v1/safes/%s/multisig-transactions/?nonce=%d", apiURL, safeAddress, nonce)

	// Make HTTP request
	resp, err := http.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("error fetching transaction data: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %s", resp.Status)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	// Parse response
	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("error parsing API response: %w", err)
	}

	selected, err := selectQueuedTransaction(apiResp, safeAddress, nonce)
	if err != nil {
		return nil, err
	}
	tx := *selected

	// Without it there is nothing to hold the fields to, and the hashes below would only restate
	// whatever the response said.
	if err := validateHash("safeTxHash", tx.SafeTxHash); err != nil {
		return nil, fmt.Errorf("the Safe transaction service response cannot be bound to a hash: %w", err)
	}

	// The hashed transaction is the inner one for a nested approval, so it carries the
	// inner Safe's nonce; the requested nonce belongs to the outer Safe.
	txNonce := int(nonce)

	var nested *Nested
	content := tx

	// Check if this is an approveHash transaction
	if tx.Data != "" && strings.HasPrefix(tx.Data, "0xd4d9bdcd") {
		// Extract the hash from the data (skip first 10 chars for function signature, take next 64)
		// A truncated argument used to fall through to a plain transaction with no child block.
		if len(tx.Data) < 74 {
			return nil, fmt.Errorf("approveHash calldata %q is too short to carry a 32-byte hash", tx.Data)
		}
		innerHash := "0x" + tx.Data[10:74]

		// Fetch the inner transaction using v2 API
		innerEndpoint := fmt.Sprintf("%s/api/v2/multisig-transactions/%s/", apiURL, innerHash)
		innerResp, err := http.Get(innerEndpoint)
		if err != nil {
			return nil, fmt.Errorf("error fetching inner transaction data: %w", err)
		}
		defer innerResp.Body.Close()

		// Anything other than 200 used to fall through this block, verifying an approveHash
		// call with no child transaction attached and nothing saying the child was never read.
		if innerResp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf(
				"the Safe transaction service answered %s for the approved transaction %s, so what this approves is unknown",
				innerResp.Status, innerHash)
		}

		innerBody, err := io.ReadAll(innerResp.Body)
		if err != nil {
			return nil, fmt.Errorf("error reading inner transaction response body: %w", err)
		}

		// Parse inner transaction as a single transaction (not wrapped in APIResponse)
		var innerTx struct {
			SafeTxHash     string `json:"safeTxHash"`
			To             string `json:"to"`
			Value          string `json:"value"`
			Data           string `json:"data"`
			Operation      int    `json:"operation"`
			SafeTxGas      string `json:"safeTxGas"` // API returns as string
			BaseGas        string `json:"baseGas"`   // API returns as string
			GasPrice       string `json:"gasPrice"`
			GasToken       string `json:"gasToken"`
			RefundReceiver string `json:"refundReceiver"`
			Nonce          string `json:"nonce"`
			Safe           string `json:"safe"`
		}

		if err := json.Unmarshal(innerBody, &innerTx); err != nil {
			return nil, fmt.Errorf("error parsing inner transaction response: %w", err)
		}

		// Convert string values to integers for inner transaction
		innerSafeTxGas, err := parseAPIUint("inner transaction safeTxGas", innerTx.SafeTxGas)
		if err != nil {
			return nil, err
		}
		innerBaseGas, err := parseAPIUint("inner transaction baseGas", innerTx.BaseGas)
		if err != nil {
			return nil, err
		}
		txNonce, err = parseAPIUint("inner transaction nonce", innerTx.Nonce)
		if err != nil {
			return nil, err
		}
		outerGasPrice, err := parseAPIUint("gasPrice", tx.GasPrice)
		if err != nil {
			return nil, err
		}

		// The hash the parent's calldata approves is the only statement of which
		// transaction the child is meant to be, so the response has to answer to it.
		if !strings.EqualFold(innerTx.SafeTxHash, innerHash) {
			return nil, fmt.Errorf(
				"asked for the approved transaction %s and the service returned %q instead",
				innerHash, innerTx.SafeTxHash)
		}

		// Create nested data from outer transaction (using OUTER safe's info)
		nested = &Nested{
			Safe:           safeAddress,
			SafeVersion:    safeVersion,
			SafeTxHash:     tx.SafeTxHash,
			Nonce:          int(nonce),
			Data:           tx.Data,
			Operation:      tx.Operation,
			To:             tx.To,
			SafeTxGas:      tx.SafeTxGas,
			BaseGas:        tx.BaseGas,
			GasPrice:       outerGasPrice,
			GasToken:       tx.GasToken,
			RefundReceiver: tx.RefundReceiver,
		}

		// Use inner transaction data as the main content
		content.To = innerTx.To
		content.Value = innerTx.Value
		content.Data = innerTx.Data
		content.Operation = innerTx.Operation
		content.SafeTxGas = innerSafeTxGas
		content.BaseGas = innerBaseGas
		content.GasPrice = innerTx.GasPrice
		content.GasToken = innerTx.GasToken
		content.RefundReceiver = innerTx.RefundReceiver
		content.SafeTxHash = innerTx.SafeTxHash

		// For the main transaction, we need the INNER safe's info
		safeAddress = innerTx.Safe // Update to use inner safe address

		// Fetch the inner safe's version for the main transaction
		safeVersion, err = fetchSafeVersion(apiURL, innerTx.Safe)
		if err != nil {
			return nil, fmt.Errorf("error fetching inner safe version: %w", err)
		}
	}

	// Value can exceed 64-bit range, so it is carried as a big.Int.
	valueBig, err := ParseWei(content.Value)
	if err != nil {
		return nil, err
	}

	gasPrice, err := parseAPIUint("gasPrice", content.GasPrice)
	if err != nil {
		return nil, err
	}

	// Create SafeTransaction
	safeTx := &SafeTransaction{
		Safe:           safeAddress,
		SafeVersion:    safeVersion,
		SafeTxHash:     content.SafeTxHash,
		Chain:          int(chainID),
		To:             content.To,
		Value:          valueBig,
		Data:           content.Data,
		Operation:      content.Operation,
		SafeTxGas:      content.SafeTxGas,
		BaseGas:        content.BaseGas,
		GasPrice:       gasPrice,
		GasToken:       content.GasToken,
		RefundReceiver: content.RefundReceiver,
		Nonce:          txNonce,
		Nested:         nested,
	}

	return safeTx, nil
}

// parseAPIUint parses one of the decimal integer strings the Safe API returns. fmt.Sscanf
// reports the leading digits of a malformed value as a plausible number instead of failing.
func parseAPIUint(field, value string) (int, error) {
	parsed, err := strconv.ParseUint(value, 10, 63)
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q: %w", field, value, err)
	}
	return int(parsed), nil
}

// getNetworkInfo returns the API URL and chain ID for a network
func getNetworkInfo(network string) (string, uint64, error) {
	network = strings.ToLower(network)

	var apiURL string
	var chainID uint64

	// The per-network safe-transaction-*.safe.global hosts now 308-redirect to api.safe.global, so
	// address it directly rather than relying on a cross-origin redirect.
	switch network {
	case "ethereum":
		apiURL = "https://api.safe.global/tx-service/eth"
		chainID = MainnetChainID
	case "op", "optimism":
		apiURL = "https://api.safe.global/tx-service/oeth"
		chainID = OPMainnetChainID
	case "base":
		apiURL = "https://api.safe.global/tx-service/base"
		chainID = BaseMainnetChainID
	case "sepolia":
		apiURL = "https://api.safe.global/tx-service/sep"
		chainID = SepoliaChainID
	default:
		return "", 0, fmt.Errorf("unsupported network: %s (must be ethereum, op, base, or sepolia)", network)
	}

	return apiURL, chainID, nil
}
