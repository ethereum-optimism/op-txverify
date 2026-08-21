#!/usr/bin/env node
// Regression tests for the pure functions in core/web/app.js. It is a classic script written for a
// browser, so it is evaluated with minimal stubs. These cover the places where a defect would feed
// a signer a wrong value rather than an error.
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const web = join(dirname(fileURLToPath(import.meta.url)), '..', 'core', 'web');

// Whatever reaches fflate.strToU8 is what gets compressed into the QR codes, so capturing it there
// and aborting is enough: the Start handler reports the abort like any other failure.
const encoded = [];
const fflate = { strToU8: (s) => { encoded.push(s); throw new Error('web-test: stop after capture'); } };

// Canned Safe transaction service responses, replaced per case. An unregistered URL answers 404,
// which the chain loop reads as "not on this chain" without retrying.
let api = {};
let clipboard = '';

// Handlers are kept per type, and dispatchEvent returns what the handler returned, so awaiting a
// click still awaits the async work it starts.
const stubEl = () => ({
  value: '', textContent: '', innerHTML: '', disabled: false, style: {}, handlers: {},
  addEventListener(type, fn) { (this.handlers[type] ||= []).push(fn); },
  dispatchEvent(event) {
    let result;
    for (const fn of this.handlers[event.type] || []) result = fn(event);
    return result;
  },
  click() { return this.dispatchEvent({ type: 'click' }); }
});
// A registered response is a body, or { status } for a service that answered with a failure.
const respond = (url) => {
  const entry = api[url];
  if (entry === undefined) return { ok: false, status: 404, statusText: 'Not Found' };
  if (entry.status) return { ok: false, status: entry.status, statusText: 'Service Unavailable' };
  return { ok: true, json: async () => entry, arrayBuffer: async () => new ArrayBuffer(0) };
};
const globals = {
  document: { getElementById: stubEl },
  window: {
    addEventListener() {}, location: { search: '' },
    Go: class { constructor() { this.importObject = {}; } run() {} },
  },
  WebAssembly: { instantiate: async () => ({ instance: {} }) },
  URLSearchParams,
  Event,
  console: { log() {}, warn() {}, error() {} },
  fetch: async (url) => respond(url),
  // Retry backoff without the wait: five attempts of real exponential backoff is 15 seconds.
  setTimeout: (fn) => setTimeout(fn, 0),
  navigator: { clipboard: { readText: async () => clipboard } },
  TextDecoder,
  fflate,
  localStorage: { getItem: () => null, setItem() {} },
};

const exported = [
  'extractTransactionHash', 'checkUrlForTransactionData', 'assertExactInteger', 'assertOperation',
  'state', 'DOM', 'extractSafeChainId', 'parsePreimage', 'esc', 'assertVersionSupported',
  'assertTxHash', 'VERIFY_DOM', 'setInputsDisabled',
];
// verify.js is a second classic script that reads app.js's globals, so both are evaluated together.
const source = readFileSync(join(web, 'app.js'), 'utf8') + '\n' + readFileSync(join(web, 'verify.js'), 'utf8');
const fns = new Function(...Object.keys(globals), `${source}\nreturn { ${exported.join(', ')}, ` +
  'renderCheck: typeof renderCheck === \'function\' ? renderCheck : undefined, ' +
  'renderAction: typeof renderCall === \'function\' ? renderCall : undefined, ' +
  'renderRawDetails: typeof renderRawDetails === \'function\' ? renderRawDetails : undefined };')(
  ...Object.values(globals)
);

let failures = 0;
const report = (name, ok, got, want) => {
  if (!ok) { failures++; console.log(`FAIL  ${name}\n        got  ${got}\n        want ${want}`); }
  else console.log(`PASS  ${name}`);
};
// The types must match too, so a string '0' cannot pass where the number 0 is required.
const check = (name, got, want) =>
  report(name, typeof got === typeof want && String(got) === String(want), got, want);
// Matching the message keeps a rejection for the wrong reason from reading as a pass.
const checkMessage = (name, got, want) =>
  report(name, String(got).includes(want), got, `a message containing: ${want}`);
const checkContains = (name, got, parts) =>
  report(name, parts.every(part => String(got).includes(part)), got, `HTML containing: ${parts.join(', ')}`);
