package core

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	// EIP-712 constants
	DomainSeparatorTypehash    = "0x47e79534a245952e8b16893a336b85a3d9ea9fa8c573f3d803afb92a79469218"
	DomainSeparatorTypehashOld = "0x035aff83d86937d35b32e04f0ddc6ff469290eef2f1b692d8a815c89404d4749"
	SafeTxTypehash             = "0xbb8310d486368db6bd6f849402fdd73ad53d316b5a4b2644ad6efe0f941286d8"
	SafeTxTypehashOld          = "0x14d461bc7412367e924637b363c7bf29b8f47e2f84869f4426e5633d8af47b20"

	// Function signatures needed for parseMulticall
	SafeMultisendSig   = "multiSend(bytes)"
	Aggregate3Sig      = "aggregate3((address,bool,bytes)[])"
	Aggregate3ValueSig = "aggregate3Value((address,bool,uint256,bytes)[])"
)

// Known contract addresses
const (
	SafeMultisendAddress     = "0xA1dabEF33b3B82c7814B6D82A79e50F4AC44102B"
	SafeMultisendCallOnly130 = "0x40A2aCCbd92BCA938b02010E17A5b8929b49130D"
	SafeMultisendCallOnly141 = "0x9641d764fc13c8B624c04430C7356C1C7C8102e2"
	Multicall3Address        = "0xcA11bde05977b3631167028862bE2a173976CA11"
	Multicall3Delegatecall   = "0x93dc480940585D9961bfcEab58124fFD3d60f76a"
	USDCMainnetAddress       = "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"
	OPTokenAddress           = "0x4200000000000000000000000000000000000042"
	SuperfluidOP             = "0x1828Bff08BD244F7990edDCd9B19cc654b33cDB4"
	OptimismGovernor         = "0xcDF27F107725988f2261Ce2256bDfCdE8B382B10"
	OPGrants1                = "0x2501c477D0A35545a387Aa4A3EEe4292A9a8B3F0"
	OPGrants2                = "0x19793c7824Be70ec58BB673CA42D2779d12581BE"
	ProxyAdminOwner          = "0x5a0Aae59D09fccBdDb6C6CcEB07B7279367C3d2A"
	OptimismPortal           = "0xbEb5Fc579115071764c7423A4f12eDde41f106Ed"
	OPCMv220Mainnet          = "0x1C7BFA38a25ad22caFC556A9BD827E1da7eC1791"
	OPCMv300Mainnet          = "0x3A1f523a4bc09cd344A2745a108Bb0398288094F"
	OPCMv410Mainnet          = "0x8123739C1368C2DEDc8C564255bc417FEEeBFF9D"
	OPCMv500Mainnet          = "0xFa1Ef97fb02B0dA2Ee2346b8e310907ab5519449"
	OPCMv600Mainnet          = "0x50F47B43c24F40B92C873Fa0704D4207586D0C9f"
	OPCMv700Mainnet          = "0x9Ce712Ff84E02659846dc6450BB9b7642fE8bE5D"
	OPCMv220Sepolia          = "0x6b6f9129efb1b7a48f84e3b787333d1dca02ee34"
	OPCMv300Sepolia          = "0xfBceeD4DE885645fBdED164910E10F52fEBFAB35"
	OPCMv410Sepolia          = "0x3bb6437aba031afbf9cb3538fa064161e2bf2d78"
	OPCMv500Sepolia          = "0xC69e4c24Db479191676611a25D977203c3BDca62"
	OPCMv600Sepolia          = "0xF0a2e224519E876979eA6B2cd15eF5CC3d6703bd"
	OPCMv700Sepolia          = "0x44e197058fb98Fb3618453b3D90CDEf6f5Db8297"
	CCTPv2                   = "0x28b5a0e9C621a5BadaA536219b3a228C8168cf5d"
	OPL1StandardBridge       = "0x99C9fc46f92E8a1c0deC1b1747d010903E884bE1"
	OPL2StandardBridge       = "0x4200000000000000000000000000000000000010"
	SaferSafes               = "0xA8447329e52F64AED2bFc9E7a2506F7D369f483a"
	EtherFiSpokeProxy        = "0xdffcC3536D932eb51Df51a7F5FA407c4270d5308"
	EtherFiSpokeImpl         = "0xA1f75D801633a1941cae6670352d627884dC3b68"
	SafeMigration141         = "0x526643F69b81B008F46d95CD5ced5eC0edFFDaC6"
	SafeMasterCopy141        = "0x41675c099f32341bf84bfc5382af534df5c7461a"
	SafeFallbackHandler141   = "0xfd0732dc9e303f09fcef3a7388ad10a83459ec99"
)

// Functions on ERC20 tokens that require decimal adjustment
var TokenFunctions = map[string]bool{
	"transfer":          true,
	"transferFrom":      true,
	"approve":           true,
	"increaseAllowance": true,
	"decreaseAllowance": true,
}

// ContractInfo stores information about a known contract
type ContractInfo struct {
	Name     string
	Decimals int
}

// FunctionInfo stores information about a known function
type FunctionInfo struct {
	Name      string
	Signature string
	ABI       abi.Method
}

// KnownABIRecord is a reviewed decode entry.  The ABI JSON retains tuple
// components, while Signature and ParameterNames make the rendered call and
// provenance independently reviewable.
type KnownABIRecord struct {
	Signature      string
	ParameterNames []string
	Source         string
	ChainID        uint64
	Address        string
	ABIJSON        string
}

// ChainID constants for supported networks
const (
	MainnetChainID     = 1
	OPMainnetChainID   = 10
	BaseMainnetChainID = 8453
	SepoliaChainID     = 11155111
	OPSepoliaChainID   = 11155420
	BaseSepoliaChainID = 84532
)

