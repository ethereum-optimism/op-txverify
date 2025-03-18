/**
 * Parse the transaction data and identify the function call
 * @param {string} to - Target address
 * @param {string} data - Transaction data
 * @param {number} chainID - Chain ID
 * @param {Object} options - Verification options
 * @returns {Object} Parsed call data
 */
function parseTransactionData(to, data, chainID, options = {}) {
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

  // Get function selector
  const functionSelector = cleanData.substring(0, 8);

  // Check if we know this function
  if (KNOWN_FUNCTIONS[functionSelector]) {
    const functionInfo = KNOWN_FUNCTIONS[functionSelector];
    
    // Parse the function arguments
    let parsedData = {};
    try {
      parsedData = parseArguments(functionInfo, cleanData);

      // If this is a token function on a known contract with decimals, adjust the amount
      if (contractInfo && TOKEN_FUNCTIONS[functionInfo.name] && 
          contractInfo.decimals > 0 && parsedData.amount) {
        parsedData.amount = parseDecimals(parsedData.amount, contractInfo.decimals);
      }
      
      // Check if any of the arguments are known contracts
      for (const [key, value] of Object.entries(parsedData)) {
        if (typeof value === 'string' && value.startsWith('0x') && value.length === 42) {
          const argContract = getKnownContract(value.toLowerCase(), chainID);
          if (argContract) {
            parsedData[key] = `${value} (${argContract.name} ✅)`;
          }
        }
      }
    } catch (err) {
      console.error("Error parsing function arguments:", err);
    }
    
    // Check if this is a multicall contract and function
    const isMulticallContract = MULTICALL_ADDRESSES[chainID] && 
      MULTICALL_ADDRESSES[chainID][normalizedTo];
    const isMulticallFunction = functionInfo.name === "multiSend" || 
      functionInfo.name === "aggregate3";
    
    if (isMulticallContract && isMulticallFunction) {
      try {
        const subcalls = parseMulticall(normalizedTo, chainID, functionInfo, parsedData, options);
        return {
          target: to,
          targetName: targetName,
          functionName: functionInfo.name,
          subCalls: subcalls
        };
      } catch (err) {
        console.error("Error parsing multicall:", err);
      }
    }
    
    // Regular function call
    return {
      target: to,
      targetName: targetName,
      functionName: functionInfo.name,
      functionData: data,
      parsedData: parsedData
    };
  }

  // Unknown function
  return {
    target: to,
    targetName: targetName,
    functionName: "unknown",
    rawData: data
  };
}

/**
 * Parses the amount and returns it as a string with the correct number of decimals
 * @param {BigNumber} amount - Amount as BigNumber
 * @param {number} decimals - Number of decimals
 * @returns {string} Formatted string
 */
function parseDecimals(amount, decimals) {
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
 * Converts byte array to hex string
 * @param {Uint8Array|Array<number>} bytes - Byte array
 * @returns {string} Hex string
 */
function bytesToHex(bytes) {
  return Array.from(bytes).map(b => b.toString(16).padStart(2, '0')).join('');
}

/**
 * Decodes the function arguments from calldata
 * @param {Object} functionInfo - FunctionFragment object
 * @param {string} calldata - Transaction calldata
 * @returns {Object} Decoded arguments
 */
function parseArguments(functionInfo, calldata) {
  // Remove 0x prefix if present
  calldata = calldata.startsWith('0x') ? calldata : '0x' + calldata;

  // Create an interface with just this function
  const iface = new ethers.utils.Interface([functionInfo]);
  
  // Decode the call data
  const decodedData = iface.parseTransaction({ data: calldata });
  
  // Convert to a map
  const result = {};
  for (let i = 0; i < decodedData.args.length; i++) {
    let name = functionInfo.inputs[i]?.name || `arg${i}`;
    let arg = decodedData.args[i];

    // Handle byte arrays for better readability
    if (arg instanceof Uint8Array || (Array.isArray(arg) && arg.every(item => typeof item === 'number' && item < 256))) {
      result[name] = '0x' + bytesToHex(arg);
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
    if (getFunctionSelector(functionInfo.format()) === getFunctionSelector(SAFE_MULTISEND_SIG)) {
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
        const to = ethers.utils.getAddress('0x' + bytesToHex(data.slice(pos, pos + 20)));
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
        const callData = '0x' + bytesToHex(data.slice(pos, pos + length));
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
      throw new Error(`Unsupported function ${functionInfo.format()} for Safe Multisend contract`);
    }
  } else if (normalizedAddress === MULTICALL3_ADDRESS.toLowerCase()) {
    // For Multicall3 contract, check the function signature
    if (getFunctionSelector(functionInfo.format()) === getFunctionSelector(AGGREGATE3_SIG)) {
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
      throw new Error(`Unsupported function ${functionInfo.format()} for Multicall3 contract`);
    }
  } else {
    // No generic parsing - return an error for unknown contracts
    throw new Error(`Unsupported multicall contract: ${contractAddress}`);
  }

  return subcalls;
}
