package core

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

// APIResponse represents the response from the Safe API
type APIResponse struct {
	Count   int `json:"count"`
	Results []struct {
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
	} `json:"results"`
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

	// Check if transaction exists
	if apiResp.Count == 0 {
		return nil, fmt.Errorf("no transaction found for safe %s with nonce %d", safeAddress, nonce)
	}

	// Use the first transaction in the results
	tx := apiResp.Results[0]

	var nested *Nested
	content := tx

	// Check if this is an approveHash transaction
	if tx.Data != "" && strings.HasPrefix(tx.Data, "0xd4d9bdcd") {
		// Extract the hash from the data (skip first 10 chars for function signature, take next 64)
		if len(tx.Data) >= 74 {
			innerHash := "0x" + tx.Data[10:74]

			// Fetch the inner transaction using v2 API
			innerEndpoint := fmt.Sprintf("%s/api/v2/multisig-transactions/%s/", apiURL, innerHash)
			innerResp, err := http.Get(innerEndpoint)
			if err != nil {
				return nil, fmt.Errorf("error fetching inner transaction data: %w", err)
			}
			defer innerResp.Body.Close()

			if innerResp.StatusCode == http.StatusOK {
				innerBody, err := io.ReadAll(innerResp.Body)
				if err != nil {
					return nil, fmt.Errorf("error reading inner transaction response body: %w", err)
				}

				// Parse inner transaction as a single transaction (not wrapped in APIResponse)
				var innerTx struct {
					To             string `json:"to"`
					Value          string `json:"value"`
					Data           string `json:"data"`
					Operation      int    `json:"operation"`
					SafeTxGas      int    `json:"safeTxGas"`
					BaseGas        int    `json:"baseGas"`
					GasPrice       string `json:"gasPrice"`
					GasToken       string `json:"gasToken"`
					RefundReceiver string `json:"refundReceiver"`
					Nonce          int    `json:"nonce"`
					Safe           string `json:"safe"`
				}

				if err := json.Unmarshal(innerBody, &innerTx); err != nil {
					return nil, fmt.Errorf("error parsing inner transaction response: %w", err)
				}

				// Create nested data from outer transaction
				nested = &Nested{
					Safe:        tx.To, // The safe that the outer transaction is calling
					SafeVersion: safeVersion,
					Nonce:       int(nonce),
					Data:        tx.Data,
					Operation:   tx.Operation,
					To:          tx.To,
				}

				// Use inner transaction data as the main content
				content.To = innerTx.To
				content.Value = innerTx.Value
				content.Data = innerTx.Data
				content.Operation = innerTx.Operation
				content.SafeTxGas = innerTx.SafeTxGas
				content.BaseGas = innerTx.BaseGas
				content.GasPrice = innerTx.GasPrice
				content.GasToken = innerTx.GasToken
				content.RefundReceiver = innerTx.RefundReceiver
			}
		}
	}

	// Convert string values to appropriate types
	var value, gasPrice int
	fmt.Sscanf(content.Value, "%d", &value)
	fmt.Sscanf(content.GasPrice, "%d", &gasPrice)

	// Create SafeTransaction
	safeTx := &SafeTransaction{
		Safe:           safeAddress,
		SafeVersion:    safeVersion,
		Chain:          int(chainID),
		To:             content.To,
		Value:          value,
		Data:           content.Data,
		Operation:      content.Operation,
		SafeTxGas:      content.SafeTxGas,
		BaseGas:        content.BaseGas,
		GasPrice:       gasPrice,
		GasToken:       content.GasToken,
		RefundReceiver: content.RefundReceiver,
		Nonce:          int(nonce),
		Nested:         nested,
	}

	return safeTx, nil
}

// getNetworkInfo returns the API URL and chain ID for a network
func getNetworkInfo(network string) (string, uint64, error) {
	network = strings.ToLower(network)

	var apiURL string
	var chainID uint64

	switch network {
	case "ethereum":
		apiURL = "https://safe-transaction-mainnet.safe.global"
		chainID = MainnetChainID
	case "op", "optimism":
		apiURL = "https://safe-transaction-optimism.safe.global"
		chainID = OPMainnetChainID
	default:
		return "", 0, fmt.Errorf("unsupported network: %s (must be ethereum, op, or base)", network)
	}

	return apiURL, chainID, nil
}