// ChainNames maps chain IDs to their names
var ChainNames = map[uint64]string{
	MainnetChainID:     "Ethereum",
	OPMainnetChainID:   "OP Mainnet",
	SepoliaChainID:     "Sepolia",
	OPSepoliaChainID:   "OP Sepolia",
	BaseMainnetChainID: "Base Mainnet",
	BaseSepoliaChainID: "Base Sepolia",
}

// KnownContracts maps chain IDs to a map of addresses to contract info
var KnownContracts = map[uint64]map[string]ContractInfo{
	MainnetChainID: {
		strings.ToLower(SafeMultisendAddress):     {Name: "GNOSIS SAFE MULTISEND", Decimals: 0},
		strings.ToLower(SafeMultisendCallOnly130): {Name: "GNOSIS SAFE MULTISEND (v1.3.0)", Decimals: 0},
		strings.ToLower(SafeMultisendCallOnly141): {Name: "GNOSIS SAFE MULTISEND (v1.4.1)", Decimals: 0},
		strings.ToLower(Multicall3Address):        {Name: "MULTICALL3", Decimals: 0},
		strings.ToLower(Multicall3Delegatecall):   {Name: "MULTICALL3 DELEGATECALL", Decimals: 0},
		strings.ToLower(USDCMainnetAddress):       {Name: "USDC", Decimals: 6},
		strings.ToLower(ProxyAdminOwner):          {Name: "SUPERCHAIN PROXY ADMIN OWNER", Decimals: 0},
		strings.ToLower(OptimismPortal):           {Name: "OPTIMISM PORTAL", Decimals: 0},
		strings.ToLower(OPCMv220Mainnet):          {Name: "OPContractsManager V2.2.0", Decimals: 0},
		strings.ToLower(OPCMv300Mainnet):          {Name: "OPContractsManager V3.0.0", Decimals: 0},
		strings.ToLower(OPCMv410Mainnet):          {Name: "OPContractsManager V4.1.0", Decimals: 0},
		strings.ToLower(OPCMv500Mainnet):          {Name: "OPContractsManager V5.0.0", Decimals: 0},
		strings.ToLower(OPCMv600Mainnet):          {Name: "OPContractsManager V6.0.0", Decimals: 0},
		strings.ToLower(OPCMv700Mainnet):          {Name: "OPContractsManager V7.0.0", Decimals: 0},
		strings.ToLower(CCTPv2):                   {Name: "CCTP V2", Decimals: 0},
		strings.ToLower(OPL1StandardBridge):       {Name: "OP L1StandardBridge", Decimals: 0},
		strings.ToLower(SaferSafes):               {Name: "SaferSafes", Decimals: 0},
		strings.ToLower(SafeMigration141):         {Name: "Safe Migration Contract (v1.4.1)", Decimals: 0},
		strings.ToLower(SafeMasterCopy141):        {Name: "Safe Master Copy (v1.4.1)", Decimals: 0},
		strings.ToLower(SafeFallbackHandler141):   {Name: "Safe Fallback Handler (v1.4.1)", Decimals: 0},
	},
	OPMainnetChainID: {
		strings.ToLower(SafeMultisendAddress):     {Name: "GNOSIS SAFE MULTISEND", Decimals: 0},
		strings.ToLower(SafeMultisendCallOnly141): {Name: "GNOSIS SAFE MULTISEND", Decimals: 0},
		strings.ToLower(SafeMultisendCallOnly130): {Name: "GNOSIS SAFE MULTISEND", Decimals: 0},
		strings.ToLower(Multicall3Address):        {Name: "MULTICALL3", Decimals: 0},
		strings.ToLower(Multicall3Delegatecall):   {Name: "MULTICALL3 DELEGATECALL", Decimals: 0},
		strings.ToLower(OPTokenAddress):           {Name: "OP TOKEN", Decimals: 18},
		strings.ToLower(SuperfluidOP):             {Name: "SUPERFLUID OP", Decimals: 18},
		strings.ToLower(OptimismGovernor):         {Name: "OPTIMISM GOVERNOR", Decimals: 0},
		strings.ToLower(OPGrants1):                {Name: "OP GRANTS 1 (3F0)", Decimals: 0},
		strings.ToLower(OPGrants2):                {Name: "OP GRANTS 2 (1BE)", Decimals: 0},
		strings.ToLower(OPL2StandardBridge):       {Name: "OP L2StandardBridge", Decimals: 0},
		strings.ToLower(SaferSafes):               {Name: "SaferSafes", Decimals: 0},
		strings.ToLower(EtherFiSpokeProxy):        {Name: "ETHERFI SPOKE (PROXY)", Decimals: 0},
		strings.ToLower(EtherFiSpokeImpl):         {Name: "ETHERFI SPOKE (IMPLEMENTATION)", Decimals: 0},
	},
	BaseMainnetChainID: {
		strings.ToLower(SafeMultisendAddress):     {Name: "GNOSIS SAFE MULTISEND", Decimals: 0},
		strings.ToLower(SafeMultisendCallOnly141): {Name: "GNOSIS SAFE MULTISEND", Decimals: 0},
		strings.ToLower(SafeMultisendCallOnly130): {Name: "GNOSIS SAFE MULTISEND", Decimals: 0},
		strings.ToLower(Multicall3Address):        {Name: "MULTICALL3", Decimals: 0},
		strings.ToLower(Multicall3Delegatecall):   {Name: "MULTICALL3 DELEGATECALL", Decimals: 0},
		strings.ToLower(OPL2StandardBridge):       {Name: "Base L2StandardBridge", Decimals: 0},
	},
	SepoliaChainID: {
		strings.ToLower(SafeMultisendAddress):   {Name: "GNOSIS SAFE MULTISEND", Decimals: 0},
		strings.ToLower(Multicall3Address):      {Name: "MULTICALL3", Decimals: 0},
		strings.ToLower(Multicall3Delegatecall): {Name: "MULTICALL3 DELEGATECALL", Decimals: 0},
		strings.ToLower(OPCMv220Sepolia):        {Name: "OPContractsManager V2.2.0", Decimals: 0},
		strings.ToLower(OPCMv300Sepolia):        {Name: "OPContractsManager V3.0.0", Decimals: 0},
		strings.ToLower(OPCMv410Sepolia):        {Name: "OPContractsManager V4.1.0", Decimals: 0},
		strings.ToLower(OPCMv500Sepolia):        {Name: "OPContractsManager V5.0.0", Decimals: 0},
		strings.ToLower(OPCMv600Sepolia):        {Name: "OPContractsManager V6.0.0", Decimals: 0},
		strings.ToLower(OPCMv700Sepolia):        {Name: "OPContractsManager V7.0.0", Decimals: 0},
		strings.ToLower(SaferSafes):             {Name: "SaferSafes", Decimals: 0},
	},
	OPSepoliaChainID: {
		strings.ToLower(SafeMultisendAddress):   {Name: "GNOSIS SAFE MULTISEND", Decimals: 0},
		strings.ToLower(Multicall3Address):      {Name: "MULTICALL3", Decimals: 0},
		strings.ToLower(Multicall3Delegatecall): {Name: "MULTICALL3 DELEGATECALL", Decimals: 0},
	},
}

