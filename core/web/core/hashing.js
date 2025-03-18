import { ethers } from '../lib/ethers/index.js';
import { DOMAIN_SEPARATOR_TYPEHASH, SAFE_TX_TYPEHASH } from './constants.js';

/**
 * Calculates the EIP-712 domain hash for a Safe transaction
 * @param {Object} tx - The Safe transaction object
 * @returns {string} - The hex-encoded domain hash
 */
export function calculateDomainHash(tx) {
  // Pack domain values similar to the Go implementation
  const encodedData = ethers.utils.defaultAbiCoder.encode(
    ['bytes32', 'uint256', 'address'],
    [
      DOMAIN_SEPARATOR_TYPEHASH,
      ethers.BigNumber.from(tx.chain),
      tx.safe
    ]
  );

  // Calculate hash
  return ethers.utils.keccak256(encodedData);
}

/**
 * Calculates the EIP-712 message hash for a Safe transaction
 * @param {Object} tx - The Safe transaction object
 * @returns {string} - The hex-encoded message hash
 */
export function calculateMessageHash(tx) {
  // Calculate data hash
  const dataHash = ethers.utils.keccak256(tx.data || '0x');
  
  // Pack message values similar to the Go implementation
  const encodedData = ethers.utils.defaultAbiCoder.encode(
    ['bytes32', 'address', 'uint256', 'bytes32', 'uint8', 'uint256', 'uint256', 'uint256', 'address', 'address', 'uint256'],
    [
      SAFE_TX_TYPEHASH,
      tx.to,
      ethers.BigNumber.from(tx.value),
      dataHash,
      tx.operation,
      ethers.BigNumber.from(tx.safeTxGas),
      ethers.BigNumber.from(tx.baseGas),
      ethers.BigNumber.from(tx.gasPrice),
      tx.gasToken,
      tx.refundReceiver,
      ethers.BigNumber.from(tx.nonce)
    ]
  );

  // Calculate hash
  return ethers.utils.keccak256(encodedData);
}

/**
 * Calculates the EIP-712 approve hash for a Safe transaction
 * @param {Object} tx - The Safe transaction object
 * @returns {string} - The hex-encoded approve hash
 */
export function calculateApproveHash(tx) {
  // First calculate domain hash
  const domainHash = calculateDomainHash(tx);
  
  // Then calculate message hash
  const messageHash = calculateMessageHash(tx);

  // Create the EIP-712 prefix: 0x1901
  const prefix = '0x1901';
  
  // Concatenate all components and calculate final hash
  const concatData = ethers.utils.hexConcat([
    prefix,
    domainHash,
    messageHash
  ]);
  
  return ethers.utils.keccak256(concatData);
}
