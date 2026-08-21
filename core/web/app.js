// The writer page: fetches a Safe transaction, compresses it, erasure-codes it and displays the
// shares as a QR sequence for an air-gapped device to scan.

const CONFIG = {
    CHUNK_SIZE: 500,       // Size of each data chunk in characters
    DISPLAY_TIME: 250,     // Time to display each QR code in ms
    ERROR_CORRECTION: 'L', // QR code error correction level
    REDUNDANCY: 0.6,       // Add 30% more shares for redundancy
    MIN_SHARES: 13         // Minimum number of shares to generate
};

const DOM = {
    txInput: document.getElementById('txInput'),
    startBtn: document.getElementById('startBtn'),
    pasteBtn: document.getElementById('pasteBtn'),
    status: document.getElementById('status'),
    qrcodeDiv: document.getElementById('qrcode'),
    qrOverlay: document.getElementById('qrOverlay'),
    generateProgress: document.getElementById('generateProgress'),
    chunksContainer: document.getElementById('chunksContainer')
};

// Mapping of chain IDs to base URLs. The old safe-transaction-*.safe.global hosts 308-redirect to
// this origin, and CSP re-checks the redirect target against connect-src, so we point here directly.
const CHAIN_ID_TO_BASE_URL = {
    1: 'https://api.safe.global/tx-service/eth',
    10: 'https://api.safe.global/tx-service/oeth',
    8453: 'https://api.safe.global/tx-service/base',
    11155111: 'https://api.safe.global/tx-service/sep'
};

const SAFE_FETCH_MAX_ATTEMPTS = 5;
const SAFE_FETCH_BASE_DELAY_MS = 1000;

const state = {
    chunks: [],
    currentChunkIndex: 0,
    displayInterval: null,
    transferId: null,
    qrCodeImages: [],
    displayedChunks: new Set(),
    directPayloadText: null
};

// A non-fatal decoder turns invalid UTF-8 into U+FFFD, and the replacements would be re-encoded into
// the QR codes as if the link had carried them.
const PAYLOAD_DECODER = new TextDecoder('utf-8', { fatal: true });

// The payload is only ever re-serialized into the QR codes, so re-encoding it can only lose
// information: JSON.parse cannot hold a uint256.
function validateDirectPayload(json) {
    const parsed = JSON.parse(json);
    if (parsed === null || typeof parsed !== 'object' || Array.isArray(parsed)) {
        throw new Error('transaction payload is not a JSON object');
    }
    return json;
}

function checkUrlForTransactionData() {
    const urlParams = new URLSearchParams(window.location.search);
    const txData = urlParams.get('tx');
    const compressedTxData = urlParams.get('txz');
    
    if (txData || compressedTxData) {
        try {
            const decodedData = compressedTxData
                ? decodeCompressedTransactionData(compressedTxData)
                : PAYLOAD_DECODER.decode(base64UrlToUint8Array(txData));
            state.directPayloadText = validateDirectPayload(decodedData);
            
            DOM.status.textContent = "Transaction data found in URL";
            
            DOM.startBtn.click();
        } catch (error) {
            console.error("Error parsing transaction data from URL:", error);
            DOM.status.textContent = "Invalid transaction data in URL";
        }
    }
}

function base64ToUint8Array(base64) {
    const binary = atob(base64);
    const bytes = new Uint8Array(binary.length);
    for (let i = 0; i < binary.length; i++) {
        bytes[i] = binary.charCodeAt(i);
    }
    return bytes;
}

// Accepts either alphabet, as decodeBase64Payload in core/url_payload.go does. The space is a + that
// URLSearchParams decoded, and atob strips whitespace instead of failing, so leaving it would decode
// the payload to different bytes without an error.
function base64UrlToUint8Array(base64Url) {
    let base64 = base64Url.replace(/-/g, '+').replace(/_/g, '/').replace(/ /g, '+');
    const padding = base64.length % 4;
    if (padding) {
        base64 += '='.repeat(4 - padding);
    }
    return base64ToUint8Array(base64);
}