const rejection = (fn) => { try { fn(); return 'no error thrown'; } catch (error) { return error.message; } };
const checkOrder = (name, html, parts) => {
  const positions = parts.map(part => String(html).indexOf(part));
  report(name, positions.every((position, index) =>
    position >= 0 && (index === 0 || position > positions[index - 1])),
  positions.join(', '), `increasing positions for: ${parts.join(', ')}`);
};
const contrastRatio = (foreground, background) => {
  const luminance = (hex) => {
    const channels = hex.match(/[0-9a-f]{2}/gi).map(channel => parseInt(channel, 16) / 255)
      .map(channel => channel <= 0.04045 ? channel / 12.92 : ((channel + 0.055) / 1.055) ** 2.4);
    return 0.2126 * channels[0] + 0.7152 * channels[1] + 0.0722 * channels[2];
  };
  const [lighter, darker] = [luminance(foreground), luminance(background)].sort((a, b) => b - a);
  return (lighter + 0.05) / (darker + 0.05);
};

// The page auto-starts without awaiting, so give that work a turn before asserting on it.
const settle = () => new Promise(resolve => setTimeout(resolve, 0));
const b64 = (text) => Buffer.from(text, 'utf8').toString('base64');

// At a 390px viewport the transaction input has room for 19 monospace characters beside the paste
// button. Longer placeholder text is clipped before the signer can read it.
const page = readFileSync(join(web, 'index.html'), 'utf8');
const txPlaceholder = page.match(/id="txInput"[^>]*placeholder="([^"]*)"/)[1];
report('the transaction placeholder fits on a phone', txPlaceholder.length <= 19,
  `${txPlaceholder.length} characters`, 'at most 19 characters');

// The verifier lookup is a distinct form: sharing the landing page's flex row made its second and
// third controls disappear off a phone-sized viewport. Keep its visible labels and responsive
// layout as an inexpensive guard alongside the browser containment check.
const lookupMarkup = page.match(/<div class="verify-lookup-fields">([\s\S]*?)<\/div>/)?.[1] || '';
report('verifier lookup controls have dedicated visible field labels',
  ['Network', 'Safe address', 'Nonce'].every(label =>
    lookupMarkup.includes(`<span class="verify-lookup-label">${label}</span>`)) &&
  ['verifyNetwork', 'verifySafe', 'verifyNonce'].every(id => lookupMarkup.includes(`id="${id}"`)),
  lookupMarkup, 'Network, Safe address, and Nonce labels with the existing control ids');

// A FastLZ control byte below 32 is a literal run of control+1 bytes, so runs of at most 32 bytes
// encode any text as itself. Enough to exercise the ?txz= path without a compressor.
const fastLZLiterals = (text) => {
  const bytes = Buffer.from(text, 'utf8');
  const out = [];
  for (let i = 0; i < bytes.length; i += 32) {
    const run = bytes.subarray(i, i + 32);
    out.push(run.length - 1, ...run);
  }
  return Buffer.from(out).toString('base64url');
};

const openLink = async (search) => {
  encoded.length = 0;
  api = {};
  fns.state.directPayloadText = null;
  fns.DOM.txInput.value = '';
  globals.window.location.search = search;
  fns.checkUrlForTransactionData();
  await settle();
};

// A supplied ?tx=/?txz= payload is forwarded to the QR encoder as the verbatim text that was
// decoded, never parsed and re-serialized: JSON.parse cannot hold an integer above 2^53-1, so a
// round-trip would put a value in the QR codes that the link never carried.
for (const payload of [
  '{"value":1000000000000000001,"nonce":3}',
  '{"value":115792089237316195423570985008687907853269984665640564039457584007913129639935}',
  '{"value":1e18}',
  '{"meta":{"value":1000000000000000001},"value":2}',
  '{ "value" : 42 }',
  '{"value":1,"n":"café"}',
]) {
  await openLink(`?tx=${b64(payload)}`);
  check(`?tx= forwarded verbatim: ${payload.slice(0, 40)}`, encoded[0], payload);
}
await openLink(`?txz=${fastLZLiterals('{"value":1000000000000000001,"n":"café"}')}`);
check('?txz= forwarded verbatim', encoded[0], '{"value":1000000000000000001,"n":"café"}');

// Both base64 alphabets are accepted, as the CLI accepts both. URLSearchParams decodes the + of a
// standard-alphabet payload to a space, and atob strips whitespace instead of failing, so a page
// that did not restore it would decode different bytes without an error.
const ALPHABET_PAYLOAD = '{"value":"1","memo":"xx~"}';
for (const [alphabet, encoding] of [['standard', 'base64'], ['url-safe', 'base64url']]) {
  await openLink(`?tx=${Buffer.from(ALPHABET_PAYLOAD, 'utf8').toString(encoding)}`);
  check(`?tx= accepts the ${alphabet} alphabet`, encoded[0], ALPHABET_PAYLOAD);
}

