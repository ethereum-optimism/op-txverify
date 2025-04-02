/**
 * VerificationResult represents the complete output of the verification process
 * @typedef {Object} VerificationResult
 * @property {Object} transaction - The Safe transaction
 * @property {string} domainHash - Domain hash for EIP-712
 * @property {string} messageHash - Message hash for EIP-712
 * @property {string} approveHash - Final hash to be approved
 * @property {Object} call - Parsed call data
 * @property {VerificationResult} [nestedResult] - Result for nested transaction
 */

/**
 * Verify a Safe transaction
 * @param {Object} tx - The Safe transaction to verify
 * @param {Object} options - Verification options
 * @returns {VerificationResult} The verification result
 */
function verifyTransaction(tx, options = {}) {
  // Check if this is a nested transaction
  let nestedResult = null;
  
  if (tx.nested) {
      // Verify the inner transaction first
      nestedResult = verifyTransactionInternal(tx, options);
      
      // Manipulate the transaction to generate the outer result
      const originalTx = { ...tx };
      tx.to = originalTx.nested.safe;
      tx.safe = originalTx.nested.safe;
      tx.nonce = originalTx.nested.nonce;
      tx.operation = originalTx.nested.operation;
      tx.value = 0;
      tx.data = originalTx.nested.data;
  }
  
  // Verify the main transaction
  const result = verifyTransactionInternal(tx, options);
  
  // Attach nested result if it exists
  if (nestedResult) {
      result.nestedResult = nestedResult;
  }
  
  return result;
}

/**
* Internal function for transaction verification
* @param {Object} tx - The Safe transaction to verify
* @param {Object} options - Verification options
* @returns {VerificationResult} The verification result
*/
function verifyTransactionInternal(tx, options = {}) {
  // Strip chain prefix from addresses
  tx.to = stripChainPrefix(tx.to);
  tx.safe = stripChainPrefix(tx.safe);
  
  // Parse the transaction data
  const call = parseTransactionData(tx.to, tx.data, tx.chain, options);
  
  // Calculate the domain and message hashes
  const domainHash = calculateDomainHash(tx);
  const messageHash = calculateMessageHash(tx);
  const approveHash = calculateApproveHash(tx);
  
  // Create the verification result
  return {
      transaction: tx,
      domainHash: domainHash,
      messageHash: messageHash,
      approveHash: approveHash,
      call: call
  };
}

/**
* Removes chain prefixes like "oeth:", "eth:", etc. from addresses
* @param {string} address - The address to process
* @returns {string} Address without chain prefix
*/
function stripChainPrefix(address) {
  if (address.includes(':')) {
      return address.split(':')[1];
  }
  return address;
}