function fastLZDecompress(data) {
    const out = [];
    for (let i = 0; i < data.length;) {
        const control = data[i];
        const tag = control >> 5;
        if (tag === 0) {
            const length = control + 1;
            i += 1;
            if (i + length > data.length) {
                throw new Error("Invalid compressed transaction data");
            }
            for (let j = 0; j < length; j++) {
                out.push(data[i + j]);
            }
            i += length;
            continue;
        }

        let length;
        let distance;
        if (tag < 7) {
            if (i + 1 >= data.length) {
                throw new Error("Invalid compressed transaction data");
            }
            length = tag + 2;
            distance = ((control & 0x1f) << 8) | data[i + 1];
            i += 2;
        } else {
            if (i + 2 >= data.length) {
                throw new Error("Invalid compressed transaction data");
            }
            length = data[i + 1] + 9;
            distance = ((control & 0x1f) << 8) | data[i + 2];
            i += 3;
        }

        const ref = out.length - distance - 1;
        if (ref < 0) {
            throw new Error("Invalid compressed transaction data");
        }
        for (let j = 0; j < length; j++) {
            if (ref + j >= out.length) {
                throw new Error("Invalid compressed transaction data");
            }
            out.push(out[ref + j]);
        }
    }
    return new Uint8Array(out);
}

function decodeCompressedTransactionData(compressedTxData) {
    const compressedBytes = base64UrlToUint8Array(compressedTxData);
    const decompressedBytes = fastLZDecompress(compressedBytes);
    return PAYLOAD_DECODER.decode(decompressedBytes);
}

// Safe UI links carry the chain as a short-name prefix on the safe address, e.g. `safe=oeth:0x...`.
// Anything verifying hashes must take the chain from here rather than letting it be inferred from
// whichever tx service answers first.
const SAFE_SHORT_NAME_TO_CHAIN_ID = {
    eth: 1,
    oeth: 10,
    base: 8453,
    sep: 11155111
};

function extractSafeChainId(input) {
    const match = /[?&]safe=([a-z0-9]+):0x[0-9a-fA-F]{40}/.exec(input);
    if (!match) return null;

    const chainId = SAFE_SHORT_NAME_TO_CHAIN_ID[match[1]];
    if (!chainId) {
        throw new Error(`Safe URL names chain "${match[1]}", which this page does not support`);
    }
    return chainId;
}

function extractTransactionHash(input) {
    if (input.includes('app.safe.global') && input.includes('id=')) {
        const idParam = input.split('id=')[1];
        const txHash = idParam.includes('&') ? idParam.split('&')[0] : idParam;
        
        // The hash might be part of a longer string like multisig_0x...address_0x...hash
        if (txHash.includes('multisig_') && txHash.includes('_0x')) {
            return txHash.split('_').pop(); // Return the last part after the last underscore
        }
        
        return txHash;
    }
    
    return input.trim();
}

function delay(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
}

// Generic fetch with exponential backoff retries
async function fetchWithRetries(url, options = {}) {
    const maxAttempts = options.maxAttempts || SAFE_FETCH_MAX_ATTEMPTS;
    const baseDelayMs = options.baseDelayMs || SAFE_FETCH_BASE_DELAY_MS;
    const context = options.context || 'fetch';
    const notFoundStatuses = options.notFoundStatuses || [];
    let lastError = null;

    for (let attempt = 1; attempt <= maxAttempts; attempt++) {
        try {
            const response = await fetch(url);
            if (!response.ok) {
                if (notFoundStatuses.includes(response.status)) {
                    return null;
                }
                throw new Error(`${context}: ${response.status} ${response.statusText}`);
            }
            return await response.json();
        } catch (error) {
            lastError = error;
            console.warn(`${context} attempt ${attempt} failed`, error);

            if (attempt < maxAttempts) {
                const backoff = baseDelayMs * Math.pow(2, attempt - 1);
                const jitter = Math.random() * baseDelayMs;
                await delay(backoff + jitter);
            }
        }
    }

    throw new Error(`${context} failed after ${maxAttempts} attempts: ${lastError?.message || 'Unknown error'}`);
}

async function fetchSafeVersion(chainId, safeAddress) {
    const baseUrl = CHAIN_ID_TO_BASE_URL[chainId];
    if (!baseUrl) {
        throw new Error(`Unsupported chain ${chainId} for Safe ${safeAddress}`);
    }
    const url = `${baseUrl}/api/v1/safes/${safeAddress}/`;
    const data = await fetchWithRetries(url, {
        context: `Safe version fetch for ${safeAddress}`
    });
    if (!data.version) {
        throw new Error(`Safe version missing in response for ${safeAddress}`);
    }
    return data.version;
}

// The Safe API returns these as decimal strings. Number() silently rounds above 2^53-1, and the
// rounded figure would then be hashed by both the wasm recomputation and the Safe contract, so an
// unrepresentable value must fail rather than be quietly altered.
function assertExactInteger(name, raw) {
    const parsed = Number(raw);
    // Negatives are rejected too: these are uint256 fields, and a negative would be ABI-packed as a
    // very large unsigned value, hashing something other than what was displayed.
    if (!Number.isSafeInteger(parsed) || parsed < 0 || String(parsed) !== String(raw).trim()) {
        throw new Error(`${name} is ${raw}, which is not an exactly representable unsigned integer`);
    }
    return parsed;
}

