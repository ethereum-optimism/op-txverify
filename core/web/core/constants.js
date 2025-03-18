import { ethers } from 'ethers';

// EIP-712 constants
export const DOMAIN_SEPARATOR_TYPEHASH = '0x47e79534a245952e8b16893a336b85a3d9ea9fa8c573f3d803afb92a79469218';
export const SAFE_TX_TYPEHASH = '0xbb8310d486368db6bd6f849402fdd73ad53d316b5a4b2644ad6efe0f941286d8';

// Function signatures needed for parseMulticall
export const SAFE_MULTISEND_SIG = 'multiSend(bytes transactions)';
export const AGGREGATE3_SIG = 'aggregate3((address target,bool allowFailure,bytes callData)[] calls)';

// Known contract addresses
export const SAFE_MULTISEND_ADDRESS = '0xA1dabEF33b3B82c7814B6D82A79e50F4AC44102B';
export const SAFE_MULTISEND_CALL_ONLY_141 = '0x9641d764fc13c8B624c04430C7356C1C7C8102e2';
export const MULTICALL3_ADDRESS = '0xcA11bde05977b3631167028862bE2a173976CA11';
export const USDC_MAINNET_ADDRESS = '0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48';
export const OP_TOKEN_ADDRESS = '0x4200000000000000000000000000000000000042';
export const SUPERFLUID_OP = '0x1828Bff08BD244F7990edDCd9B19cc654b33cDB4';
export const OPTIMISM_GOVERNOR = '0xcDF27F107725988f2261Ce2256bDfCdE8B382B10';
export const OP_GRANTS1 = '0x2501c477D0A35545a387Aa4A3EEe4292A9a8B3F0';
export const OP_GRANTS2 = '0x19793c7824Be70ec58BB673CA42D2779d12581BE';

// Known function signatures
export const KNOWN_SIGNATURES = [
  'transfer(address to,uint256 amount)',
  'transferFrom(address from,address to,uint256 amount)',
  'approve(address spender,uint256 amount)',
  'increaseAllowance(address spender,uint256 amount)',
  'decreaseAllowance(address spender,uint256 amount)',
  'approveHash(bytes32 hashToApprove)',
  'aggregate3((address target,bool allowFailure,bytes callData)[] calls)',
  'multiSend(bytes transactions)',
  'callAgreement(address agreementClass, bytes callData, bytes userData)',
  'createVestingScheduleFromAmountAndDuration(address superToken, address receiver, uint256 totalAmount, uint32 totalDuration, uint32 startDate, uint32 cliffPeriod, uint32 claimPeriod)',
  'propose(address[] targets, uint256[] values, bytes[] calldatas, string description, uint8 proposalType)',
];

// Functions on ERC20 tokens that require decimal adjustment
export const TOKEN_FUNCTIONS = {
  'transfer': true,
  'transferFrom': true,
  'approve': true,
  'increaseAllowance': true,
  'decreaseAllowance': true
};

// ChainID constants for supported networks
export const MAINNET_CHAIN_ID = 1;
export const OP_MAINNET_CHAIN_ID = 10;

// ChainNames maps chain IDs to their names
export const CHAIN_NAMES = {
  [MAINNET_CHAIN_ID]: 'Ethereum',
  [OP_MAINNET_CHAIN_ID]: 'OP Mainnet'
};

// KnownContracts maps chain IDs to a map of addresses to contract info
export const KNOWN_CONTRACTS = {
  [MAINNET_CHAIN_ID]: {
    [SAFE_MULTISEND_ADDRESS.toLowerCase()]: { name: 'GNOSIS SAFE MULTISEND', decimals: 0 },
    [MULTICALL3_ADDRESS.toLowerCase()]: { name: 'MULTICALL3', decimals: 0 },
    [USDC_MAINNET_ADDRESS.toLowerCase()]: { name: 'USDC', decimals: 6 },
  },
  [OP_MAINNET_CHAIN_ID]: {
    [SAFE_MULTISEND_ADDRESS.toLowerCase()]: { name: 'GNOSIS SAFE MULTISEND CALL ONLY', decimals: 0 },
    [SAFE_MULTISEND_CALL_ONLY_141.toLowerCase()]: { name: 'GNOSIS SAFE MULTISEND CALL ONLY', decimals: 0 },
    [MULTICALL3_ADDRESS.toLowerCase()]: { name: 'MULTICALL3', decimals: 0 },
    [OP_TOKEN_ADDRESS.toLowerCase()]: { name: 'OP TOKEN', decimals: 18 },
    [SUPERFLUID_OP.toLowerCase()]: { name: 'SUPERFLUID OP', decimals: 18 },
    [OPTIMISM_GOVERNOR.toLowerCase()]: { name: 'OPTIMISM GOVERNOR', decimals: 0 },
    [OP_GRANTS1.toLowerCase()]: { name: 'OP GRANTS 1 (3F0)', decimals: 0 },
    [OP_GRANTS2.toLowerCase()]: { name: 'OP GRANTS 2 (1BE)', decimals: 0 },
  }
};

// MulticallAddresses maps chain IDs to a set of addresses known to be multicall contracts
export const MULTICALL_ADDRESSES = {
  [MAINNET_CHAIN_ID]: {
    [SAFE_MULTISEND_ADDRESS.toLowerCase()]: true,
    [MULTICALL3_ADDRESS.toLowerCase()]: true,
  },
  [OP_MAINNET_CHAIN_ID]: {
    [SAFE_MULTISEND_ADDRESS.toLowerCase()]: true,
    [SAFE_MULTISEND_CALL_ONLY_141.toLowerCase()]: true,
    [MULTICALL3_ADDRESS.toLowerCase()]: true,
  }
};

// Initialize known functions map based on signatures
export function parseFunctionSignature(signature) {
  // Extract function name from signature
  const functionName = signature.split('(')[0];
  
  // Get parameters string and remove closing parenthesis
  const paramsStr = signature.split('(')[1].slice(0, -1);
  
  // Generate inputs array
  const inputs = [];
  if (paramsStr) {
    const params = paramsStr.split(',');
    for (let i = 0; i < params.length; i++) {
      const parts = params[i].trim().split(' ');
      const paramType = parts[0];
      const paramName = parts.length > 1 ? parts[1] : `arg${i}`;
      inputs.push({ name: paramName, type: paramType });
    }
  }
  
  // Create ABI fragment for this function
  const abiFragment = {
    name: functionName,
    type: 'function',
    inputs: inputs,
    outputs: []
  };
  
  // Calculate function selector
  const functionInterface = new ethers.utils.Interface([abiFragment]);
  const functionSelector = functionInterface.getSighash(functionName);
  
  return {
    name: functionName,
    signature: signature,
    abi: abiFragment,
    selector: functionSelector.slice(2) // remove 0x prefix
  };
}

// Process KNOWN_SIGNATURES to generate KNOWN_FUNCTIONS
export const KNOWN_FUNCTIONS = {};
KNOWN_SIGNATURES.forEach(signature => {
  try {
    const funcInfo = parseFunctionSignature(signature);
    KNOWN_FUNCTIONS[funcInfo.selector] = funcInfo;
  } catch (err) {
    console.error(`Error parsing function signature ${signature}:`, err);
  }
});

// Helper function to get known contract info
export function getKnownContract(address, chainID) {
  const normalizedAddr = address.toLowerCase();
  if (KNOWN_CONTRACTS[chainID] && KNOWN_CONTRACTS[chainID][normalizedAddr]) {
    return KNOWN_CONTRACTS[chainID][normalizedAddr];
  }
  return null;
}