// A payload that is not a JSON object, or is not valid UTF-8, must leave the page unarmed rather
// than arm it with something the QR codes would carry as if the link had said it.
for (const payload of ['{"value":', 'null', '7', '[{"value":1}]']) {
  await openLink(`?tx=${b64(payload)}`);
  check(`refused: ${payload}`, `${!!fns.state.directPayloadText} ${encoded.length}`, 'false 0');
}
await openLink(`?tx=${Buffer.from('7b2261223a22fffe227d', 'hex').toString('base64')}`);
check('refused: invalid UTF-8', `${!!fns.state.directPayloadText} ${encoded.length}`, 'false 0');

/**
 * The Safe transaction service path. The v2 endpoint returns the integer fields as decimal strings.
 */
const TX_SERVICE = 'https://api.safe.global/tx-service/eth/api';
const ZERO_ADDRESS = `0x${'0'.repeat(40)}`;
const OUTER_SAFE = `0x${'a'.repeat(40)}`;
const INNER_SAFE = `0x${'b'.repeat(40)}`;
const HASH = `0x${'1'.repeat(64)}`;
const INNER_HASH = `0x${'2'.repeat(64)}`;

// safeTxHash is the hash the response is served under, which the page holds it to.
const safeTx = (hash, over) => ({
  safeTxHash: hash,
  safe: INNER_SAFE, to: `0x${'c'.repeat(40)}`, value: '1000000000000000001', data: '0x',
  operation: 0, safeTxGas: '0', baseGas: '0', gasPrice: '0', gasToken: ZERO_ADDRESS,
  refundReceiver: ZERO_ADDRESS, nonce: '7', ...over,
});
const plainApi = (over) => ({
  [`${TX_SERVICE}/v2/multisig-transactions/${HASH}/`]: safeTx(HASH, over),
  [`${TX_SERVICE}/v1/safes/${INNER_SAFE}/`]: { version: '1.4.1' },
});
// An approveHash of INNER_HASH, which the page follows to the transaction being approved.
const nestedApi = (over) => ({
  [`${TX_SERVICE}/v2/multisig-transactions/${HASH}/`]: safeTx(HASH, {
    safe: OUTER_SAFE, data: `0xd4d9bdcd${INNER_HASH.slice(2)}`, value: '0', nonce: '4', ...over
  }),
  [`${TX_SERVICE}/v2/multisig-transactions/${INNER_HASH}/`]: safeTx(INNER_HASH),
  [`${TX_SERVICE}/v1/safes/${OUTER_SAFE}/`]: { version: '1.3.0' },
  [`${TX_SERVICE}/v1/safes/${INNER_SAFE}/`]: { version: '1.4.1' },
});

const start = async (typed, responses) => {
  encoded.length = 0;
  api = responses;
  fns.state.directPayloadText = null;
  fns.DOM.txInput.value = typed;
  await fns.DOM.startBtn.click();
};

await start(HASH, plainApi());
const fetched = JSON.parse(encoded[0]);
check("value is carried through as the API's decimal string", fetched.value, '1000000000000000001');
check('chain is the chain the transaction was found on', fetched.chain, 1);
check('safe_version comes from the Safe the transaction belongs to', fetched.safe_version, '1.4.1');
check('nonce is parsed', fetched.nonce, 7);
check('a plain transaction has nothing nested', fetched.nested, null);

await start(HASH, nestedApi());
const nested = JSON.parse(encoded[0]);
check('the approved transaction is what gets hashed', nested.safe, INNER_SAFE);
check('the approveHash transaction is carried as nested', nested.nested.safe, OUTER_SAFE);
check('the nested nonce is the approving Safe\'s', nested.nested.nonce, 4);
check('the nested version is the approving Safe\'s', nested.nested.safe_version, '1.3.0');

// The fields a response carries only claim to be the transaction that was asked for. Binding them
// to the hash they were fetched under is what stops a page showing green for a transaction nobody
// asked about, since any hashed field can be altered into a self-consistent one with a valid hash.
await start(HASH, plainApi());
check('the fetched fields are bound to the requested hash',
  JSON.parse(encoded[0]).safe_tx_hash, HASH.toLowerCase());

await start(HASH, nestedApi());
const boundNested = JSON.parse(encoded[0]);
check('the approved transaction is bound to the hash the calldata names',
  boundNested.safe_tx_hash, INNER_HASH.toLowerCase());
check('the approving transaction is bound to the requested hash',
  boundNested.nested.safe_tx_hash, HASH.toLowerCase());