const TX_HASH_PATTERN = /^0x[0-9a-fA-F]{64}$/;

function assertTxHash(name, raw) {
    if (!TX_HASH_PATTERN.test(String(raw || ''))) {
        throw new Error(`${name} is ${raw || '(missing)'}, which is not a 32-byte hash`);
    }
    return String(raw).toLowerCase();
}

// A response that answers a request for one hash with another transaction's fields must not pass.
// Why that is not paranoia: core.SafeTransaction.SafeTxHash.
function assertAnsweredHash(what, requested, returned) {
    if (assertTxHash(`${what} requested hash`, requested) !== assertTxHash(`${what} safeTxHash`, returned)) {
        throw new Error(
            `Asked the Safe transaction service for ${what} ${requested} and it returned ${returned}. ` +
            `The fields it sent belong to a different transaction, so nothing here is what was asked for.`
        );
    }
}

// Safe's Operation enum is CALL (0) or DELEGATECALL (1). Anything else is not a Safe operation.
function assertOperation(raw) {
    const parsed = assertExactInteger('operation', raw);
    if (parsed !== 0 && parsed !== 1) {
        throw new Error(`operation is ${raw}; Safe allows only 0 (CALL) or 1 (DELEGATECALL)`);
    }
    return parsed;
}

// chainId pins the lookup to one network. Without it every chain is probed and whichever service
// answers first decides what is verified: Safe <= 1.2.0 omits chainId from its domain separator, so
// identical fields on 2 chains give identical Ledger hashes and a wrong-chain lookup would not show
// up in the hash comparison.
async function fetchTransactionData(txHash, chainId = null) {
    if (chainId !== null && !CHAIN_ID_TO_BASE_URL[chainId]) {
        throw new Error(`Unsupported chain ${chainId}`);
    }
    const requestedHash = assertTxHash('transaction hash', txHash);
    const chainIds = chainId !== null ? [String(chainId)] : Object.keys(CHAIN_ID_TO_BASE_URL);
    // A chain that answered "no such transaction" and a chain whose service was unreachable are
    // different facts, and reporting the second as the first tells a signer their transaction does
    // not exist when nobody knows whether it does.
    const absent = [];
    const unreachable = [];
    
    let foundContent = null;
    let foundChainId = null;
    let foundBaseUrl = null;
    
    for (const chainId of chainIds) {
        const baseUrl = CHAIN_ID_TO_BASE_URL[chainId];
        const url = `${baseUrl}/api/v2/multisig-transactions/${txHash}/`;

        try {
            const result = await fetchWithRetries(url, {
                context: `Transaction fetch on chain ${chainId}`,
                notFoundStatuses: [404, 422]  // Transaction not found on this chain
            });

            if (result === null) {
                absent.push(`chain ${chainId}`);
                continue;
            }

            foundContent = result;
            foundChainId = chainId;
            foundBaseUrl = baseUrl;
            break;
        } catch (error) {
            unreachable.push(`chain ${chainId}: ${error.message}`);
        }
    }

    if (!foundContent) {
        if (unreachable.length) {
            throw new Error(
                `The Safe transaction service did not answer, so whether this transaction exists is ` +
                `unknown: ${unreachable.join('; ')}` +
                (absent.length ? ` (not found on ${absent.join(', ')})` : '')
            );
        }
        throw new Error(`Transaction not found on ${absent.join(', ')}`);
    }

    assertAnsweredHash('the transaction', requestedHash, foundContent.safeTxHash);
    
    let content = foundContent;
    let nested = null;
    
    let boundHash = requestedHash;
    if (content.data && content.data.startsWith('0xd4d9bdcd')) {
        const innerHash = assertTxHash('approved hash', '0x' + content.data.slice(10, 10 + 64));
        console.log("Detected approveHash transaction, inner hash:", innerHash);

        const innerUrl = `${foundBaseUrl}/api/v2/multisig-transactions/${innerHash}/`;
        const innerContent = await fetchWithRetries(innerUrl, {
            context: `Inner transaction fetch for ${innerHash}`
        });
        // The parent's calldata is the only statement of which transaction the child is meant to
        // be, so the response has to answer to it.
        assertAnsweredHash('the approved transaction', innerHash, innerContent.safeTxHash);

        nested = {
            safe: content.safe,
            safe_version: await fetchSafeVersion(foundChainId, content.safe),
            safe_tx_hash: requestedHash,
            nonce: assertExactInteger('nonce', content.nonce),
            data: content.data,
            operation: assertOperation(content.operation),
            to: content.to,
        };

        content = innerContent;
        boundHash = innerHash;
    }

    return {
        safe: content.safe,
        safe_version: await fetchSafeVersion(foundChainId, content.safe),
        // Binds these fields to the hash they were fetched under, so verification refuses to hash
        // anything other than the transaction that was asked for.
        safe_tx_hash: boundHash,
        chain: parseInt(foundChainId),
        to: content.to,
        // Passed through as a string: parseInt loses precision above 2^53-1 wei, which
        // silently corrupts the message hash. op-txverify accepts a string or a number.
        value: content.value,
        data: content.data,
        operation: assertOperation(content.operation),
        safe_tx_gas: assertExactInteger('safeTxGas', content.safeTxGas),
        base_gas: assertExactInteger('baseGas', content.baseGas),
        gas_price: assertExactInteger('gasPrice', content.gasPrice),
        gas_token: content.gasToken,
        refund_receiver: content.refundReceiver,
        nonce: assertExactInteger('nonce', content.nonce),
        nested,
    };
}

