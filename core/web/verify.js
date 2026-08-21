/**
 * Verify mode. Shows a signer the two EIP-712 hashes their hardware wallet will display, plus the
 * decoded calldata, without involving the computer running the Safe UI.
 *
 * Two independent sources for every hash: the Safe contract's own encodeTransactionData over
 * eth_call, and the wasm module recomputing it from the fetched parameters. A wrong preimage
 * disagrees visibly instead of substituting a hash.
 *
 * A classic script; takes fetchWithRetries, fetchTransactionData, extractTransactionHash,
 * CHAIN_ID_TO_BASE_URL, DOM and state from app.js.
 */

// PublicNode remains an availability fallback. The first candidate is a same-origin Netlify
// function, which chooses an operator-configured upstream without exposing its URL or widening
// CSP beyond `connect-src 'self'`.
const CHAIN_ID_TO_PUBLIC_RPC = {
    1: 'https://ethereum-rpc.publicnode.com',
    10: 'https://optimism-rpc.publicnode.com',
    8453: 'https://base-rpc.publicnode.com',
    11155111: 'https://ethereum-sepolia-rpc.publicnode.com'
};

const RPC_ENDPOINT_CHAIN_CACHE = new Map();
const RPC_TIMEOUT_MS = 5_000;

class RPCAvailabilityError extends Error {}

function rpcCandidates(chainId) {
    const fallback = CHAIN_ID_TO_PUBLIC_RPC[chainId];
    if (!fallback) throw new Error(`No RPC configured for chain ${chainId}`);
    return [
        { url: `/rpc/${chainId}`, name: 'same-origin operator RPC', sameOrigin: true },
        { url: fallback, name: 'PublicNode RPC' },
    ];
}

async function rpcRequest(endpoint, request) {
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), RPC_TIMEOUT_MS);
    let response;
    try {
        response = await fetch(endpoint.url, {
            method: 'POST',
            headers: { 'content-type': 'application/json' },
            body: JSON.stringify(request),
            signal: controller.signal,
        });
    } catch {
        clearTimeout(timer);
        throw new RPCAvailabilityError(`${endpoint.name} is unavailable`);
    }
    try {
        if (!response.ok) {
            if ((endpoint.sameOrigin && response.status === 404) || response.status === 429 || response.status >= 500) {
                throw new RPCAvailabilityError(`${endpoint.name} is unavailable`);
            }
            throw new Error(`${endpoint.name} rejected the request (${response.status})`);
        }

        let json;
        try {
            json = await response.json();
        } catch {
            throw new RPCAvailabilityError(`${endpoint.name} returned invalid JSON`);
        }
        if (!json || json.jsonrpc !== '2.0' || json.id !== request.id ||
            (!Object.prototype.hasOwnProperty.call(json, 'result') && !json.error)) {
            throw new RPCAvailabilityError(`${endpoint.name} returned an invalid RPC response`);
        }
        if (json.error) {
            if (request.method === 'eth_chainId') {
                throw new RPCAvailabilityError(`${endpoint.name} cannot verify its chain`);
            }
            throw new Error(`${endpoint.name} RPC error: ${json.error.message || 'unknown error'}`);
        }
        return json.result;
    } finally {
        clearTimeout(timer);
    }
}

async function verifyRPCChain(endpoint, chainId) {
    if (RPC_ENDPOINT_CHAIN_CACHE.has(endpoint.url)) return;
    const result = await rpcRequest(endpoint, {
        jsonrpc: '2.0', id: 1, method: 'eth_chainId', params: [],
    });
    const returnedChainId = typeof result === 'string' && /^0x[0-9a-f]+$/i.test(result)
        ? parseInt(result, 16)
        : NaN;
    if (returnedChainId !== chainId) {
        throw new RPCAvailabilityError(`${endpoint.name} is configured for the wrong chain`);
    }
    RPC_ENDPOINT_CHAIN_CACHE.set(endpoint.url, true);
}