const OTHER_HASH = `0x${'3'.repeat(64)}`;
for (const [name, responses, want] of [
  ['returned under a different hash', plainApi({ safeTxHash: OTHER_HASH }), 'different transaction'],
  ['returned with no hash at all', plainApi({ safeTxHash: undefined }), 'not a 32-byte hash'],
  ['approving a transaction the service answers with another', {
    ...nestedApi(),
    [`${TX_SERVICE}/v2/multisig-transactions/${INNER_HASH}/`]: safeTx(OTHER_HASH),
  }, 'different transaction'],
]) {
  await start(HASH, responses);
  checkMessage(`refuses fields ${name}`, fns.DOM.status.textContent, want);
  check(`nothing is encoded for fields ${name}`, encoded.length, 0);
}

// A service that did not answer is not a statement that the transaction does not exist, and a
// signer told "not found" would go looking for a transaction that is queued and fine.
await start(HASH, { [`${TX_SERVICE}/v2/multisig-transactions/${HASH}/`]: { status: 503 } });
checkMessage('a service failure is reported as a service failure',
  fns.DOM.status.textContent, 'did not answer');
check('a service failure encodes nothing', encoded.length, 0);
await start(HASH, {});
checkMessage('a genuine absence is still reported as not found',
  fns.DOM.status.textContent, 'Transaction not found');

for (const raw of ['0x', HASH.slice(0, -1), `${HASH}0`, '0xzz', '', undefined]) {
  checkMessage(`assertTxHash rejects ${JSON.stringify(raw)}`,
    rejection(() => fns.assertTxHash('transaction hash', raw)), 'not a 32-byte hash');
}
check('assertTxHash lowercases what it accepts',
  fns.assertTxHash('transaction hash', `0x${'A'.repeat(64)}`), `0x${'a'.repeat(64)}`);

// One case per assertion call site: each of these values would otherwise be hashed as a different
// number than the API reported, which is a wrong message hash the signer cannot see.
for (const [where, cannedApi] of [['', plainApi], ['nested ', nestedApi]]) {
  for (const [field, raw] of [
    ['safeTxGas', '9007199254740992'],
    ['safeTxGas', '1e18'],
    ['baseGas', '9007199254740992'],
    ['gasPrice', '9007199254740992'],
    ['nonce', '9007199254740992'],
    ['operation', null],
  ]) {
    // The nested fields come off the approveHash transaction, so only these two reach an assertion.
    if (where && !['nonce', 'operation'].includes(field)) continue;
    await start(HASH, cannedApi({ [field]: raw }));
    checkMessage(`refuses ${where}${field} ${raw}`, fns.DOM.status.textContent, `${field} is ${raw}`);
    check(`nothing is encoded for ${where}${field} ${raw}`, encoded.length, 0);
  }
}

// Arming must not put a payload where a hash the signer can read is displayed. A disabled input does
// not stop the paste handler assigning to it, so the box decides and typing over it disarms.
const ARMED = `{"safe":"0x${'e'.repeat(40)}","value":"1"}`;
await openLink(`?tx=${b64(ARMED)}`);
check('the link arms the page', encoded[0], ARMED);
encoded.length = 0;
api = plainApi();
clipboard = HASH;
await fns.DOM.pasteBtn.click();
await settle();
check('a paste is what the box shows', fns.DOM.txInput.value, HASH);
check('a paste replaces the armed payload rather than adding to it', encoded.length, 1);
checkMessage('the pasted hash is what reaches the QR encoder', encoded[0], `"safe":"${INNER_SAFE}"`);

// The Safe API's integer fields must be exactly representable, or fail loudly.
for (const raw of ['9007199254740992', '1e18', '-1', '1.5', '0x10', '', '  ']) {
  checkMessage(`assertExactInteger rejects ${JSON.stringify(raw)}`,
    rejection(() => fns.assertExactInteger('baseGas', raw)), `baseGas is ${raw}`);
}
check('assertExactInteger accepts 0', fns.assertExactInteger('baseGas', '0'), 0);
check('assertExactInteger accepts a normal value', fns.assertExactInteger('safeTxGas', '21000'), 21000);

check('assertOperation accepts CALL', fns.assertOperation('0'), 0);
check('assertOperation accepts DELEGATECALL', fns.assertOperation(1), 1);
// Number() coerces every one of these to 0 or 1, so the enum check alone would let them through.
for (const raw of [null, '', [], false, true, '1.0', '0x1', 2, -1]) {
  checkMessage(`assertOperation rejects ${JSON.stringify(raw)}`,
    rejection(() => fns.assertOperation(raw)), `operation is ${raw}`);
}

