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

const stubEl = () => ({
  value: '', textContent: '', innerHTML: '', disabled: false, style: {},
  addEventListener(_, fn) { this.handler = fn; },
  click() { return this.handler && this.handler(); }
});
const globals = {
  document: { getElementById: stubEl },
  window: { addEventListener() {}, location: { search: '' } },
  URLSearchParams,
  console: { log() {}, warn() {}, error() {} },
  fetch: async (url) => (url in api
    ? { ok: true, json: async () => api[url] }
    : { ok: false, status: 404, statusText: 'Not Found' }),
  setTimeout,
  navigator: { clipboard: { readText: async () => clipboard } },
  TextDecoder,
  fflate,
  localStorage: { getItem: () => null, setItem() {} },
};

const exported = [
  'extractTransactionHash', 'checkUrlForTransactionData', 'assertExactInteger', 'assertOperation',
  'state', 'DOM', 'extractSafeChainId', 'parsePreimage', 'esc', 'assertVersionSupported',
];
// verify.js is a second classic script that reads app.js's globals, so both are evaluated together.
const source = readFileSync(join(web, 'app.js'), 'utf8') + '\n' + readFileSync(join(web, 'verify.js'), 'utf8');
const fns = new Function(...Object.keys(globals), `${source}\nreturn { ${exported.join(', ')} };`)(
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
const rejection = (fn) => { try { fn(); return 'no error thrown'; } catch (error) { return error.message; } };

// The page auto-starts without awaiting, so give that work a turn before asserting on it.
const settle = () => new Promise(resolve => setTimeout(resolve, 0));
const b64 = (text) => Buffer.from(text, 'utf8').toString('base64');

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

const safeTx = (over) => ({
  safe: INNER_SAFE, to: `0x${'c'.repeat(40)}`, value: '1000000000000000001', data: '0x',
  operation: 0, safeTxGas: '0', baseGas: '0', gasPrice: '0', gasToken: ZERO_ADDRESS,
  refundReceiver: ZERO_ADDRESS, nonce: '7', ...over,
});
const plainApi = (over) => ({
  [`${TX_SERVICE}/v2/multisig-transactions/${HASH}/`]: safeTx(over),
  [`${TX_SERVICE}/v1/safes/${INNER_SAFE}/`]: { version: '1.4.1' },
});
// An approveHash of INNER_HASH, which the page follows to the transaction being approved.
const nestedApi = (over) => ({
  [`${TX_SERVICE}/v2/multisig-transactions/${HASH}/`]: safeTx({
    safe: OUTER_SAFE, data: `0xd4d9bdcd${INNER_HASH.slice(2)}`, value: '0', nonce: '4', ...over
  }),
  [`${TX_SERVICE}/v2/multisig-transactions/${INNER_HASH}/`]: safeTx(),
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

console.log(failures ? `\n${failures} failure(s)` : '\nweb-test: all checks passed');
process.exit(failures ? 1 : 0);
