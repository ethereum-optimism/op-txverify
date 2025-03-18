// Global constants for the application
const DOMAIN_SEPARATOR_TYPEHASH = '0x47e79534a245952e8b16893a336b85a3d9ea9fa8c573f3d803afb92a79469218';
const SAFE_TX_TYPEHASH = '0xbb8310d486368db6bd6f849402fdd73ad53d316b5a4b2644ad6efe0f941286d8';

// Function signatures needed for parseMulticall
const SAFE_MULTISEND_SIG = 'multiSend(bytes transactions)';
const AGGREGATE3_SIG = 'aggregate3((address target,bool allowFailure,bytes callData)[] calls)';

// Known contract addresses
const SAFE_MULTISEND_ADDRESS = '0xA1dabEF33b3B82c7814B6D82A79e50F4AC44102B';
const SAFE_MULTISEND_CALL_ONLY_141 = '0x9641d764fc13c8B624c04430C7356C1C7C8102e2';
const MULTICALL3_ADDRESS = '0xcA11bde05977b3631167028862bE2a173976CA11';
const USDC_MAINNET_ADDRESS = '0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48';
const OP_TOKEN_ADDRESS = '0x4200000000000000000000000000000000000042';
const SUPERFLUID_OP = '0x1828Bff08BD244F7990edDCd9B19cc654b33cDB4';
const OPTIMISM_GOVERNOR = '0xcDF27F107725988f2261Ce2256bDfCdE8B382B10';
const OP_GRANTS1 = '0x2501c477D0A35545a387Aa4A3EEe4292A9a8B3F0';
const OP_GRANTS2 = '0x19793c7824Be70ec58BB673CA42D2779d12581BE';

// ChainID constants for supported networks
const MAINNET_CHAIN_ID = 1;
const OP_MAINNET_CHAIN_ID = 10;

// ChainNames maps chain IDs to their names
const CHAIN_NAMES = {
  1: 'Ethereum',
  10: 'OP Mainnet',
  8453: 'Base',
  42161: 'Arbitrum One'
};

// Known contracts by chain ID and address
const KNOWN_CONTRACTS = {
  1: {
    // Ethereum Mainnet known contracts
    [SAFE_MULTISEND_ADDRESS.toLowerCase()]: { name: 'GNOSIS SAFE MULTISEND', decimals: 0 },
    [MULTICALL3_ADDRESS.toLowerCase()]: { name: 'MULTICALL3', decimals: 0 },
    [USDC_MAINNET_ADDRESS.toLowerCase()]: { name: 'USDC', decimals: 6 },
  },
  10: {
    // Optimism Mainnet known contracts
    [SAFE_MULTISEND_ADDRESS.toLowerCase()]: { name: 'GNOSIS SAFE MULTISEND CALL ONLY', decimals: 0 },
    [SAFE_MULTISEND_CALL_ONLY_141.toLowerCase()]: { name: 'GNOSIS SAFE MULTISEND CALL ONLY', decimals: 0 },
    [MULTICALL3_ADDRESS.toLowerCase()]: { name: 'MULTICALL3', decimals: 0 },
    [OP_TOKEN_ADDRESS.toLowerCase()]: { name: 'OP TOKEN', decimals: 18 },
    [SUPERFLUID_OP.toLowerCase()]: { name: 'SUPERFLUID OP', decimals: 18 },
    [OPTIMISM_GOVERNOR.toLowerCase()]: { name: 'OPTIMISM GOVERNOR', decimals: 0 },
    [OP_GRANTS1.toLowerCase()]: { name: 'OP GRANTS 1 (3F0)', decimals: 0 },
    [OP_GRANTS2.toLowerCase()]: { name: 'OP GRANTS 2 (1BE)', decimals: 0 },
  },
  // Add other chains as needed
};

// MulticallAddresses maps chain IDs to a set of addresses known to be multicall contracts
const MULTICALL_ADDRESSES = {
  1: {
    [SAFE_MULTISEND_ADDRESS.toLowerCase()]: true,
    [MULTICALL3_ADDRESS.toLowerCase()]: true,
  },
  10: {
    [SAFE_MULTISEND_ADDRESS.toLowerCase()]: true,
    [SAFE_MULTISEND_CALL_ONLY_141.toLowerCase()]: true,
    [MULTICALL3_ADDRESS.toLowerCase()]: true,
  },
};

// Functions on ERC20 tokens that require decimal adjustment
const TOKEN_FUNCTIONS = {
  'transfer': true,
  'transferFrom': true,
  'approve': true,
  'increaseAllowance': true,
  'decreaseAllowance': true
};

// Known function signatures
const KNOWN_SIGNATURES = [
  'transfer(address to,uint256 amount)',
  'transferFrom(address from,address to,uint256 amount)',
  'approve(address spender,uint256 amount)',
  'approveHash(bytes32 hashToApprove)',
  'multiSend(bytes transactions)',
  'increaseAllowance(address spender,uint256 amount)',
  'decreaseAllowance(address spender,uint256 amount)',
  'aggregate3((address target,bool allowFailure,bytes callData)[] calls)',
  'callAgreement(address agreementClass, bytes callData, bytes userData)',
  'createVestingScheduleFromAmountAndDuration(address superToken, address receiver, uint256 totalAmount, uint32 totalDuration, uint32 startDate, uint32 cliffPeriod, uint32 claimPeriod)',
  'propose(address[] targets, uint256[] values, bytes[] calldatas, string description, uint8 proposalType)',
];

// Process KNOWN_SIGNATURES to generate KNOWN_FUNCTIONS
const KNOWN_FUNCTIONS = {};

// Initialize KNOWN_FUNCTIONS
function initKnownFunctions() {
  for (const signature of KNOWN_SIGNATURES) {
    const functionInfo = ethers.utils.FunctionFragment.from(signature);
    const selector = getFunctionSelector(signature);
    KNOWN_FUNCTIONS[selector] = functionInfo;
  }
}

// Helper function to get the function selector from a function signature
function getFunctionSelector(functionSignature) {
  const functionInfo = ethers.utils.FunctionFragment.from(functionSignature);
  return ethers.utils.id(functionInfo.format()).slice(2, 10);
}

// Initialize KNOWN_FUNCTIONS immediately
initKnownFunctions();

// Helper function to get known contract info
function getKnownContract(address, chainID) {
  const normalizedAddr = address.toLowerCase();
  if (KNOWN_CONTRACTS[chainID] && KNOWN_CONTRACTS[chainID][normalizedAddr]) {
    return KNOWN_CONTRACTS[chainID][normalizedAddr];
  }
  return null;
}