// Shown beside the hashes: a signer must be able to see which chain was verified, since the hash
// comparison alone cannot reveal a wrong-chain lookup on Safe <= 1.2.0.
const NETWORK_NAMES = {
    1: 'Ethereum',
    10: 'OP Mainnet',
    8453: 'Base',
    11155111: 'Sepolia'
};

const EXPLORERS = {
    1: 'https://etherscan.io/address/',
    10: 'https://optimistic.etherscan.io/address/',
    8453: 'https://basescan.org/address/',
    11155111: 'https://sepolia.etherscan.io/address/'
};

// Only the lookup inputs are remembered. Persisting transaction parameters and pre-filling from them
// would mean hashing fields this page supplied rather than fields it fetched, which is the same hole
// as accepting them from a ?tx= link.
const LOOKUP_KEY = 'op-txverify:lookup';

const VERIFY_DOM = {
    network: document.getElementById('verifyNetwork'),
    safe: document.getElementById('verifySafe'),
    safePreview: document.getElementById('verifySafePreview'),
    nonce: document.getElementById('verifyNonce'),
    btn: document.getElementById('verifyBtn'),
    panel: document.getElementById('verifyPanel'),
    build: document.getElementById('buildInfo')
};

const IDLE_STATUS = DOM.status.textContent;

// A displayed result is a statement about the inputs it was produced from. Editing any of them
// makes it a statement about a transaction nobody asked about, and a run still in flight would
// write its result under inputs it never read. One counter covers both: a run owns the display
// only while it is still the current generation.
let generation = 0;

const VERIFY_INPUTS = [DOM.txInput, VERIFY_DOM.network, VERIFY_DOM.safe, VERIFY_DOM.nonce];

function updateSafePreview() {
    VERIFY_DOM.safePreview.textContent = (VERIFY_DOM.safe.value || '').trim();
}

function invalidateResults() {
    generation++;
    VERIFY_DOM.panel.innerHTML = '';
    DOM.status.textContent = IDLE_STATUS;
}

// The paste button writes to the input, so it is disabled along with it: a disabled input still
// accepts a programmatic write.
function setInputsDisabled(disabled) {
    for (const element of [...VERIFY_INPUTS, VERIFY_DOM.btn, DOM.startBtn, DOM.pasteBtn]) {
        element.disabled = disabled;
    }
}

let wasmReady = null;

function loadWasm() {
    if (!wasmReady) {
        wasmReady = (async () => {
            const go = new window.Go();
            const wasm = await WebAssembly.instantiate(
                await (await fetch('/wasm/main.wasm')).arrayBuffer(),
                go.importObject
            );
            // Not awaited: the module's main blocks forever so the export stays callable.
            go.run(wasm.instance);
        })();
    }
    return wasmReady;
}