// MulticallAddresses maps chain IDs to a set of addresses known to be multicall contracts
var MulticallAddresses = map[uint64]map[string]bool{
	MainnetChainID: {
		strings.ToLower(SafeMultisendAddress):     true,
		strings.ToLower(SafeMultisendCallOnly130): true,
		strings.ToLower(SafeMultisendCallOnly141): true,
		strings.ToLower(Multicall3Address):        true,
		strings.ToLower(Multicall3Delegatecall):   true,
	},
	OPMainnetChainID: {
		strings.ToLower(SafeMultisendAddress):     true,
		strings.ToLower(SafeMultisendCallOnly141): true,
		strings.ToLower(Multicall3Address):        true,
		strings.ToLower(Multicall3Delegatecall):   true,
	},
	SepoliaChainID: {
		strings.ToLower(SafeMultisendAddress):   true,
		strings.ToLower(Multicall3Address):      true,
		strings.ToLower(Multicall3Delegatecall): true,
	},
	OPSepoliaChainID: {
		strings.ToLower(SafeMultisendAddress):   true,
		strings.ToLower(Multicall3Address):      true,
		strings.ToLower(Multicall3Delegatecall): true,
	},
}

// KnownFunctions maps canonical selector hex to its reviewed ABI method.
var KnownFunctions = make(map[string]FunctionInfo)