// Safe UI links carry the hash after the last underscore of the id parameter.
check('extracts the hash from a Safe UI link',
  fns.extractTransactionHash('https://app.safe.global/transactions/tx?safe=eth:0xAAA&id=multisig_0xAAA_0xbeef'),
  '0xbeef');
check('passes a bare hash through', fns.extractTransactionHash('  0xbeef  '), '0xbeef');

// --- verify.js ---------------------------------------------------------------------------------

// The raw eth_call return is ABI-encoded dynamic bytes, so the trailing 32 bytes are padding, not
// the message hash. Real response for Safe 0x847B5c17...9D92 nonce 65.
const preimage =
  '0x0000000000000000000000000000000000000000000000000000000000000020' +
  '0000000000000000000000000000000000000000000000000000000000000042' +
  '1901a4a9c312badf3fcaa05eafe5dc9bee8bd9316c78ee8b0bebe3115bb21b732672' +
  '334e4c2b39031202af9c3402f920fcaef14aaa65e0c39142808f09f4d9a97a92' +
  '000000000000000000000000000000000000000000000000000000000000';

check('domain hash', fns.parsePreimage(preimage).domainHash,
  '0xa4a9c312badf3fcaa05eafe5dc9bee8bd9316c78ee8b0bebe3115bb21b732672');
check('message hash', fns.parsePreimage(preimage).messageHash,
  '0x334e4c2b39031202af9c3402f920fcaef14aaa65e0c39142808f09f4d9a97a92');
check('trailing bytes are padding, not the message hash',
  preimage.slice(-64) === fns.parsePreimage(preimage).messageHash.slice(2), false);
checkMessage('refuses a wrong length word', rejection(() => fns.parsePreimage(preimage.replace('0042', '0041'))), '66-byte');
checkMessage('refuses a missing 0x1901 prefix', rejection(() => fns.parsePreimage(preimage.replace('1901a4a9', '1902a4a9'))), 'EIP-712');
checkMessage('refuses a truncated body whose length word lies',
  rejection(() => fns.parsePreimage('0x' + '0'.repeat(62) + '20' + '0'.repeat(62) + '42' + 'aa'.repeat(36))), '320-character');

// The chain must come from the Safe URL, never from whichever tx service answers first: Safe <= 1.2.0
// omits chainId from its domain separator, so the same fields on 2 chains give the same Ledger hashes.
const safeUrl = (sn) =>
  `https://app.safe.global/transactions/tx?safe=${sn}:0x847B5c174615B1B7fDF770882256e2D3E95b9D92&id=multisig_0x847B_0xabc`;
check('reads the eth prefix', fns.extractSafeChainId(safeUrl('eth')), 1);
check('reads the oeth prefix', fns.extractSafeChainId(safeUrl('oeth')), 10);
check('reads the base prefix', fns.extractSafeChainId(safeUrl('base')), 8453);
check('reads the sep prefix', fns.extractSafeChainId(safeUrl('sep')), 11155111);
check('no prefix yields null', fns.extractSafeChainId('0xabc'), null);
checkMessage('refuses an unsupported chain prefix', rejection(() => fns.extractSafeChainId(safeUrl('opsep'))), 'does not support');

// Safe 1.5.0 removed encodeTransactionData, and an unreadable version must fail closed.
for (const bad of ['1.5.0', '2.0.0', '', '1.x', '1']) {
  checkMessage(`refuses Safe version ${bad || '(empty)'}`,
    rejection(() => fns.assertVersionSupported('signing', bad)), 'encodeTransactionData');
}
for (const ok of ['1.4.1', '1.3.0', '1.1.1']) {
  let threw = false;
  try { fns.assertVersionSupported('signing', ok); } catch { threw = true; }
  check(`accepts Safe version ${ok}`, threw, false);
}

// ABI string arguments can carry arbitrary text, and rendering API-sourced data is this page's job.
check('escapes tags', fns.esc('</pre><script>x</script>'), '&lt;/pre&gt;&lt;script&gt;x&lt;/script&gt;');
check('escapes attribute-breaking quotes', fns.esc('" autofocus onfocus=alert(1) x="'),
  '&quot; autofocus onfocus=alert(1) x=&quot;');

