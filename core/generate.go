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

// GenerateTransaction fetches transaction data from the Safe API and returns a SafeTransaction
func GenerateTransaction(network string, safeAddress string, nonce uint64) (*SafeTransaction, error) {
	// Get network info
	apiURL, chainID, err := getNetworkInfo(network)
	if err != nil {
		return nil, err
	}

	// Normalize safe address
	safeAddress = common.HexToAddress(safeAddress).Hex()

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

	// Convert string values to appropriate types
	var value, gasPrice int
	fmt.Sscanf(tx.Value, "%d", &value)
	fmt.Sscanf(tx.GasPrice, "%d", &gasPrice)

	// Create SafeTransaction
	safeTx := &SafeTransaction{
		Safe:           safeAddress,
		Chain:          int(chainID),
		To:             tx.To,
		Value:          value,
		Data:           tx.Data,
		Operation:      tx.Operation,
		SafeTxGas:      tx.SafeTxGas,
		BaseGas:        tx.BaseGas,
		GasPrice:       gasPrice,
		GasToken:       tx.GasToken,
		RefundReceiver: tx.RefundReceiver,
		Nonce:          int(nonce),
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