// legacyABIRecords preserves the previously supported decode corpus in the
// same reviewed record shape as history additions.
var legacyABIRecords = []KnownABIRecord{
	{Signature: "transfer(address,uint256)", ParameterNames: []string{"to", "amount"}, Source: "existing local signature database", ABIJSON: `[{"inputs":[{"name":"to","type":"address"},{"name":"amount","type":"uint256"}],"name":"transfer","type":"function"}]`},
	{Signature: "transferFrom(address,address,uint256)", ParameterNames: []string{"from", "to", "amount"}, Source: "existing local signature database", ABIJSON: `[{"inputs":[{"name":"from","type":"address"},{"name":"to","type":"address"},{"name":"amount","type":"uint256"}],"name":"transferFrom","type":"function"}]`},
	{Signature: "approve(address,uint256)", ParameterNames: []string{"spender", "amount"}, Source: "existing local signature database", ABIJSON: `[{"inputs":[{"name":"spender","type":"address"},{"name":"amount","type":"uint256"}],"name":"approve","type":"function"}]`},
	{Signature: "increaseAllowance(address,uint256)", ParameterNames: []string{"spender", "amount"}, Source: "existing local signature database", ABIJSON: `[{"inputs":[{"name":"spender","type":"address"},{"name":"amount","type":"uint256"}],"name":"increaseAllowance","type":"function"}]`},
	{Signature: "decreaseAllowance(address,uint256)", ParameterNames: []string{"spender", "amount"}, Source: "existing local signature database", ABIJSON: `[{"inputs":[{"name":"spender","type":"address"},{"name":"amount","type":"uint256"}],"name":"decreaseAllowance","type":"function"}]`},
	{Signature: "approveHash(bytes32)", ParameterNames: []string{"hashToApprove"}, Source: "existing local signature database", ABIJSON: `[{"inputs":[{"name":"hashToApprove","type":"bytes32"}],"name":"approveHash","type":"function"}]`},
	{Signature: "aggregate3((address,bool,bytes)[])", ParameterNames: []string{"calls"}, Source: "existing local signature database", ABIJSON: `[{"inputs":[{"name":"calls","type":"tuple[]","components":[{"name":"target","type":"address"},{"name":"allowFailure","type":"bool"},{"name":"callData","type":"bytes"}]}],"name":"aggregate3","type":"function"}]`},
	{Signature: "aggregate3Value((address,bool,uint256,bytes)[])", ParameterNames: []string{"calls"}, Source: "existing local signature database", ABIJSON: `[{"inputs":[{"name":"calls","type":"tuple[]","components":[{"name":"target","type":"address"},{"name":"allowFailure","type":"bool"},{"name":"value","type":"uint256"},{"name":"callData","type":"bytes"}]}],"name":"aggregate3Value","type":"function"}]`},
	{Signature: "multiSend(bytes)", ParameterNames: []string{"transactions"}, Source: "existing local signature database", ABIJSON: `[{"inputs":[{"name":"transactions","type":"bytes"}],"name":"multiSend","type":"function"}]`},
	{Signature: "callAgreement(address,bytes,bytes)", ParameterNames: []string{"agreementClass", "callData", "userData"}, Source: "existing local signature database", ABIJSON: `[{"inputs":[{"name":"agreementClass","type":"address"},{"name":"callData","type":"bytes"},{"name":"userData","type":"bytes"}],"name":"callAgreement","type":"function"}]`},
	{Signature: "createVestingScheduleFromAmountAndDuration(address,address,uint256,uint32,uint32,uint32,uint32)", ParameterNames: []string{"superToken", "receiver", "totalAmount", "totalDuration", "startDate", "cliffPeriod", "claimPeriod"}, Source: "existing local signature database", ABIJSON: `[{"inputs":[{"name":"superToken","type":"address"},{"name":"receiver","type":"address"},{"name":"totalAmount","type":"uint256"},{"name":"totalDuration","type":"uint32"},{"name":"startDate","type":"uint32"},{"name":"cliffPeriod","type":"uint32"},{"name":"claimPeriod","type":"uint32"}],"name":"createVestingScheduleFromAmountAndDuration","type":"function"}]`},
	{Signature: "propose(address[],uint256[],bytes[],string,uint8)", ParameterNames: []string{"targets", "values", "calldatas", "description", "proposalType"}, Source: "existing local signature database", ABIJSON: `[{"inputs":[{"name":"targets","type":"address[]"},{"name":"values","type":"uint256[]"},{"name":"calldatas","type":"bytes[]"},{"name":"description","type":"string"},{"name":"proposalType","type":"uint8"}],"name":"propose","type":"function"}]`},
	{Signature: "upgrade((address,address,bytes32)[])", ParameterNames: []string{"opChainConfigs"}, Source: "existing local signature database", ABIJSON: `[{"inputs":[{"name":"opChainConfigs","type":"tuple[]","components":[{"name":"systemConfigProxy","type":"address"},{"name":"proxyAdmin","type":"address"},{"name":"absolutePrestate","type":"bytes32"}]}],"name":"upgrade","type":"function"}]`},
	{Signature: "attest((bytes32,(address,uint64,bool,bytes32,bytes,uint256)))", ParameterNames: []string{"request"}, Source: "existing local signature database", ABIJSON: `[{"inputs":[{"components":[{"name":"schema","type":"bytes32"},{"components":[{"name":"recipient","type":"address"},{"name":"expirationTime","type":"uint64"},{"name":"revocable","type":"bool"},{"name":"refUID","type":"bytes32"},{"name":"data","type":"bytes"},{"name":"value","type":"uint256"}],"name":"data","type":"tuple"}],"name":"request","type":"tuple"}],"name":"attest","type":"function"}]`},
	{Signature: "addOwnerWithThreshold(address,uint256)", ParameterNames: []string{"owner", "threshold"}, Source: "existing local signature database", ABIJSON: `[{"inputs":[{"name":"owner","type":"address"},{"name":"threshold","type":"uint256"}],"name":"addOwnerWithThreshold","type":"function"}]`},
	{Signature: "removeOwner(address,address,uint256)", ParameterNames: []string{"prevOwner", "owner", "threshold"}, Source: "existing local signature database", ABIJSON: `[{"inputs":[{"name":"prevOwner","type":"address"},{"name":"owner","type":"address"},{"name":"threshold","type":"uint256"}],"name":"removeOwner","type":"function"}]`},
	{Signature: "execTransaction(address,uint256,bytes,uint8,uint256,uint256,uint256,address,address,bytes)", ParameterNames: []string{"to", "value", "data", "operation", "safeTxGas", "baseGas", "gasPrice", "gasToken", "refundReceiver", "signatures"}, Source: "existing local signature database", ABIJSON: `[{"inputs":[{"name":"to","type":"address"},{"name":"value","type":"uint256"},{"name":"data","type":"bytes"},{"name":"operation","type":"uint8"},{"name":"safeTxGas","type":"uint256"},{"name":"baseGas","type":"uint256"},{"name":"gasPrice","type":"uint256"},{"name":"gasToken","type":"address"},{"name":"refundReceiver","type":"address"},{"name":"signatures","type":"bytes"}],"name":"execTransaction","type":"function"}]`},
	{Signature: "depositTransaction(address,uint256,uint64,bool,bytes)", ParameterNames: []string{"to", "value", "gasLimit", "isCreation", "data"}, Source: "existing local signature database", ABIJSON: `[{"inputs":[{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"value","type":"uint256"},{"internalType":"uint64","name":"gasLimit","type":"uint64"},{"internalType":"bool","name":"isCreation","type":"bool"},{"internalType":"bytes","name":"data","type":"bytes"}],"name":"depositTransaction","outputs":[],"stateMutability":"payable","type":"function"}]`},
	{Signature: "swapOwner(address,address,address)", ParameterNames: []string{"prevOwner", "oldOwner", "newOwner"}, Source: "existing local signature database", ABIJSON: `[{"inputs":[{"name":"prevOwner","type":"address"},{"name":"oldOwner","type":"address"},{"name":"newOwner","type":"address"}],"name":"swapOwner","type":"function"}]`},
	{Signature: "updatePrestate((address,address,bytes32)[])", ParameterNames: []string{"prestateUpdateInputs"}, Source: "existing local signature database", ABIJSON: `[{"inputs":[{"components":[{"name":"systemConfigProxy","type":"address"},{"name":"proxyAdmin","type":"address"},{"name":"absolutePrestate","type":"bytes32"}],"name":"prestateUpdateInputs","type":"tuple[]"}],"name":"updatePrestate","outputs":[],"stateMutability":"nonpayable","type":"function"}]`},
	{Signature: "depositForBurn(uint256,uint32,bytes32,address,bytes32,uint256,uint32)", ParameterNames: []string{"amount", "destinationDomain", "mintRecipient", "burnToken", "destinationCaller", "maxFee", "minFinalityThreshold"}, Source: "existing local signature database", ABIJSON: `[{"inputs":[{"name":"amount","type":"uint256"},{"name":"destinationDomain","type":"uint32"},{"name":"mintRecipient","type":"bytes32"},{"name":"burnToken","type":"address"},{"name":"destinationCaller","type":"bytes32"},{"name":"maxFee","type":"uint256"},{"name":"minFinalityThreshold","type":"uint32"}],"name":"depositForBurn","outputs":[],"stateMutability":"nonpayable","type":"function"}]`},
	{Signature: "bridgeETHTo(address,uint32,bytes)", ParameterNames: []string{"to", "minGasLimit", "extraData"}, Source: "existing local signature database", ABIJSON: `[{"inputs":[{"internalType":"address","name":"to","type":"address"},{"internalType":"uint32","name":"minGasLimit","type":"uint32"},{"internalType":"bytes","name":"extraData","type":"bytes"}],"name":"bridgeETHTo","outputs":[],"stateMutability":"payable","type":"function"}]`},
	{Signature: "upgradeSuperchainConfig(address,address)", ParameterNames: []string{"superchainConfig", "superchainProxyAdmin"}, Source: "existing local signature database", ABIJSON: `[{"inputs":[{"name":"superchainConfig","type":"address"},{"name":"superchainProxyAdmin","type":"address"}],"name":"upgradeSuperchainConfig","outputs":[],"stateMutability":"nonpayable","type":"function"}]`},
	{Signature: "upgrade((address,(bool,uint256,uint32,bytes)[],(string,bytes)[]))", ParameterNames: []string{"_inp"}, Source: "existing local signature database", ABIJSON: `[{"inputs":[{"name":"_inp","type":"tuple","components":[{"name":"systemConfig","type":"address"},{"name":"disputeGameConfigs","type":"tuple[]","components":[{"name":"enabled","type":"bool"},{"name":"initBond","type":"uint256"},{"name":"gameType","type":"uint32"},{"name":"gameArgs","type":"bytes"}]},{"name":"extraInstructions","type":"tuple[]","components":[{"name":"key","type":"string"},{"name":"data","type":"bytes"}]}]}],"name":"upgrade","outputs":[],"stateMutability":"nonpayable","type":"function"}]`},
	{Signature: "upgradeSuperchain((address,(string,bytes)[]))", ParameterNames: []string{"_inp"}, Source: "existing local signature database", ABIJSON: `[{"inputs":[{"name":"_inp","type":"tuple","components":[{"name":"superchainConfig","type":"address"},{"name":"extraInstructions","type":"tuple[]","components":[{"name":"key","type":"string"},{"name":"data","type":"bytes"}]}]}],"name":"upgradeSuperchain","outputs":[],"stateMutability":"nonpayable","type":"function"}]`},
	{Signature: "migrate((address[],(bool,uint256,uint32,bytes)[],(bytes32,uint256),uint32))", ParameterNames: []string{"_input"}, Source: "existing local signature database", ABIJSON: `[{"inputs":[{"name":"_input","type":"tuple","components":[{"name":"chainSystemConfigs","type":"address[]"},{"name":"disputeGameConfigs","type":"tuple[]","components":[{"name":"enabled","type":"bool"},{"name":"initBond","type":"uint256"},{"name":"gameType","type":"uint32"},{"name":"gameArgs","type":"bytes"}]},{"name":"startingAnchorRoot","type":"tuple","components":[{"name":"root","type":"bytes32"},{"name":"l2SequenceNumber","type":"uint256"}]},{"name":"startingRespectedGameType","type":"uint32"}]}],"name":"migrate","outputs":[],"stateMutability":"nonpayable","type":"function"}]`},
	{Signature: "upgrade((address,bytes32,bytes32)[])", ParameterNames: []string{"_opChainConfigs"}, Source: "existing local signature database", ABIJSON: `[{"inputs":[{"name":"_opChainConfigs","type":"tuple[]","components":[{"name":"systemConfigProxy","type":"address"},{"name":"cannonPrestate","type":"bytes32"},{"name":"cannonKonaPrestate","type":"bytes32"}]}],"name":"upgrade","outputs":[],"stateMutability":"nonpayable","type":"function"}]`},
	{Signature: "updatePrestate((address,bytes32,bytes32)[])", ParameterNames: []string{"_prestateUpdateInputs"}, Source: "existing local signature database", ABIJSON: `[{"inputs":[{"name":"_prestateUpdateInputs","type":"tuple[]","components":[{"name":"systemConfigProxy","type":"address"},{"name":"cannonPrestate","type":"bytes32"},{"name":"cannonKonaPrestate","type":"bytes32"}]}],"name":"updatePrestate","outputs":[],"stateMutability":"nonpayable","type":"function"}]`},
	{Signature: "addGameType((string,address,address,uint32,bytes32,uint256,uint256,uint64,uint64,uint256,address,bool)[])", ParameterNames: []string{"_gameConfigs"}, Source: "existing local signature database", ABIJSON: `[{"inputs":[{"name":"_gameConfigs","type":"tuple[]","components":[{"name":"saltMixer","type":"string"},{"name":"systemConfig","type":"address"},{"name":"delayedWETH","type":"address"},{"name":"disputeGameType","type":"uint32"},{"name":"disputeAbsolutePrestate","type":"bytes32"},{"name":"disputeMaxGameDepth","type":"uint256"},{"name":"disputeSplitDepth","type":"uint256"},{"name":"disputeClockExtension","type":"uint64"},{"name":"disputeMaxClockDuration","type":"uint64"},{"name":"initialBond","type":"uint256"},{"name":"vm","type":"address"},{"name":"permissioned","type":"bool"}]}],"name":"addGameType","outputs":[],"stateMutability":"nonpayable","type":"function"}]`},
	{Signature: "migrate((bool,(bytes32,uint256),(address,address,uint256,uint256,uint256,uint64,uint64),(address,bytes32,bytes32)[]))", ParameterNames: []string{"_input"}, Source: "existing local signature database", ABIJSON: `[{"inputs":[{"name":"_input","type":"tuple","components":[{"name":"usePermissionlessGame","type":"bool"},{"name":"startingAnchorRoot","type":"tuple","components":[{"name":"root","type":"bytes32"},{"name":"l2SequenceNumber","type":"uint256"}]},{"name":"gameParameters","type":"tuple","components":[{"name":"proposer","type":"address"},{"name":"challenger","type":"address"},{"name":"maxGameDepth","type":"uint256"},{"name":"splitDepth","type":"uint256"},{"name":"initBond","type":"uint256"},{"name":"clockExtension","type":"uint64"},{"name":"maxClockDuration","type":"uint64"}]},{"name":"opChainConfigs","type":"tuple[]","components":[{"name":"systemConfigProxy","type":"address"},{"name":"cannonPrestate","type":"bytes32"},{"name":"cannonKonaPrestate","type":"bytes32"}]}]}],"name":"migrate","outputs":[],"stateMutability":"nonpayable","type":"function"}]`},
	{Signature: "signCancellation(bytes32)", ParameterNames: []string{"safeTxHash"}, Source: "existing local signature database", ABIJSON: `[{"inputs":[{"internalType": "bytes32","name": "safeTxHash","type": "bytes32"}],"name": "signCancellation","outputs": [],"stateMutability": "nonpayable","type": "function"}]`},
	{Signature: "challenge(address)", ParameterNames: []string{"safe"}, Source: "existing local signature database", ABIJSON: `[{"inputs":[{"internalType": "contract Safe","name": "safe","type": "address"}],"name": "challenge","outputs": [],"stateMutability": "nonpayable","type": "function"}]`},
	{Signature: "respond()", ParameterNames: []string{}, Source: "existing local signature database", ABIJSON: `[{"inputs":[],"name": "respond","outputs": [],"stateMutability": "nonpayable","type": "function"}]`},
	{Signature: "changeOwnershipToFallback(address)", ParameterNames: []string{"safe"}, Source: "existing local signature database", ABIJSON: `[{"inputs":[{"internalType": "contract Safe","name": "safe","type": "address"}],"name": "changeOwnershipToFallback","outputs": [],"stateMutability": "nonpayable","type": "function"}]`},
	{Signature: "setFallbackHandler(address)", ParameterNames: []string{"handler"}, Source: "existing local signature database", ABIJSON: `[{"inputs":[{"name":"handler","type":"address"}],"name":"setFallbackHandler","outputs":[],"stateMutability":"nonpayable","type":"function"}]`},
	{Signature: "changeMasterCopy(address)", ParameterNames: []string{"masterCopy"}, Source: "existing local signature database", ABIJSON: `[{"inputs":[{"name":"masterCopy","type":"address"}],"name":"changeMasterCopy","outputs":[],"stateMutability":"nonpayable","type":"function"}]`},
	{Signature: "migrateWithFallbackHandler()", ParameterNames: []string{}, Source: "existing local signature database", ABIJSON: `[{"inputs":[],"name":"migrateWithFallbackHandler","outputs":[],"stateMutability":"nonpayable","type":"function"}]`},
}