// These fixtures call the production renderer used by runVerify. A string inspection here checks
// the signer-visible HTML without duplicating the renderer in a test-only DOM implementation.
const ETHERFI_SPOKE = '0xdffcC3536D932eb51Df51a7F5FA407c4270d5308';
const LOCAL_DOMAIN_HASH = `0x${'4'.repeat(64)}`;
const LOCAL_MESSAGE_HASH = `0x${'5'.repeat(64)}`;
const rendererCheck = (over = {}) => ({
  label: 'ledger', target: INNER_SAFE, safeVersion: '1.4.1', approveHash: HASH,
  domainHash: LOCAL_DOMAIN_HASH, messageHash: LOCAL_MESSAGE_HASH,
  call: {
    target: ETHERFI_SPOKE, targetLabel: 'ETHERFI SPOKE (PROXY)', functionName: 'withdraw',
    signature: 'withdraw(uint256,uint256,address)', operation: 'CALL',
    arguments: [
      { name: 'reserveId', type: 'uint256', value: '0' },
      { name: 'amount', type: 'uint256', value: '5000057' },
      { name: 'onBehalfOf', type: 'address', value: INNER_SAFE },
    ],
  },
  safeFields: {
    to: ETHERFI_SPOKE, value: '0', data: '0x0ad58d2f00', operation: '0', safeTxGas: '0',
    baseGas: '0', gasPrice: '0', gasToken: ZERO_ADDRESS, refundReceiver: ZERO_ADDRESS, nonce: '3',
  },
  ...over,
});
const render = (checkValue = rendererCheck(), onchain = {
  domainHash: LOCAL_DOMAIN_HASH, messageHash: LOCAL_MESSAGE_HASH,
}, agree = true) => fns.renderCheck?.(checkValue, onchain, 10, agree) || '';

report('production check renderer is available', typeof fns.renderCheck === 'function',
  typeof fns.renderCheck, 'function');
report('production action component renderer is available', typeof fns.renderAction === 'function',
  typeof fns.renderAction, 'function');
report('production raw-details component renderer is available', typeof fns.renderRawDetails === 'function',
  typeof fns.renderRawDetails, 'function');

const friendlyHTML = render();
const friendlyAction = fns.renderAction?.(rendererCheck().call) || '';
checkContains('EtherFi labels and values come from the production action component', friendlyAction, [
  '<article', '<dl', 'ETHERFI SPOKE (PROXY)', ETHERFI_SPOKE, 'withdraw',
  'withdraw(uint256,uint256,address)', 'CALL', 'reserveId', '>0<', 'amount', '5000057',
  'onBehalfOf', INNER_SAFE,
]);
report('known calls do not fall back to preformatted Go-map text', !friendlyHTML.includes('verify-decode') &&
  !friendlyHTML.includes('<pre'), friendlyHTML, 'semantic HTML without verify-decode or pre');

const recursiveHTML = render(rendererCheck({ call: {
  target: `0x${'c'.repeat(40)}`, targetLabel: 'Batch caller', functionName: 'aggregate3',
  signature: 'aggregate3((address,bool,bytes)[])', operation: 'DELEGATECALL', arguments: [], calls: [
    rendererCheck().call,
    {
      target: `0x${'d'.repeat(40)}`, functionName: 'setConfig', signature: 'setConfig((uint256,bool)[])',
      operation: 'CALL', arguments: [{ name: 'configs', type: '(uint256,bool)[]', value: [[
        { name: 'limit', type: 'uint256', value: '9007199254740993' },
        { name: 'enabled', type: 'bool', value: true },
      ]] }], calls: [{
        target: `0x${'e'.repeat(40)}`, functionName: 'pause', signature: 'pause()',
        operation: 'DELEGATECALL', arguments: [],
      }],
    },
  ],
} }));
checkContains('nested actions are recursive, numbered, precision-safe, and flag delegatecall', recursiveHTML, [
  'Action 1', 'Action 2', 'Action 2.1', 'DELEGATECALL', 'limit', '9007199254740993',
  'enabled', 'true', 'pause()',
]);

const malicious = '</dd><script>window.pwned=1</script><span title="';
const maliciousHTML = render(rendererCheck({ call: {
  target: ETHERFI_SPOKE, targetLabel: malicious, functionName: 'propose',
  signature: 'propose(address[],uint256[],bytes[],string,uint8)', operation: 'CALL',
  arguments: [{ name: 'description', type: 'string', value: malicious }],
} }));
report('every ABI and label string is escaped by the production renderer',
  !maliciousHTML.includes('<script>') && !maliciousHTML.includes('<span title="') &&
    maliciousHTML.includes('&lt;/dd&gt;&lt;script&gt;window.pwned=1&lt;/script&gt;'),
  maliciousHTML, 'escaped text with no injected script or attribute');

