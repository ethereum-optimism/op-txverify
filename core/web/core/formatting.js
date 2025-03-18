/**
 * Parses a number with decimal places
 * @param {number|string} value - The value to parse
 * @param {number} decimals - Number of decimal places
 * @returns {string} Formatted string with correct decimal places
 */
function parseDecimals(value, decimals) {
    if (!value) return '0';
    
    // Convert to string and remove any existing decimal point
    let valueStr = value.toString().replace('.', '');
    
    // Pad with leading zeros if needed
    if (valueStr.length <= decimals) {
        valueStr = '0'.repeat(decimals - valueStr.length + 1) + valueStr;
    }
    
    // Insert decimal point
    const decimalPos = valueStr.length - decimals;
    const formattedValue = valueStr.slice(0, decimalPos) + '.' + valueStr.slice(decimalPos);
    
    // Remove trailing zeros after decimal point
    return formattedValue.replace(/\.?0+$/, '');
}

/**
 * Creates a formatted result object for display
 * @param {Object} result - The verification result
 * @returns {Object} Formatted result for display
 */
function formatResultForDisplay(result) {
    const tx = result.transaction;
    const displayResult = {
        basicInfo: {
            safe: formatAddressWithKnownContract(tx.safe, tx.chain),
            chainId: formatChainId(tx.chain),
            target: formatAddressWithKnownContract(tx.to, tx.chain),
            value: parseDecimals(tx.value, 18) + ' ETH',
            nonce: tx.nonce,
            operation: formatOperation(tx.operation)
        },
        hasNestedTransaction: !!result.nestedResult,
        callDetails: result.call,
        hashes: {
            domainHash: result.domainHash,
            messageHash: result.messageHash,
            approveHash: result.approveHash
        }
    };
    
    if (result.nestedResult) {
        const nestedTx = result.nestedResult.transaction;
        displayResult.nestedInfo = {
            safe: formatAddressWithKnownContract(nestedTx.safe, nestedTx.chain),
            nonce: nestedTx.nonce,
            hash: result.nestedResult.approveHash,
            callDetails: result.nestedResult.call
        };
    }
    
    return displayResult;
}

/**
 * Formats an address with known contract name if available
 * @param {string} address - The address
 * @param {number} chainId - The chain ID
 * @returns {string} Formatted address string
 */
function formatAddressWithKnownContract(address, chainId) {
    const contractInfo = getKnownContract(address, chainId);
    if (contractInfo) {
        return `${address} (${contractInfo.name} ✅)`;
    }
    return address;
}

/**
 * Formats a chain ID with network name if known
 * @param {number} chainId - The chain ID
 * @returns {string} Formatted chain ID string
 */
function formatChainId(chainId) {
    const networkNames = {
        1: 'Ethereum',
        10: 'OP Mainnet',
        8453: 'Base',
        42161: 'Arbitrum One'
    };
    
    if (networkNames[chainId]) {
        return `${chainId} (${networkNames[chainId]} ✅)`;
    }
    return chainId.toString();
}

/**
 * Formats the operation type
 * @param {number} operation - Operation code
 * @returns {string} Operation description
 */
function formatOperation(operation) {
    if (operation === 0) return 'CALL';
    if (operation === 1) return 'DELEGATECALL';
    return 'UNKNOWN OPERATION ❌';
} 