const safeHistoryReviewSource = "Safe history review: .superpowers/sdd/op-txverify-followup-plan/task-1-brief.md"

// historyABIRecords contains the reviewed additions from the Safe history.
// Scope is provenance only: selector decoding intentionally remains global,
// because an address can be reached through a proxy or a delegatecall.
var historyABIRecords = []KnownABIRecord{
	{Signature: "setRequired(uint256)", ParameterNames: []string{"required"}, Source: safeHistoryReviewSource, ABIJSON: `[{"inputs":[{"name":"required","type":"uint256"}],"name":"setRequired","type":"function"}]`},
	{Signature: "configureLivenessModule((uint256,address))", ParameterNames: []string{"config"}, Source: safeHistoryReviewSource, ABIJSON: `[{"inputs":[{"name":"config","type":"tuple","components":[{"name":"required","type":"uint256"},{"name":"livenessModule","type":"address"}]}],"name":"configureLivenessModule","type":"function"}]`},
	{Signature: "execute(bytes)", ParameterNames: []string{"data"}, Source: safeHistoryReviewSource, ABIJSON: `[{"inputs":[{"name":"data","type":"bytes"}],"name":"execute","type":"function"}]`},
	{Signature: "withdraw(uint256,uint256,address)", ParameterNames: []string{"assets", "shares", "receiver"}, Source: safeHistoryReviewSource, ABIJSON: `[{"inputs":[{"name":"assets","type":"uint256"},{"name":"shares","type":"uint256"},{"name":"receiver","type":"address"}],"name":"withdraw","type":"function"}]`},
	{Signature: "setOwner(address)", ParameterNames: []string{"owner"}, Source: safeHistoryReviewSource, ABIJSON: `[{"inputs":[{"name":"owner","type":"address"}],"name":"setOwner","type":"function"}]`},
	{Signature: "setImplementation(uint32,address)", ParameterNames: []string{"gameType", "implementation"}, Source: safeHistoryReviewSource, ABIJSON: `[{"inputs":[{"name":"gameType","type":"uint32"},{"name":"implementation","type":"address"}],"name":"setImplementation","type":"function"}]`},
	{Signature: "createProxyWithNonce(address,bytes,uint256)", ParameterNames: []string{"singleton", "initializer", "saltNonce"}, Source: safeHistoryReviewSource, ABIJSON: `[{"inputs":[{"name":"singleton","type":"address"},{"name":"initializer","type":"bytes"},{"name":"saltNonce","type":"uint256"}],"name":"createProxyWithNonce","type":"function"}]`},
	{Signature: "migrateEth(address)", ParameterNames: []string{"recipient"}, Source: safeHistoryReviewSource, ABIJSON: `[{"inputs":[{"name":"recipient","type":"address"}],"name":"migrateEth","type":"function"}]`},
	{Signature: "setDeputy(address,bytes)", ParameterNames: []string{"deputy", "permissions"}, Source: safeHistoryReviewSource, ABIJSON: `[{"inputs":[{"name":"deputy","type":"address"},{"name":"permissions","type":"bytes"}],"name":"setDeputy","type":"function"}]`},
	{Signature: "setUnsafeBlockSigner(address)", ParameterNames: []string{"signer"}, Source: safeHistoryReviewSource, ABIJSON: `[{"inputs":[{"name":"signer","type":"address"}],"name":"setUnsafeBlockSigner","type":"function"}]`},
	{Signature: "setInitBond(uint32,uint256)", ParameterNames: []string{"gameType", "initBond"}, Source: safeHistoryReviewSource, ABIJSON: `[{"inputs":[{"name":"gameType","type":"uint32"},{"name":"initBond","type":"uint256"}],"name":"setInitBond","type":"function"}]`},
	{Signature: "unpause()", Source: safeHistoryReviewSource, ABIJSON: `[{"inputs":[],"name":"unpause","type":"function"}]`},
	{Signature: "initialize(address,address)", ParameterNames: []string{"owner", "guardian"}, Source: safeHistoryReviewSource, ABIJSON: `[{"inputs":[{"name":"owner","type":"address"},{"name":"guardian","type":"address"}],"name":"initialize","type":"function"}]`},
	{Signature: "setBytes32(bytes32,bytes32)", ParameterNames: []string{"key", "value"}, Source: safeHistoryReviewSource, ABIJSON: `[{"inputs":[{"name":"key","type":"bytes32"},{"name":"value","type":"bytes32"}],"name":"setBytes32","type":"function"}]`},
	{Signature: "setRecommended(uint256)", ParameterNames: []string{"required"}, Source: safeHistoryReviewSource, ABIJSON: `[{"inputs":[{"name":"required","type":"uint256"}],"name":"setRecommended","type":"function"}]`},
	{Signature: "enableModule(address)", ParameterNames: []string{"module"}, Source: safeHistoryReviewSource, ABIJSON: `[{"inputs":[{"name":"module","type":"address"}],"name":"enableModule","type":"function"}]`},
	{Signature: "changeThreshold(uint256)", ParameterNames: []string{"threshold"}, Source: safeHistoryReviewSource, ABIJSON: `[{"inputs":[{"name":"threshold","type":"uint256"}],"name":"changeThreshold","type":"function"}]`},
	{Signature: "pause()", Source: safeHistoryReviewSource, ABIJSON: `[{"inputs":[],"name":"pause","type":"function"}]`},
	{Signature: "supply(uint256,uint256,address)", ParameterNames: []string{"assets", "shares", "receiver"}, Source: safeHistoryReviewSource, ABIJSON: `[{"inputs":[{"name":"assets","type":"uint256"},{"name":"shares","type":"uint256"},{"name":"receiver","type":"address"}],"name":"supply","type":"function"}]`},
	{Signature: "initialize(address,address,address,uint32)", ParameterNames: []string{"owner", "feeRecipient", "oracle", "gasLimit"}, Source: safeHistoryReviewSource, ABIJSON: `[{"inputs":[{"name":"owner","type":"address"},{"name":"feeRecipient","type":"address"},{"name":"oracle","type":"address"},{"name":"gasLimit","type":"uint32"}],"name":"initialize","type":"function"}]`},
	{Signature: "changeAdmin(address)", ParameterNames: []string{"admin"}, Source: safeHistoryReviewSource, ABIJSON: `[{"inputs":[{"name":"admin","type":"address"}],"name":"changeAdmin","type":"function"}]`},
	{Signature: "setGasConfig(uint256,uint256)", ParameterNames: []string{"gasLimit", "baseFee"}, Source: safeHistoryReviewSource, ABIJSON: `[{"inputs":[{"name":"gasLimit","type":"uint256"},{"name":"baseFee","type":"uint256"}],"name":"setGasConfig","type":"function"}]`},
	{Signature: "upgradeAndCall(address,address,bytes)", ParameterNames: []string{"proxy", "implementation", "data"}, Source: safeHistoryReviewSource, ABIJSON: `[{"inputs":[{"name":"proxy","type":"address"},{"name":"implementation","type":"address"},{"name":"data","type":"bytes"}],"name":"upgradeAndCall","type":"function"}]`},
	{Signature: "upgrade(address,address)", ParameterNames: []string{"proxy", "implementation"}, Source: safeHistoryReviewSource, ABIJSON: `[{"inputs":[{"name":"proxy","type":"address"},{"name":"implementation","type":"address"}],"name":"upgrade","type":"function"}]`},
	{Signature: "setAddress(string,address)", ParameterNames: []string{"key", "value"}, Source: safeHistoryReviewSource, ABIJSON: `[{"inputs":[{"name":"key","type":"string"},{"name":"value","type":"address"}],"name":"setAddress","type":"function"}]`},
	{Signature: "setRespectedGameType(address,uint32)", ParameterNames: []string{"disputeGameFactory", "gameType"}, Source: safeHistoryReviewSource, ABIJSON: `[{"inputs":[{"name":"disputeGameFactory","type":"address"},{"name":"gameType","type":"uint32"}],"name":"setRespectedGameType","type":"function"}]`},
	{Signature: "setImplementation(uint32,address,bytes)", ParameterNames: []string{"gameType", "implementation", "initData"}, Source: safeHistoryReviewSource, ABIJSON: `[{"inputs":[{"name":"gameType","type":"uint32"},{"name":"implementation","type":"address"},{"name":"initData","type":"bytes"}],"name":"setImplementation","type":"function"}]`},
	{Signature: "setGasLimit(uint64)", ParameterNames: []string{"gasLimit"}, Source: safeHistoryReviewSource, ABIJSON: `[{"inputs":[{"name":"gasLimit","type":"uint64"}],"name":"setGasLimit","type":"function"}]`},
	{Signature: "setEIP1559Params(uint32,uint32)", ParameterNames: []string{"denominator", "elasticity"}, Source: safeHistoryReviewSource, ABIJSON: `[{"inputs":[{"name":"denominator","type":"uint32"},{"name":"elasticity","type":"uint32"}],"name":"setEIP1559Params","type":"function"}]`},
	{Signature: "setBatcherHash(bytes32)", ParameterNames: []string{"batcherHash"}, Source: safeHistoryReviewSource, ABIJSON: `[{"inputs":[{"name":"batcherHash","type":"bytes32"}],"name":"setBatcherHash","type":"function"}]`},
	{Signature: "setAddress(bytes32,address)", ParameterNames: []string{"key", "value"}, Source: safeHistoryReviewSource, ABIJSON: `[{"inputs":[{"name":"key","type":"bytes32"},{"name":"value","type":"address"}],"name":"setAddress","type":"function"}]`},
	{Signature: "updateDynamicConfig((uint256,uint256),bool)", ParameterNames: []string{"config", "isEcotone"}, Source: safeHistoryReviewSource, ABIJSON: `[{"inputs":[{"name":"config","type":"tuple","components":[{"name":"maxSequencerDrift","type":"uint256"},{"name":"sequencerWindowSize","type":"uint256"}]},{"name":"isEcotone","type":"bool"}],"name":"updateDynamicConfig","type":"function"}]`},
	{Signature: "phase2()", Source: safeHistoryReviewSource, ABIJSON: `[{"inputs":[],"name":"phase2","type":"function"}]`},
	{Signature: "disableModule(address,address)", ParameterNames: []string{"prevModule", "module"}, Source: safeHistoryReviewSource, ABIJSON: `[{"inputs":[{"name":"prevModule","type":"address"},{"name":"module","type":"address"}],"name":"disableModule","type":"function"}]`},
	{Signature: "setGuard(address)", ParameterNames: []string{"guard"}, Source: safeHistoryReviewSource, ABIJSON: `[{"inputs":[{"name":"guard","type":"address"}],"name":"setGuard","type":"function"}]`},
	{Signature: "phase1()", Source: safeHistoryReviewSource, ABIJSON: `[{"inputs":[],"name":"phase1","type":"function"}]`},
	{Signature: "transferOwnership(address)", ParameterNames: []string{"newOwner"}, Source: safeHistoryReviewSource, ABIJSON: `[{"inputs":[{"name":"newOwner","type":"address"}],"name":"transferOwnership","type":"function"}]`},
}