const unknownRaw = '0xdeadbeef00000001';
const unknownHTML = render(rendererCheck({ call: {
  target: ETHERFI_SPOKE, functionName: 'Unknown function', operation: 'CALL',
  selector: '0xdeadbeef', rawCalldata: unknownRaw, arguments: [],
} }));
checkContains('unknown calldata stays visible and fails intent verification clearly', unknownHTML, [
  'Unknown function', ETHERFI_SPOKE, '0xdeadbeef', unknownRaw,
  'hashes may agree', 'not established', 'DO NOT SIGN', 'independently decoded',
]);
const friendlyRawDetails = fns.renderRawDetails?.(rendererCheck().safeFields) || '';
report('unknown warning is not part of the production raw-details component',
  !friendlyRawDetails.includes('verify-intent-warning'), friendlyRawDetails,
  'Raw transaction fields without an intent warning');

const malformedKnownRaw = '0x13af4035'; // setOwner(address), missing the address argument.
const malformedKnownHTML = render(rendererCheck({ call: {
  target: ETHERFI_SPOKE, functionName: 'Unknown function', operation: 'CALL',
  selector: malformedKnownRaw, rawCalldata: malformedKnownRaw, arguments: [],
} }));
checkContains('known selector with truncated arguments uses the unknown warning renderer', malformedKnownHTML,
  ['Unknown function', malformedKnownRaw, 'DO NOT SIGN', 'independently decoded']);

for (const [name, call, visible] of [
  ['native transfer', {
    target: INNER_SAFE, functionName: 'Send native ETH', operation: 'CALL', arguments: [
      { name: 'recipient', type: 'address', value: INNER_SAFE },
      { name: 'value', type: 'uint256', value: '9007199254740993' },
    ],
  }, ['Send native ETH', 'recipient', INNER_SAFE, 'value', '9007199254740993']],
  ['no calldata', { target: INNER_SAFE, functionName: 'No calldata', operation: 'CALL', arguments: [] },
    ['No calldata', INNER_SAFE, 'CALL']],
]) {
  checkContains(`${name} has a clear signer-visible presentation`,
    render(rendererCheck({ call })), visible);
}

checkContains('Safe fields and calldata come from the production raw-details component', friendlyRawDetails, [
  '<details class="verify-raw" open>', '<summary>Raw transaction fields</summary>', '<code>to</code>', ETHERFI_SPOKE,
  '<code>value</code>', '<code>data</code>', '0x0ad58d2f00', '<code>operation</code>',
  '<code>safeTxGas</code>', '<code>baseGas</code>', '<code>gasPrice</code>', '<code>gasToken</code>',
  '<code>refundReceiver</code>', '<code>nonce</code>',
]);
checkOrder('assembled check keeps hashes, action, then raw details in order', friendlyHTML, [
  '<dt>Domain hash</dt>', '<dt>Message hash</dt>', '<dt>safeTxHash</dt>', friendlyAction,
  friendlyRawDetails,
]);

const mismatchHTML = render(rendererCheck(), {
  domainHash: `0x${'6'.repeat(64)}`, messageHash: `0x${'7'.repeat(64)}`,
}, false);
checkContains('hash mismatches remain DO NOT SIGN and show both sources', mismatchHTML, [
  'DO NOT SIGN', 'Safe contract', 'Local recomputation', LOCAL_DOMAIN_HASH, LOCAL_MESSAGE_HASH,
]);
checkOrder('mismatching hashes keep warning, hash rows, action, and raw details in order', mismatchHTML, [
  '<p class="verify-intent-warning">', '<dt>Domain hash</dt>', '<dt>Message hash</dt>',
  '<dt>safeTxHash</dt>', friendlyAction, friendlyRawDetails,
]);

