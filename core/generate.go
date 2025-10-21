package core

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
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
		// Approve-hash calldata must be exactly 74 characters:
		//  - 0x prefix (2)
		//  - 4-byte function selector (8 hex chars)
		//  - 32-byte transaction hash (64 hex chars)
		if len(tx.Data) != 74 {
			return nil, fmt.Errorf("invalid approveHash calldata length: got %d, want 74 (0x + 4-byte selector + 32-byte hash)", len(tx.Data))
		}

		// Extract the inner transaction hash (skip 0x + 8 selector chars => start at index 10)
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
			var innerSafeTxGas, innerBaseGas int
			fmt.Sscanf(innerTx.SafeTxGas, "%d", &innerSafeTxGas)
			fmt.Sscanf(innerTx.BaseGas, "%d", &innerBaseGas)

			// Create nested data from outer transaction (using OUTER safe's info)
			nested = &Nested{
				Safe:        safeAddress,
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
			content.SafeTxGas = innerSafeTxGas
			content.BaseGas = innerBaseGas
			content.GasPrice = innerTx.GasPrice
			content.GasToken = innerTx.GasToken
			content.RefundReceiver = innerTx.RefundReceiver

			// For the main transaction, we need the INNER safe's info
			safeAddress = innerTx.Safe // Update to use inner safe address

			// Fetch the inner safe's version for the main transaction
			safeVersion, err = fetchSafeVersion(apiURL, innerTx.Safe)
			if err != nil {
				return nil, fmt.Errorf("error fetching inner safe version: %w", err)
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

// chainIDToAPIURL maps chain IDs to their Safe API URLs
var chainIDToAPIURL = map[uint64]string{
	MainnetChainID:   "https://safe-transaction-mainnet.safe.global",
	OPMainnetChainID: "https://safe-transaction-optimism.safe.global",
}

// ExtractTransactionHash extracts a transaction hash from a Safe URL or returns the hash if already provided
func ExtractTransactionHash(input string) (string, error) {
	input = strings.TrimSpace(input)

	// Check if input is a URL
	if strings.Contains(input, "app.safe.global") && strings.Contains(input, "id=") {
		// Extract the transaction hash from the URL
		idParamRegex := regexp.MustCompile(`id=([^&]+)`)
		matches := idParamRegex.FindStringSubmatch(input)
		if len(matches) < 2 {
			return "", fmt.Errorf("could not extract transaction ID from URL")
		}

		txHash := matches[1]

		// The hash might be part of a longer string like:
		// - multisig_0x...address_0x...hash
		// - transfer_0x...address_hash (without 0x prefix on hash)
		if strings.Contains(txHash, "_0x") {
			parts := strings.Split(txHash, "_")
			// Return the last part after the last underscore (the actual hash)
			hash := parts[len(parts)-1]

			// Add 0x prefix if missing
			if !strings.HasPrefix(hash, "0x") {
				hash = "0x" + hash
			}

			return hash, nil
		}

		// If no underscore pattern, check if it already has 0x prefix
		if !strings.HasPrefix(txHash, "0x") {
			txHash = "0x" + txHash
		}

		return txHash, nil
	}

	// If it's already a hash, return it cleaned
	input = strings.TrimSpace(input)
	if !strings.HasPrefix(input, "0x") {
		return "", fmt.Errorf("invalid transaction hash format: must start with 0x")
	}

	return input, nil
}

// TransactionMetadata holds information about a transaction retrieved from the API
type TransactionMetadata struct {
	Network     string
	ChainID     uint64
	SafeAddress string
	Nonce       uint64
	Transaction *SafeTransaction
}

// FetchTransactionByHash fetches transaction data from the Safe API using a transaction hash
// It tries all supported chains and returns the transaction along with metadata
func FetchTransactionByHash(txHash string) (*TransactionMetadata, error) {
	// Validate hash format
	if !strings.HasPrefix(txHash, "0x") {
		return nil, fmt.Errorf("invalid transaction hash format: must start with 0x")
	}

	// Try each supported chain (ensure we try mainnet first)
	chainIDs := []uint64{MainnetChainID, OPMainnetChainID}
	var errors []string

	for _, chainID := range chainIDs {
		apiURL := chainIDToAPIURL[chainID]

		// Construct API endpoint using v2 API
		endpoint := fmt.Sprintf("%s/api/v2/multisig-transactions/%s/", apiURL, txHash)

		// Make HTTP request
		resp, err := http.Get(endpoint)
		if err != nil {
			errors = append(errors, fmt.Sprintf("chain %d: %v", chainID, err))
			continue
		}

		// If not found, try next chain
		if resp.StatusCode == http.StatusNotFound {
			resp.Body.Close()
			errors = append(errors, fmt.Sprintf("chain %d: transaction not found", chainID))
			continue
		}

		// Check for other errors
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			errors = append(errors, fmt.Sprintf("chain %d: API request failed with status: %s", chainID, resp.Status))
			continue
		}

		// Read response body
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			errors = append(errors, fmt.Sprintf("chain %d: error reading response body: %v", chainID, err))
			continue
		}

		// Parse response - v2 API returns a single transaction, not wrapped in APIResponse
		var apiTx struct {
			Safe           string `json:"safe"`
			To             string `json:"to"`
			Value          string `json:"value"`
			Data           string `json:"data"`
			Operation      int    `json:"operation"`
			SafeTxGas      string `json:"safeTxGas"`
			BaseGas        string `json:"baseGas"`
			GasPrice       string `json:"gasPrice"`
			GasToken       string `json:"gasToken"`
			RefundReceiver string `json:"refundReceiver"`
			Nonce          string `json:"nonce"`
		}

		if err := json.Unmarshal(body, &apiTx); err != nil {
			errors = append(errors, fmt.Sprintf("chain %d: error parsing API response: %v", chainID, err))
			continue
		}

		// We found the transaction! Now fetch the Safe version
		safeAddress := apiTx.Safe
		safeVersion, err := fetchSafeVersion(apiURL, safeAddress)
		if err != nil {
			return nil, fmt.Errorf("error fetching safe version for %s: %w", safeAddress, err)
		}

		// Parse nonce from string
		var nonce int
		fmt.Sscanf(apiTx.Nonce, "%d", &nonce)

		// Check if this is an approveHash transaction
		var nested *Nested
		content := apiTx

		if apiTx.Data != "" && strings.HasPrefix(apiTx.Data, "0xd4d9bdcd") {
			// Approve-hash calldata must be exactly 74 characters:
			//  - 0x prefix (2)
			//  - 4-byte function selector (8 hex chars)
			//  - 32-byte transaction hash (64 hex chars)
			if len(apiTx.Data) != 74 {
				return nil, fmt.Errorf("invalid approveHash calldata length: got %d, want 74 (0x + 4-byte selector + 32-byte hash)", len(apiTx.Data))
			}

			// Extract the inner transaction hash (skip 0x + 8 selector chars => start at index 10)
			innerHash := "0x" + apiTx.Data[10:74]

			// Fetch the inner transaction
			innerEndpoint := fmt.Sprintf("%s/api/v2/multisig-transactions/%s/", apiURL, innerHash)
			innerResp, err := http.Get(innerEndpoint)
			if err == nil && innerResp.StatusCode == http.StatusOK {
				innerBody, err := io.ReadAll(innerResp.Body)
				innerResp.Body.Close()
				if err == nil {
					var innerTx struct {
						To             string `json:"to"`
						Value          string `json:"value"`
						Data           string `json:"data"`
						Operation      int    `json:"operation"`
						SafeTxGas      string `json:"safeTxGas"`
						BaseGas        string `json:"baseGas"`
						GasPrice       string `json:"gasPrice"`
						GasToken       string `json:"gasToken"`
						RefundReceiver string `json:"refundReceiver"`
						Safe           string `json:"safe"`
					}

					if err := json.Unmarshal(innerBody, &innerTx); err == nil {
						// Create nested data from outer transaction
						nested = &Nested{
							Safe:        safeAddress,
							SafeVersion: safeVersion,
							Nonce:       nonce,
							Data:        apiTx.Data,
							Operation:   apiTx.Operation,
							To:          apiTx.To,
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

						// Update to use inner safe address
						safeAddress = innerTx.Safe

						// Fetch the inner safe's version
						safeVersion, err = fetchSafeVersion(apiURL, innerTx.Safe)
						if err != nil {
							return nil, fmt.Errorf("error fetching inner safe version: %w", err)
						}
					}
				}
			}
		}

		// Convert string values to appropriate types
		var value, safeTxGas, baseGas, gasPrice int
		fmt.Sscanf(content.Value, "%d", &value)
		fmt.Sscanf(content.SafeTxGas, "%d", &safeTxGas)
		fmt.Sscanf(content.BaseGas, "%d", &baseGas)
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
			SafeTxGas:      safeTxGas,
			BaseGas:        baseGas,
			GasPrice:       gasPrice,
			GasToken:       content.GasToken,
			RefundReceiver: content.RefundReceiver,
			Nonce:          nonce,
			Nested:         nested,
		}

		// Determine network name from chain ID
		var networkName string
		switch chainID {
		case MainnetChainID:
			networkName = "ethereum"
		case OPMainnetChainID:
			networkName = "op"
		default:
			networkName = fmt.Sprintf("chain-%d", chainID)
		}

		return &TransactionMetadata{
			Network:     networkName,
			ChainID:     chainID,
			SafeAddress: safeAddress,
			Nonce:       uint64(nonce),
			Transaction: safeTx,
		}, nil
	}

	// If we get here, we didn't find the transaction on any chain
	if len(errors) > 0 {
		return nil, fmt.Errorf("transaction not found on any supported chain. Errors: %s", strings.Join(errors, "; "))
	}

	return nil, fmt.Errorf("transaction not found on any supported chain")
}