// Everything rendered here originates in an API response or an ABI decode, and ABI string arguments
// can carry arbitrary text, so nothing reaches innerHTML unescaped.
function esc(text) {
    return String(text).replace(/[&<>"']/g, c =>
        ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
}

function loadLookup() {
    try {
        return JSON.parse(localStorage.getItem(LOOKUP_KEY)) || {};
    } catch {
        return {};
    }
}

function saveLookup(network, safe, nonce) {
    try {
        localStorage.setItem(LOOKUP_KEY, JSON.stringify({ network, safe, nonce }));
    } catch {
        // A browser refusing storage must not block verification.
    }
}

// Resolves a Safe address and nonce to one safeTxHash, refusing to guess when several transactions
// sit at that nonce. The hash it returns is the service's answer rather than the signer's own
// knowledge, so binding the fetched fields to it proves the two endpoints agree, not that the
// transaction is the intended one - that is what reading the decode is for.
async function resolveNonce(chainId, safeAddress, nonce) {
    const baseUrl = CHAIN_ID_TO_BASE_URL[chainId];
    if (!baseUrl) throw new Error(`Unsupported chain ${chainId}`);

    const url = `${baseUrl}/api/v1/safes/${safeAddress}/multisig-transactions/?nonce=${nonce}`;
    const data = await fetchWithRetries(url, { context: `Nonce lookup for ${safeAddress}` });

    const results = data.results || [];
    if (!data.count || results.length === 0) {
        throw new Error(`No transaction at nonce ${nonce} for ${safeAddress}`);
    }
    if (data.count > 1 || results.length > 1) {
        const pending = results.filter(r => !r.isExecuted).map(r => r.safeTxHash);
        const all = results
            .map(r => `${r.safeTxHash} (${r.isExecuted ? 'executed' : 'pending'})`)
            .join('\n  ');
        throw new Error(
            `${data.count} transactions share nonce ${nonce}, so the nonce does not identify one:\n  ` +
            `${all}\nConfirm which is intended against the channel that announced it, then enter that ` +
            `hash above. ${pending.length === 1 ? 'Exactly one is still pending.' : ''}`
        );
    }
    return results[0].safeTxHash;
}

// The raw eth_call result is ABI-encoded dynamic bytes: a 32-byte offset, a 32-byte length, the
// 66-byte preimage, then padding to a 32-byte boundary. The trailing 32 bytes are padding, not the
// message hash, so this reads the length rather than slicing from the end.
function parsePreimage(raw) {
    const body = (raw || '').replace(/^0x/, '');
    // 64 offset + 64 length + 132 payload, padded to 320.
    if (body.length !== 320) {
        throw new Error(`expected a 320-character eth_call result, got ${body.length}`);
    }
    if (parseInt(body.slice(0, 64), 16) !== 0x20) {
        throw new Error('eth_call result has an unexpected ABI offset');
    }
    if (parseInt(body.slice(64, 128), 16) !== 66) {
        throw new Error('eth_call result is not a 66-byte preimage');
    }

    const preimage = body.slice(128, 260);
    if (!preimage.startsWith('1901')) {
        throw new Error('preimage is not EIP-712 (missing the 0x1901 prefix)');
    }
    return {
        domainHash: `0x${preimage.slice(4, 68)}`,
        messageHash: `0x${preimage.slice(68, 132)}`
    };
}

async function ethCall(chainId, target, calldata) {
    const unavailable = [];
    for (const endpoint of rpcCandidates(chainId)) {
        try {
            await verifyRPCChain(endpoint, chainId);
            // parsePreimage is deliberately inside the non-availability path: a successful
            // eth_call is a signer-relevant on-chain result, never a reason to try another RPC.
            return parsePreimage(await rpcRequest(endpoint, {
                jsonrpc: '2.0',
                id: 2,
                method: 'eth_call',
                params: [{ to: target, data: calldata }, 'latest'],
            }));
        } catch (error) {
            if (!(error instanceof RPCAvailabilityError)) throw error;
            unavailable.push(error.message);
        }
    }
    throw new Error(`RPC unavailable: ${unavailable.join('; ')}`);
}

function definitionRow(label, valueHTML, warn = false, labelIsHTML = false) {
    return `<div class="verify-field${warn ? ' verify-warn' : ''}">` +
        `<dt>${labelIsHTML ? label : esc(label)}</dt><dd>${valueHTML}</dd></div>`;
}

function isArgument(value) {
    return value !== null && typeof value === 'object' &&
        Object.prototype.hasOwnProperty.call(value, 'name') &&
        Object.prototype.hasOwnProperty.call(value, 'type') &&
        Object.prototype.hasOwnProperty.call(value, 'value') &&
        nonemptyString(value.name) && nonemptyString(value.type);
}

function nonemptyString(value) {
    return typeof value === 'string' && value.trim() !== '';
}

// The WASM boundary is local, but rendering a malformed object as a known action would still turn
// missing presentation data into a false statement. Normalize every call once and fail closed to an
// explicit unknown action before either the renderer or final status inspects it.
function normalizeCall(call) {
    const validObject = call !== null && typeof call === 'object' && !Array.isArray(call);
    const source = validObject ? call : {};
    const functionName = nonemptyString(source.functionName)
        ? source.functionName
        : 'Unknown function';
    const operationValid = source.operation === 'CALL' || source.operation === 'DELEGATECALL';
    const targetValid = nonemptyString(source.target);
    const argumentsValid = Array.isArray(source.arguments) && source.arguments.every(isArgument);
    const callsValid = source.calls === undefined || Array.isArray(source.calls);
    const signatureRequired = !['Unknown function', 'Send native ETH', 'No calldata']
        .includes(functionName);
    const signatureValid = !signatureRequired || nonemptyString(source.signature);
    const malformed = !validObject || functionName === 'Unknown function' || !operationValid ||
        !targetValid || !argumentsValid || !callsValid || !signatureValid;
    const rawCalldata = nonemptyString(source.rawCalldata)
        ? source.rawCalldata
        : '(missing calldata)';
    const selector = nonemptyString(source.selector)
        ? source.selector
        : (rawCalldata.startsWith('0x') ? rawCalldata.slice(0, 10) : '(missing selector)');

    const normalized = {
        target: targetValid ? source.target : '(missing target)',
        operation: operationValid ? source.operation : '(unknown)',
        functionName: malformed ? 'Unknown function' : functionName,
        arguments: argumentsValid ? source.arguments : [],
        calls: callsValid && Array.isArray(source.calls) ? source.calls.map(normalizeCall) : [],
    };
    if (nonemptyString(source.targetLabel)) normalized.targetLabel = source.targetLabel;
    if (!malformed && nonemptyString(source.signature)) normalized.signature = source.signature;
    if (normalized.functionName === 'Unknown function') {
        normalized.selector = selector;
        normalized.rawCalldata = rawCalldata;
    }
    return normalized;
}

function renderArgument(argument) {
    return definitionRow(
        argument.name,
        `<span class="verify-type">${esc(argument.type)}</span>${renderValue(argument.value)}`
    );
}

function renderValue(value) {
    if (Array.isArray(value)) {
        return `<ol class="verify-values">${value.map(item =>
            `<li>${isArgument(item)
                ? `<dl class="verify-fields verify-fields-nested">${renderArgument(item)}</dl>`
                : renderValue(item)}</li>`).join('')}</ol>`;
    }
    if (value !== null && typeof value === 'object') {
        return `<dl class="verify-fields verify-fields-nested">${Object.entries(value)
            .map(([name, item]) => definitionRow(name, renderValue(item))).join('')}</dl>`;
    }
    return `<code>${esc(value === null ? 'null' : value)}</code>`;
}

function renderArguments(args) {
    if (!Array.isArray(args) || args.length === 0) return '';
    return `<section class="verify-arguments"><h5>Arguments</h5><dl class="verify-fields">` +
        args.map(argument => isArgument(argument)
            ? renderArgument(argument)
            : definitionRow('(unnamed)', renderValue(argument))).join('') + '</dl></section>';
}

function renderTarget(call) {
    const label = call.targetLabel
        ? `<strong class="verify-target-label">${esc(call.targetLabel)}</strong>`
        : '';
    return `${label}<code>${esc(call.target)}</code>`;
}

function renderCall(call, path = [], normalized = false) {
    if (!normalized) call = normalizeCall(call);
    const functionName = call.functionName || 'Unknown function';
    const operation = call.operation || '(unknown)';
    const delegatecall = operation === 'DELEGATECALL';
    const subject = call.targetLabel || call.target;
    let summary;
    if (functionName === 'Unknown function') {
        summary = 'Unknown function';
    } else if (functionName === 'Send native ETH') {
        summary = `Send native ETH to ${subject}`;
    } else if (functionName === 'No calldata') {
        summary = `No calldata for ${subject}`;
    } else {
        summary = `${delegatecall ? 'DELEGATECALL' : 'Call'} ${subject}: ${functionName}`;
    }

    let html = `<article class="verify-action${delegatecall ? ' verify-action-delegate' : ''}">`;
    if (path.length > 0) {
        html += `<p class="verify-action-number">${esc(`Action ${path.join('.')}`)}</p>`;
    }
    html += `<h4>${esc(summary)}</h4><dl class="verify-fields">`;
    html += definitionRow('Target', renderTarget(call));
    if (call.signature) {
        html += definitionRow('Signature', `<code>${esc(call.signature)}</code>`);
    }
    html += definitionRow(
        'Operation',
        delegatecall
            ? `<strong class="verify-operation-warn">${esc(operation)}</strong>`
            : `<code>${esc(operation)}</code>`
    );
    if (functionName === 'Unknown function') {
        html += definitionRow('Selector', `<code>${esc(call.selector)}</code>`);
        html += definitionRow('Raw calldata', `<code>${esc(call.rawCalldata)}</code>`);
    }
    html += '</dl>';
    html += renderArguments(call.arguments);

    if (functionName === 'Unknown function') {
        html += '<p class="verify-intent-warning"><strong>DO NOT SIGN.</strong> The hashes may ' +
            'agree, but this page has not established this transaction\'s intent. Do not sign until ' +
            'the action is independently decoded and confirmed.</p>';
    }

    if (Array.isArray(call.calls) && call.calls.length > 0) {
        html += '<section class="verify-subactions"><h5>Nested actions</h5>';
        html += call.calls.map((subcall, index) =>
            renderCall(subcall, [...path, index + 1], true)).join('');
        html += '</section>';
    }
    return html + '</article>';
}

const SAFE_FIELD_ORDER = [
    'to', 'value', 'data', 'operation', 'safeTxGas', 'baseGas', 'gasPrice', 'gasToken',
    'refundReceiver', 'nonce'
];

function renderSafeFields(fields) {
    const present = new Set(SAFE_FIELD_ORDER.filter(name =>
        Object.prototype.hasOwnProperty.call(fields, name)));
    const names = [...present, ...Object.keys(fields).filter(name => !present.has(name))];
    return `<dl class="verify-fields verify-raw-fields">${names.map(name =>
        definitionRow(`<code>${esc(name)}</code>`, renderValue(fields[name]), false, true)).join('')}</dl>`;
}

function renderRawDetails(fields) {
    return '<details class="verify-raw" open><summary>Raw transaction fields</summary>' +
        renderSafeFields(fields) + '</details>';
}

function hashRow(label, contractValue, localValue) {
    if (localValue === undefined) {
        return definitionRow(label, `<code>${esc(contractValue)}</code>`);
    }
    return definitionRow(label,
        `<span class="verify-hash-source">Safe contract: <code>${esc(contractValue)}</code></span>` +
        `<span class="verify-hash-source">Local recomputation: <code>${esc(localValue)}</code></span>`,
        true
    );
}

function renderCheck(check, onchain, chainId, agree, normalized = false) {
    const call = normalized ? check.call : normalizeCall(check.call);
    let html = `<section class="verify-check"><h3>${check.label === 'ledger'
        ? 'Compare these to your device'
        : 'The nested transaction being approved'}</h3>`;
    html += '<dl class="verify-fields verify-context">';
    html += definitionRow('Network', `<span>${esc(`${NETWORK_NAMES[chainId]} (chain ${chainId})`)}</span>`);
    html += definitionRow('Safe', `<code>${esc(check.target)}</code>`);
    html += '</dl>';

    if (!agree) {
        html += '<p class="verify-intent-warning"><strong>DO NOT SIGN.</strong> The Safe contract ' +
            'and the local recomputation disagree.</p>';
    }
    html += '<dl class="verify-fields verify-hashes">';
    html += hashRow('Domain hash', onchain.domainHash, agree ? undefined : check.domainHash);
    html += hashRow('Message hash', onchain.messageHash, agree ? undefined : check.messageHash);
    html += hashRow('safeTxHash', check.approveHash);
    html += definitionRow(
        'Source',
        `<span>${agree
            ? 'Safe contract and local recomputation agree'
            : 'Safe contract and local recomputation disagree'}</span>`,
        !agree
    );
    html += '</dl>';

    html += `<h3>${check.label === 'ledger' ? 'What you are signing' : 'What is being approved'}</h3>`;
    html += renderCall(call, [], true);
    html += renderRawDetails(check.safeFields);

    const explorer = EXPLORERS[chainId];
    if (check.label === 'ledger' && explorer) {
        html += '<section class="verify-ceremony"><h3>Before you sign</h3><ol>' +
            '<li>Confirm network, Safe, and decoded action.</li>' +
            '<li>Compare domain and message hashes with the hardware wallet.</li>' +
            '<li>Under the compromised-computer threat model, independently run ' +
            `<code>encodeTransactionData</code> on <a href="${esc(explorer + check.target)}` +
            `#readProxyContract">this Safe's contract explorer</a> with the exact raw transaction fields shown.</li>` +
            '<li><strong>DO NOT SIGN</strong> on any mismatch or when the action remains unknown.</li>' +
            '</ol></section>';
    }
    return html + '</section>';
}

function hasUnknownCall(call, normalized = false) {
    if (!normalized) call = normalizeCall(call);
    return call.functionName === 'Unknown function' ||
        call.calls.some(subcall => hasUnknownCall(subcall, true));
}

// Safe 1.5.0 removed encodeTransactionData; getTransactionHash returns only the final hash, which the
// message hash cannot be recovered from. Each check is gated on its own Safe's version, because for a
// nested transaction the parent and child versions differ and the child is the one being eth_called.
function assertVersionSupported(label, version) {
    const [major, minor] = (version || '').split('.').map(Number);
    if (!Number.isInteger(major) || !Number.isInteger(minor) || major > 1 || minor >= 5) {
        throw new Error(
            `The ${label} Safe reports version ${version || '(unknown)'}, which does not expose ` +
            `encodeTransactionData. Verify this transaction with the op-txverify CLI instead.`
        );
    }
}

async function runVerify(run) {
    // Read before every write, never cached: the generation can move while this run awaits.
    const current = () => run === generation;
    const status = (text) => { if (current()) DOM.status.textContent = text; };

    // A ?tx= or ?txz= link carries fields chosen by whoever built the link. encodeTransactionData is a
    // pure function of its arguments and does not read the queued transaction, so hashing supplied
    // fields would let a compromised computer choose the hash the contract confirms.
    if (state.directPayloadText) {
        throw new Error(
            'This link supplied the transaction fields instead of a hash, so the hashes would only ' +
            'restate what the link claims. Reopen the page without ?tx=/?txz= and enter the ' +
            'safeTxHash, or the Safe address, nonce and network.'
        );
    }

    const typed = (DOM.txInput.value || '').trim();
    const safeAddress = (VERIFY_DOM.safe.value || '').trim();
    const nonce = (VERIFY_DOM.nonce.value || '').trim();

    // The network is always explicit. Letting it be inferred from whichever tx service answers
    // first would be unsafe: Safe <= 1.2.0 omits chainId from its domain separator, so the same
    // fields on 2 chains give the same Ledger hashes and a wrong-chain lookup would pass the hash
    // comparison while the addresses mean something else entirely.
    const chainId = parseInt(VERIFY_DOM.network.value, 10);
    if (!CHAIN_ID_TO_PUBLIC_RPC[chainId]) throw new Error('Select a supported network');

    let txHash;
    if (typed) {
        const urlChainId = extractSafeChainId(typed);
        if (urlChainId !== null && urlChainId !== chainId) {
            throw new Error(
                `The Safe URL is for chain ${urlChainId} (${NETWORK_NAMES[urlChainId] || 'unknown'}) ` +
                `but ${NETWORK_NAMES[chainId]} is selected. Fix the selection rather than guessing.`
            );
        }
        txHash = extractTransactionHash(typed);
    } else if (safeAddress && nonce) {
        txHash = await resolveNonce(chainId, safeAddress, nonce);
        saveLookup(chainId, safeAddress, nonce);
    } else {
        throw new Error('Enter a safeTxHash, or a Safe address, nonce and network');
    }

    status(`Fetching transaction on ${NETWORK_NAMES[chainId]}...`);
    const tx = await fetchTransactionData(txHash, chainId);
    if (tx.chain !== chainId) {
        throw new Error(`Fetched a transaction on chain ${tx.chain}, expected ${chainId}`);
    }

    // fetchTransactionData held the response to the hash it asked for; this holds it to the hash
    // this page was given, so the two cannot drift apart.
    const ledgerHash = tx.nested ? tx.nested.safe_tx_hash : tx.safe_tx_hash;
    if (ledgerHash !== txHash.toLowerCase()) {
        throw new Error(`Verified ${ledgerHash} but ${txHash} was requested`);
    }

    status('Loading verifier...');
    await loadWasm();

    const out = window.txvVerify(JSON.stringify(tx));
    if (out.error) throw new Error(out.error);

    for (const check of out.contractChecks) {
        assertVersionSupported(check.label === 'ledger' ? 'signing' : 'nested', check.safeVersion);
    }

    status('Reading hashes from the Safe contract...');
    let html = '';
    let mismatch = false;
    let unknownIntent = false;

    for (const check of out.contractChecks) {
        // Requiring the recomputation to equal the hash the fields were fetched under is what
        // makes the pair below the signer's, not a consistent pair from another transaction.
        const expected = check.label === 'ledger' ? ledgerHash : tx.safe_tx_hash;
        if (check.approveHash.toLowerCase() !== expected) {
            throw new Error(
                `DO NOT SIGN. The ${check.label === 'ledger' ? 'signing' : 'approved'} transaction's ` +
                `fields hash to ${check.approveHash}, not to the requested ${expected}.`
            );
        }

        const onchain = await ethCall(tx.chain, check.target, check.calldata);
        const agree = onchain.domainHash.toLowerCase() === check.domainHash.toLowerCase()
            && onchain.messageHash.toLowerCase() === check.messageHash.toLowerCase();

        const call = normalizeCall(check.call);
        mismatch ||= !agree;
        unknownIntent ||= hasUnknownCall(call, true);
        html += renderCheck({ ...check, call }, onchain, tx.chain, agree, true);
    }

    if (!current()) return;
    VERIFY_DOM.panel.innerHTML = html;

    if (mismatch) {
        throw new Error('MISMATCH - DO NOT SIGN. The contract and the local recomputation disagree.');
    }
    status(unknownIntent
        ? 'DO NOT SIGN until every unknown function is independently decoded and confirmed.'
        : 'Compare the hashes above to your device before signing.');
}

async function showBuildInfo() {
    try {
        const info = await (await fetch('/build-info.json')).json();
        VERIFY_DOM.build.textContent =
            `build ${info.commit.slice(0, 12)} · wasm ${info.wasm_sha256.slice(0, 12)}`;
    } catch {
        VERIFY_DOM.build.textContent = '';
    }
}

for (const element of VERIFY_INPUTS) {
    element.addEventListener('input', invalidateResults);
    element.addEventListener('change', invalidateResults);
}
VERIFY_DOM.safe.addEventListener('input', updateSafePreview);
VERIFY_DOM.safe.addEventListener('change', updateSafePreview);

VERIFY_DOM.btn.addEventListener('click', async () => {
    invalidateResults();
    const run = generation;
    setInputsDisabled(true);
    try {
        await runVerify(run);
    } catch (error) {
        console.error(error);
        if (run === generation) DOM.status.textContent = `Error: ${error.message}`;
    } finally {
        setInputsDisabled(false);
    }
});

window.addEventListener('DOMContentLoaded', () => {
    const saved = loadLookup();
    if (saved.network) VERIFY_DOM.network.value = saved.network;
    if (saved.safe) VERIFY_DOM.safe.value = saved.safe;
    if (saved.nonce) VERIFY_DOM.nonce.value = saved.nonce;
    updateSafePreview();
    showBuildInfo();
});