const css = readFileSync(join(web, 'app.css'), 'utf8');
const dangerText = css.match(/--verify-danger-text:\s*(#[0-9a-f]{6})/i)?.[1] ?? '#ff0420';
report('small danger text meets WCAG AA on the warning background',
  contrastRatio(dangerText, '#fff4f5') >= 4.5,
  contrastRatio(dangerText, '#fff4f5').toFixed(2), 'at least 4.50');

const ETH_RPC = 'https://ethereum-rpc.publicnode.com';
const agreeingPreimage = (domainHash, messageHash) =>
  '0x' + '0'.repeat(62) + '20' + '0'.repeat(62) + '42' + '1901' +
  domainHash.slice(2) + messageHash.slice(2) + '0'.repeat(60);
const verifyMalformedCall = async (call) => {
  api = {
    ...plainApi({ value: '0', data: '0x' }),
    '/wasm/main.wasm': {},
    [ETH_RPC]: { result: agreeingPreimage(LOCAL_DOMAIN_HASH, LOCAL_MESSAGE_HASH) },
  };
  globals.window.txvVerify = () => ({
    contractChecks: [{ ...rendererCheck({ call }), calldata: '0x' }],
  });
  fns.state.directPayloadText = null;
  fns.DOM.txInput.value = HASH;
  fns.VERIFY_DOM.network.value = '1';
  fns.VERIFY_DOM.safe.value = '';
  fns.VERIFY_DOM.nonce.value = '';
  await fns.VERIFY_DOM.btn.click();
  return { status: fns.DOM.status.textContent, html: fns.VERIFY_DOM.panel.innerHTML };
};

for (const [name, malformedCall] of [
  ['null top-level call', null],
  ['falsey top-level function name', { ...rendererCheck().call, functionName: '' }],
  ['null nested call', { ...rendererCheck().call, calls: [null] }],
  ['falsey nested function name', {
    ...rendererCheck().call, calls: [{ ...rendererCheck().call, functionName: null }],
  }],
  ['non-array nested calls', { ...rendererCheck().call, calls: {} }],
  ['malformed argument', {
    ...rendererCheck().call, arguments: [{ name: null, type: 'uint256', value: '1' }],
  }],
  ['known call with missing arguments', (() => {
    const call = { ...rendererCheck().call };
    delete call.arguments;
    return call;
  })()],
  ['known call with null arguments', { ...rendererCheck().call, arguments: null }],
  ['known call with non-array arguments', { ...rendererCheck().call, arguments: {} }],
  ['native transfer with missing arguments', {
    target: INNER_SAFE, functionName: 'Send native ETH', operation: 'CALL',
  }],
  ['native transfer with null arguments', {
    target: INNER_SAFE, functionName: 'Send native ETH', operation: 'CALL', arguments: null,
  }],
  ['native transfer with non-array arguments', {
    target: INNER_SAFE, functionName: 'Send native ETH', operation: 'CALL', arguments: {},
  }],
]) {
  const result = await verifyMalformedCall(malformedCall);
  checkMessage(`${name} leaves final runVerify status at DO NOT SIGN`, result.status, 'DO NOT SIGN');
  report(`${name} never reports a successful hash comparison`,
    !result.status.includes('Compare the hashes'), result.status, 'status without Compare the hashes');
}

// A result describes the inputs it came from. Leaving it on screen after one of them changes shows
// a signer a green result for a transaction that is no longer the one in the box.
for (const [what, element, type] of [
  ['the hash', fns.DOM.txInput, 'input'],
  ['the network', fns.VERIFY_DOM.network, 'change'],
  ['the Safe', fns.VERIFY_DOM.safe, 'input'],
  ['the nonce', fns.VERIFY_DOM.nonce, 'input'],
]) {
  fns.VERIFY_DOM.panel.innerHTML = '<div>a previous result</div>';
  fns.DOM.status.textContent = 'Compare the hashes above to your device before signing.';
  element.dispatchEvent({ type });
  check(`changing ${what} drops the previous result`, fns.VERIFY_DOM.panel.innerHTML, '');
  check(`changing ${what} drops the previous status`, fns.DOM.status.textContent, '');
}

// The paste button writes to the input directly, which fires nothing on its own.
fns.VERIFY_DOM.panel.innerHTML = '<div>a previous result</div>';
clipboard = HASH;
api = plainApi();
await fns.DOM.pasteBtn.click();
await settle();
check('a paste drops the previous result', fns.VERIFY_DOM.panel.innerHTML, '');

// Nothing that feeds a hash stays editable while a run is reading it, the paste button included:
// it writes to the input, and a disabled input still accepts a programmatic write.
fns.setInputsDisabled(true);
check('verifying disables every input that feeds a hash',
  [fns.DOM.txInput, fns.DOM.startBtn, fns.DOM.pasteBtn, fns.VERIFY_DOM.network,
    fns.VERIFY_DOM.safe, fns.VERIFY_DOM.nonce, fns.VERIFY_DOM.btn].every(e => e.disabled), true);
fns.setInputsDisabled(false);
check('and re-enables them afterwards',
  [fns.DOM.txInput, fns.VERIFY_DOM.btn].some(e => e.disabled), false);

console.log(failures ? `\n${failures} failure(s)` : '\nweb-test: all checks passed');
process.exit(failures ? 1 : 0);
