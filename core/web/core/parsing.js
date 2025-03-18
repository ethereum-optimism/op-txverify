import { ethers } from 'ethers';
import { 
  SAFE_MULTISEND_ADDRESS,
  SAFE_MULTISEND_CALL_ONLY_141,
  SAFE_MULTISEND_SIG,
  MULTICALL3_ADDRESS,
  AGGREGATE3_SIG,
  KNOWN_FUNCTIONS as knownFunctions,
  TOKEN_FUNCTIONS as tokenFunctions,
  MULTICALL_ADDRESSES as multicallAddresses,
  getKnownContract
} from './constants.js';

/**
 * Parse the transaction data and identify the function call
 * @param {string} to - Target address
 * @param {string} data - Transaction data
 * @param {number} chainID - Chain ID
 * @param {Object} options - Verification options
 * @returns {Object} Parsed call data
 */
export function parseTransactionData(to, data, chainID, options = {}) {
  // Remove 0x prefix if present
  const cleanData = data.startsWith('0x') ? data.slice(2) : data;

  // Normalize the target address
  const normalizedTo = to.toLowerCase();

  // Check if this is a known contract
  const contractInfo = getKnownContract(normalizedTo, chainID);
  const targetName = contractInfo ? contractInfo.name : "";

  // If data is empty, return a simple transfer call
  if (cleanData.length === 0) {
    return {
      target: to,
      targetName: targetName,
      functionName: "unknown",
      rawData: "0x" + cleanData
    };
  }

  // Extract function selector (first 4 bytes)
  if (cleanData.length < 8) {
    return {
      target: to,
      targetName: targetName,
      functionName: "unknown",
      rawData: data
    };
  }

  const functionSelector = cleanData.slice(0, 8);

  // Try to identify the function from known selectors
  const functionInfo = knownFunctions[functionSelector];

  if (!functionInfo) {
    // If we can't identify the function, return the raw data
    return {
      target: to,
      targetName: targetName,
      functionName: "unknown",
      rawData: data
    };
  }

  // Parse the function arguments
  let parsedArgs;
  try {
    parsedArgs = parseArguments(functionInfo.abi, "0x" + cleanData);
  } catch (err) {
    // If we can't parse the arguments, return the raw data
    return {
      target: to,
      targetName: targetName,
      functionName: functionInfo.name,
      rawData: data
    };
  }

  // If this is a token function on a known contract where we have decimals, adjust the amount
  if (contractInfo && tokenFunctions[functionInfo.name] && contractInfo.decimals > 0 && parsedArgs["amount"]) {
    parsedArgs["amount"] = parseDecimals(parsedArgs["amount"], contractInfo.decimals);
  }

  // Check if any of the arguments are known contracts
  for (const [key, value] of Object.entries(parsedArgs)) {
    if (value && typeof value === 'string' && ethers.utils.isAddress(value)) {
      const knownContract = getKnownContract(value, chainID);
      if (knownContract) {
        parsedArgs[key] = `${value} (${knownContract.name} ✅)`;
      }
    }
  }

  // Check if this is a multicall contract and the function is a multicall function
  const isMulticallContract = multicallAddresses[chainID] && 
                             multicallAddresses[chainID][normalizedTo];
  
  const isMulticallFunction = functionInfo.name === "multiSend" || functionInfo.name === "aggregate3";

  if (isMulticallContract && isMulticallFunction) {
    // Parse subcalls - passing contract address, chain ID, and full function info
    try {
      const subcalls = parseMulticall(normalizedTo, chainID, functionInfo, parsedArgs, options);
      
      // Return with subcalls
      return {
        target: to,
        targetName: targetName,
        functionName: functionInfo.name,
        subCalls: subcalls
      };
    } catch (err) {
      console.error("Error parsing multicall:", err);
      // Return without subcalls in case of error
      return {
        target: to,
        targetName: targetName,
        functionName: functionInfo.name,
        parsedData: parsedArgs
      };
    }
  }

  // Regular function call
  return {
    target: to,
    targetName: targetName,
    functionName: functionInfo.name,
    parsedData: parsedArgs
  };
}

/**
 * Parses the amount and returns it as a string with the correct number of decimals
 * @param {BigNumber} amount - Amount as BigNumber
 * @param {number} decimals - Number of decimals
 * @returns {string} Formatted string
 */
export function parseDecimals(amount, decimals) {
  let amountStr = amount.toString();

  // Pad with leading zeros if needed
  if (amountStr.length <= decimals) {
    amountStr = '0'.repeat(decimals - amountStr.length + 1) + amountStr;
  }

  // Insert decimal point at the right position
  const decimalPos = amountStr.length - decimals;
  return amountStr.slice(0, decimalPos) + '.' + amountStr.slice(decimalPos);
}

