/**
 * Verify mode.
 *
 * Shows a signer the two EIP-712 hashes their hardware wallet will display, plus the decoded
 * calldata, without involving the computer running the Safe UI.
 *
 * The hashes are read from the Safe contract's own encodeTransactionData via eth_call. The wasm
 * module recomputes them independently from the fetched parameters, so an RPC returning a wrong
 * preimage disagrees visibly instead of substituting a hash. The decode comes from
 * core/parsing.go compiled to wasm, so there is one decoder implementation rather than a
 * JavaScript copy that could drift from the CLI.
 *
 * Depends on globals from app.js: fetchWithRetries, fetchTransactionData, extractTransactionHash,
 * CHAIN_ID_TO_BASE_URL, DOM, state.
 */

// One endpoint per chain is enough: the wasm recomputes the same hashes from the fetched
// parameters, so a wrong preimage from an RPC shows up as a mismatch rather than a wrong hash.
const CHAIN_ID_TO_RPC = {
    1: 'https://ethereum-rpc.publicnode.com',
    10: 'https://optimism-rpc.publicnode.com',
    8453: 'https://base-rpc.publicnode.com',
    11155111: 'https://ethereum-sepolia-rpc.publicnode.com'
};

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
    const rpc = CHAIN_ID_TO_RPC[chainId];
    if (!rpc) throw new Error(`No RPC configured for chain ${chainId}`);

    const response = await fetch(rpc, {
        method: 'POST',
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({
            jsonrpc: '2.0',
            id: 1,
            method: 'eth_call',
            params: [{ to: target, data: calldata }, 'latest']
        })
    });
    if (!response.ok) throw new Error(`RPC ${response.status} ${response.statusText}`);

    const json = await response.json();
    if (json.error) throw new Error(`RPC error: ${json.error.message}`);
    return parsePreimage(json.result);
}

function row(label, value, warn) {
    return `<div class="verify-row${warn ? ' verify-warn' : ''}">` +
        `<span class="verify-label">${esc(label)}</span>` +
        `<span class="verify-value">${esc(value)}</span></div>`;
}

function block(title, text) {
    return `<h3>${esc(title)}</h3><pre class="verify-decode">${esc(text)}</pre>`;
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
    if (!CHAIN_ID_TO_RPC[chainId]) throw new Error('Select a supported network');

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

        html += check.label === 'ledger'
            ? '<h3>Compare these to your device</h3>'
            : '<h3>The nested transaction being approved</h3>';
        html += row('Network', `${NETWORK_NAMES[tx.chain]} (chain ${tx.chain})`);
        html += row('Safe', check.target);
        html += row('safeTxHash', check.approveHash);

        if (agree) {
            html += row('Domain hash', onchain.domainHash);
            html += row('Message hash', onchain.messageHash);
            html += row('Source', 'Safe contract and local recomputation agree');
        } else {
            mismatch = true;
            html += row('DO NOT SIGN', 'The Safe contract and the local recomputation disagree.', true);
            html += row('Domain (contract)', onchain.domainHash, true);
            html += row('Domain (local)', check.domainHash, true);
            html += row('Message (contract)', onchain.messageHash, true);
            html += row('Message (local)', check.messageHash, true);
        }

        html += block(
            check.label === 'ledger' ? 'What you are signing' : 'What is being approved',
            check.decode
        );
        html += block('Parameters for this Safe', check.params);

        const explorer = EXPLORERS[tx.chain];
        if (explorer) {
            html += `<p class="verify-note">To confirm without trusting this page, read ` +
                `<code>encodeTransactionData</code> on <a href="${esc(explorer + check.target)}` +
                `#readProxyContract">this Safe's contract page</a> (Read as Proxy) with the ` +
                `parameters directly above.</p>`;
        }
    }

    if (!current()) return;
    VERIFY_DOM.panel.innerHTML = html;

    if (mismatch) {
        throw new Error('MISMATCH - DO NOT SIGN. The contract and the local recomputation disagree.');
    }
    status('Compare the hashes above to your device before signing.');
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
    showBuildInfo();
});