function generateTransferId() {
    return 'transfer-' + Date.now() + '-' + Math.random().toString(36).substring(2, 10);
}

async function calculateSHA256(str) {
    const encoder = new TextEncoder();
    const data = encoder.encode(str);
    
    const hashBuffer = await crypto.subtle.digest('SHA-256', data);
    
    const hashArray = Array.from(new Uint8Array(hashBuffer));
    const hashHex = hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
    
    return hashHex;
}

function stopDisplay() {
    if (state.displayInterval) {
        clearInterval(state.displayInterval);
        state.displayInterval = null;
    }
}

function uint8ArrayToBase64(uint8Array) {
    let binary = '';
    const bytes = new Uint8Array(uint8Array);
    const len = bytes.byteLength;
    for (let i = 0; i < len; i++) {
        binary += String.fromCharCode(bytes[i]);
    }
    return btoa(binary);
}

function createChunkIndicators(count) {
    DOM.chunksContainer.innerHTML = '';
    for (let i = 0; i < count; i++) {
        const indicator = document.createElement('div');
        indicator.className = 'chunk-indicator';
        indicator.dataset.index = i;
        indicator.textContent = i + 1;
        DOM.chunksContainer.appendChild(indicator);
    }
}

function updateChunkIndicators(currentIndex) {
    state.displayedChunks.add(currentIndex);
    
    const indicators = DOM.chunksContainer.querySelectorAll('.chunk-indicator');
    indicators.forEach((indicator, index) => {
        indicator.classList.remove('current');
        
        if (index === currentIndex) {
            indicator.classList.add('current');
        }
    });
}

function displayQRCode(index) {
    if (!state.chunks || index >= state.chunks.length) return;
    
    DOM.qrcodeDiv.innerHTML = '';
    
    const img = document.createElement('img');
    img.src = state.qrCodeImages[index];
    img.style.maxWidth = '100%';
    img.style.maxHeight = '100%';
    DOM.qrcodeDiv.appendChild(img);
    
    updateChunkIndicators(index);
}

