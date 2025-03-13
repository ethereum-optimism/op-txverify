package core

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// CalculateDomainHash calculates the EIP-712 domain hash for a Safe transaction
func CalculateDomainHash(tx SafeTransaction) (string, error) {
	// Parse the ABI types
	domainType, err := abi.NewType("tuple", "", []abi.ArgumentMarshaling{
		{Name: "typehash", Type: "bytes32"},
		{Name: "chainId", Type: "uint256"},
		{Name: "verifyingContract", Type: "address"},
	})
	if err != nil {
		return "", err
	}

	// Create the ABI arguments
	arguments := abi.Arguments{
		abi.Argument{Type: domainType, Name: "domain"},
	}

	// Convert inputs to appropriate types
	typehash := common.HexToHash(DomainSeparatorTypehash)
	chainID := big.NewInt(int64(tx.Chain))
	safeAddress := common.HexToAddress(tx.Safe)

	// Pack the values
	packed, err := arguments.Pack(
		struct {
			Typehash          [32]byte
			ChainId           *big.Int
			VerifyingContract common.Address
		}{
			Typehash:          typehash,
			ChainId:           chainID,
			VerifyingContract: safeAddress,
		},
	)
	if err != nil {
		return "", err
	}

	// Calculate hash
	hash := crypto.Keccak256Hash(packed)
	return hash.Hex(), nil
}

// CalculateMessageHash calculates the EIP-712 message hash for a Safe transaction
func CalculateMessageHash(tx SafeTransaction) (string, error) {
	// Parse the ABI types
	messageType, err := abi.NewType("tuple", "", []abi.ArgumentMarshaling{
		{Name: "typehash", Type: "bytes32"},
		{Name: "to", Type: "address"},
		{Name: "value", Type: "uint256"},
		{Name: "dataHash", Type: "bytes32"},
		{Name: "operation", Type: "uint8"},
		{Name: "safeTxGas", Type: "uint256"},
		{Name: "baseGas", Type: "uint256"},
		{Name: "gasPrice", Type: "uint256"},
		{Name: "gasToken", Type: "address"},
		{Name: "refundReceiver", Type: "address"},
		{Name: "nonce", Type: "uint256"},
	})
	if err != nil {
		return "", err
	}

	// Create the ABI arguments
	arguments := abi.Arguments{
		abi.Argument{Type: messageType, Name: "message"},
	}

	// Convert inputs to appropriate types
	typehash := common.HexToHash(SafeTxTypehash)
	toAddress := common.HexToAddress(tx.To)

	// Convert value to big.Int
	value := big.NewInt(int64(tx.Value))

	// Calculate data hash
	dataBytes := common.FromHex(tx.Data)
	dataHash := crypto.Keccak256Hash(dataBytes)

	// Convert uint values to big.Int
	safeTxGas := big.NewInt(int64(tx.SafeTxGas))
	baseGas := big.NewInt(int64(tx.BaseGas))
	gasPrice := big.NewInt(int64(tx.GasPrice))
	nonce := big.NewInt(int64(tx.Nonce))

	// Pack the values
	packed, err := arguments.Pack(
		struct {
			Typehash       [32]byte
			To             common.Address
			Value          *big.Int
			DataHash       [32]byte
			Operation      uint8
			SafeTxGas      *big.Int
			BaseGas        *big.Int
			GasPrice       *big.Int
			GasToken       common.Address
			RefundReceiver common.Address
			Nonce          *big.Int
		}{
			Typehash:       typehash,
			To:             toAddress,
			Value:          value,
			DataHash:       dataHash,
			Operation:      uint8(tx.Operation),
			SafeTxGas:      safeTxGas,
			BaseGas:        baseGas,
			GasPrice:       gasPrice,
			GasToken:       common.HexToAddress(tx.GasToken),
			RefundReceiver: common.HexToAddress(tx.RefundReceiver),
			Nonce:          nonce,
		},
	)
	if err != nil {
		return "", err
	}

	// Calculate hash
	hash := crypto.Keccak256Hash(packed)
	return hash.Hex(), nil
}

// CalculateApproveHash calculates the EIP-712 approve hash for a Safe transaction
func CalculateApproveHash(tx SafeTransaction) (string, error) {
	// First calculate domain hash
	domainHash, err := CalculateDomainHash(tx)
	if err != nil {
		return "", err
	}

	// Then calculate message hash
	messageHash, err := CalculateMessageHash(tx)
	if err != nil {
		return "", err
	}

	// Convert hex strings to byte arrays
	domainHashBytes := common.HexToHash(domainHash)
	messageHashBytes := common.HexToHash(messageHash)

	// Create the EIP-712 prefix: 0x1901
	prefix := []byte{0x19, 0x01}

	// Concatenate all components
	concatData := append(prefix, domainHashBytes.Bytes()...)
	concatData = append(concatData, messageHashBytes.Bytes()...)

	// Calculate final hash
	hash := crypto.Keccak256Hash(concatData)
	return hash.Hex(), nil
}