/**
 * Decodes the function arguments from calldata
 * @param {Object} method - ABI method definition
 * @param {string} calldata - Transaction calldata
 * @returns {Object} Decoded arguments
 */
function parseArguments(method, calldata) {
  // Remove 0x prefix if present
  calldata = calldata.startsWith('0x') ? calldata : '0x' + calldata;

  // Create an interface with just this function
  const iface = new ethers.utils.Interface([method]);
  
  // Decode the call data
  const decodedData = iface.parseTransaction({ data: calldata });
  
  // Convert to a map
  const result = {};
  for (let i = 0; i < decodedData.args.length; i++) {
    let name = method.inputs[i]?.name || `arg${i}`;
    let arg = decodedData.args[i];

    // Handle byte arrays for better readability
    if (arg instanceof Uint8Array || (Array.isArray(arg) && arg.every(item => typeof item === 'number' && item < 256))) {
      result[name] = '0x' + Buffer.from(arg).toString('hex');
    } else {
      result[name] = arg;
    }
  }

  return result;
}

/**
 * Parses subcalls from a multicall function
 * @param {string} contractAddress - Multicall contract address
 * @param {number} chainID - Chain ID
 * @param {Object} functionInfo - Function information
 * @param {Object} args - Function arguments
 * @param {Object} options - Verification options
 * @returns {Array} Array of parsed subcalls
 */
function parseMulticall(contractAddress, chainID, functionInfo, args, options) {
  const subcalls = [];
  const normalizedAddress = contractAddress.toLowerCase();

  // Handle Safe Multisend contracts
  if (normalizedAddress === SAFE_MULTISEND_ADDRESS.toLowerCase() || 
      normalizedAddress === SAFE_MULTISEND_CALL_ONLY_141.toLowerCase()) {
    
    // For Safe Multisend contract, check the function signature
    if (functionInfo.signature === SAFE_MULTISEND_SIG) {
      // Parse multiSend calldata
      const hexData = args["transactions"];
      if (!hexData || !hexData.startsWith('0x')) {
        throw new Error("Invalid multiSend data format");
      }

      // Convert hex string to bytes
      const data = ethers.utils.arrayify(hexData);

      // multiSend data format: operation (1 byte) + to (20 bytes) + value (32 bytes) + dataLength (32 bytes) + data (variable)
      let pos = 0;
      while (pos < data.length) {
        // Ensure we have enough data for the fixed-size fields
        if (pos + 85 > data.length) break;

        // Extract operation (skip as it's always 0 per comment in Go code)
        pos++;

        // Extract to address
        const to = ethers.utils.getAddress('0x' + Buffer.from(data.slice(pos, pos + 20)).toString('hex'));
        pos += 20;

        // Skip extracting value for now
        pos += 32;

        // Extract data length
        const dataLengthBytes = data.slice(pos, pos + 32);
        pos += 32;
        const length = ethers.BigNumber.from(dataLengthBytes).toNumber();

        // Ensure we have enough data for the variable-size field
        if (pos + length > data.length) break;

        // Extract call data
        const callData = '0x' + Buffer.from(data.slice(pos, pos + length)).toString('hex');
        pos += length;

        // Parse the subcall
        try {
          const subcall = parseTransactionData(to, callData, chainID, options);
          subcalls.push(subcall);
        } catch (err) {
          console.error(`Error parsing subcall to ${to}:`, err);
        }
      }
    } else {
      throw new Error(`Unsupported function ${functionInfo.signature} for Safe Multisend contract`);
    }
  } else if (normalizedAddress === MULTICALL3_ADDRESS.toLowerCase()) {
    // For Multicall3 contract, check the function signature
    if (functionInfo.signature === AGGREGATE3_SIG) {
      // Parse aggregate3 calldata
      const calls = args["calls"];
      if (!Array.isArray(calls)) {
        throw new Error("Invalid aggregate3 data format");
      }

      for (const call of calls) {
        try {
          const subcall = parseTransactionData(call.target, call.callData, chainID, options);
          subcalls.push(subcall);
        } catch (err) {
          console.error(`Error parsing subcall to ${call.target}:`, err);
        }
      }
    } else {
      throw new Error(`Unsupported function ${functionInfo.signature} for Multicall3 contract`);
    }
  } else {
    // No generic parsing - return an error for unknown contracts
    throw new Error(`Unsupported multicall contract: ${contractAddress}`);
  }

  return subcalls;
}