// KnownABIRecords is the complete, structured local signature database.
var KnownABIRecords = append(legacyABIRecords, historyABIRecords...)

// Initialize known functions
func init() {
	functions, err := BuildKnownFunctions(KnownABIRecords)
	if err != nil {
		panic(fmt.Sprintf("build known function selector map: %v", err))
	}
	KnownFunctions = functions
}

// BuildKnownFunctions builds the selector map from reviewed ABI records. A
// conflicting collision is unsafe: the old last-write-wins behavior could show
// callers a different method than their calldata selects.
func BuildKnownFunctions(records []KnownABIRecord) (map[string]FunctionInfo, error) {
	functions := make(map[string]FunctionInfo, len(records))
	for _, record := range records {
		if record.Signature == "" || record.Source == "" {
			return nil, fmt.Errorf("record must include signature and source")
		}
		parsedABI, err := abi.JSON(strings.NewReader(record.ABIJSON))
		if err != nil {
			return nil, fmt.Errorf("parse %s ABI: %w", record.Signature, err)
		}

		for _, method := range parsedABI.Methods {
			if method.Sig != record.Signature {
				return nil, fmt.Errorf("ABI signature %s does not match record signature %s", method.Sig, record.Signature)
			}
			if len(method.Inputs) != len(record.ParameterNames) {
				return nil, fmt.Errorf("parameter names for %s do not match ABI", record.Signature)
			}
			for i, input := range method.Inputs {
				if input.Name != record.ParameterNames[i] {
					return nil, fmt.Errorf("parameter %d for %s is %q, want %q", i, record.Signature, input.Name, record.ParameterNames[i])
				}
			}

			selector := hex.EncodeToString(crypto.Keccak256([]byte(record.Signature))[:4])
			if existing, ok := functions[selector]; ok && existing.Signature != record.Signature {
				return nil, fmt.Errorf("selector %s conflicts between %s and %s", selector, existing.Signature, record.Signature)
			}
			functions[selector] = FunctionInfo{Name: method.Name, Signature: record.Signature, ABI: method}
		}
	}
	return functions, nil
}

func GetKnownContract(address string, chainID uint64) (ContractInfo, bool) {
	normalizedAddr := strings.ToLower(address)
	if chainContracts, exists := KnownContracts[chainID]; exists {
		contractInfo, isKnown := chainContracts[normalizedAddr]
		return contractInfo, isKnown
	}
	return ContractInfo{}, false
}