// transactionData is either the verbatim JSON text of a supplied payload or an object built from the
// Safe API. Forwarding the text unchanged keeps a supplied payload byte-identical through the QR
// codes, so the air-gapped device verifies what the link carried rather than a re-serialization.
async function generateQRCodes(transactionData) {
    const jsonData = typeof transactionData === 'string' ? transactionData : JSON.stringify(transactionData);
    
    const jsonBytes = fflate.strToU8(jsonData);
    const compressedData = fflate.zlibSync(jsonBytes);
    const compressedSize = compressedData.length;

    const originalBlobs = Math.ceil(compressedSize / CONFIG.CHUNK_SIZE);
    
    // allowedFailures determines how many extra shares to create for redundancy
    // note that erasure.js creates 2*allowedFailures additional shares
    const allowedFailures = Math.max(
        Math.ceil(originalBlobs * CONFIG.REDUNDANCY / 2),
        Math.ceil((CONFIG.MIN_SHARES - originalBlobs) / 2)
    );

    console.log(`Data size: ${compressedSize} bytes, Required Shards: ${originalBlobs}, Allowed failures: ${allowedFailures}`);

    const shares = erasure.split(compressedData, originalBlobs, allowedFailures);

    state.transferId = generateTransferId();
    
    const base64CompressedData = uint8ArrayToBase64(compressedData);
    const fullChecksum = await calculateSHA256(base64CompressedData);
    
    state.chunks = [];
    for (let i = 0; i < shares.length; i++) {
        const shardData = uint8ArrayToBase64(shares[i]);
        const shardChecksum = await calculateSHA256(shardData);
        
        state.chunks.push({
            id: state.transferId,
            part: i + 1,
            total: shares.length,
            originalBlobs: originalBlobs,  // Store how many shards needed to reconstruct
            allowedFailures: allowedFailures, // Store allowed failures
            data: shardData,
            checksum: shardChecksum,
            fullChecksum: fullChecksum,
            originalSize: compressedSize // Store original size for reconstruction
        });
    }
    
    state.displayedChunks = new Set();
    
    createChunkIndicators(state.chunks.length);
    
    state.qrCodeImages = [];
    
    for (let i = 0; i < state.chunks.length; i++) {
        const chunk = state.chunks[i];
        const qrData = JSON.stringify(chunk);
        
        DOM.generateProgress.value = (i / state.chunks.length) * 100;
        
        const dataUrl = await new Promise((resolve, reject) => {
            QRCode.toDataURL(qrData, {
                errorCorrectionLevel: CONFIG.ERROR_CORRECTION,
                margin: 1,
                width: 300
            }, (err, url) => {
                if (err) reject(err);
                else resolve(url);
            });
        });
        
        state.qrCodeImages.push(dataUrl);
        
        // Small delay to allow UI to update
        await new Promise(resolve => setTimeout(resolve, 10));
    }
}

function startQRCodeDisplay() {
    DOM.status.textContent = `Displaying ${state.chunks.length} QR codes`;
    
    state.currentChunkIndex = 0;
    displayQRCode(state.currentChunkIndex);
    
    state.displayInterval = setInterval(function() {
        state.currentChunkIndex = (state.currentChunkIndex + 1) % state.chunks.length;
        displayQRCode(state.currentChunkIndex);
    }, CONFIG.DISPLAY_TIME);
}

DOM.pasteBtn.addEventListener('click', async function() {
    try {
        const text = await navigator.clipboard.readText();
        DOM.txInput.value = text;
        // Assigning to value fires nothing, and verify.js listens for input to drop a result that
        // no longer describes the box.
        DOM.txInput.dispatchEvent(new Event('input', { bubbles: true }));
        DOM.startBtn.click(); // Programmatically click the Start Display button
    } catch (err) {
        console.error('Failed to read clipboard contents: ', err);
        DOM.status.textContent = "Failed to paste from clipboard. Make sure you've granted permission.";
    }
});

DOM.startBtn.addEventListener('click', async function() {
    stopDisplay();

    try {
        let transactionData;
        
        // The box is the only source of a hash and a supplied payload is used only while it is empty,
        // so no payload can stand in for a hash the signer can read. Typing over it disarms the link.
        const txInput = DOM.txInput.value.trim();
        
        if (txInput) {
            const txHash = extractTransactionHash(txInput);
            
            if (!txHash.startsWith('0x')) {
                DOM.status.textContent = "Invalid transaction hash format";
                return;
            }
            
            if (document.getElementById('defaultSunny')) {
                document.getElementById('defaultSunny').style.display = 'none';
            }
            DOM.qrOverlay.style.display = 'flex';
            DOM.generateProgress.value = 0;
            DOM.status.textContent = "Fetching transaction data...";
            
            // Fetch transaction data
            transactionData = await fetchTransactionData(txHash);
        } else if (state.directPayloadText) {
            transactionData = state.directPayloadText;
            DOM.status.textContent = "Using transaction data from URL...";
        } else {
            DOM.status.textContent = "Please enter a transaction hash or link";
            return;
        }

        if (document.getElementById('defaultSunny')) {
            document.getElementById('defaultSunny').style.display = 'none';
        }
        DOM.qrOverlay.style.display = 'flex';
        DOM.generateProgress.value = 0;
        
        DOM.startBtn.disabled = true;
        
        DOM.status.textContent = "Generating QR codes...";
        
        await generateQRCodes(transactionData);
        
        DOM.qrOverlay.style.display = 'none';
        
        DOM.startBtn.disabled = false;
        
        startQRCodeDisplay();
        
    } catch (error) {
        console.error("Error:", error);
        DOM.status.textContent = "Error: " + error.message;
        DOM.qrOverlay.style.display = 'none';
        DOM.startBtn.disabled = false;
        document.getElementById('defaultSunny').style.display = 'block';
    }
});

window.addEventListener('beforeunload', stopDisplay);

window.addEventListener('DOMContentLoaded', checkUrlForTransactionData);
