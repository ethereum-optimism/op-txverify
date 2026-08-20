#!/usr/bin/env node
// Regression tests for the pure functions in core/web/app.js. It is a classic script written for a
// browser, so it is evaluated with minimal stubs. These cover the places where a defect would feed
// a signer a wrong value rather than an error.
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const web = join(dirname(fileURLToPath(import.meta.url)), '..', 'core', 'web');

const stubEl = () => ({ value: '', textContent: '', innerHTML: '', disabled: false, addEventListener() {}, click() {} });
const globals = {
  document: { getElementById: stubEl },
  window: { addEventListener() {}, location: { search: '' } },
  URLSearchParams,
  console,
  fetch: () => { throw new Error('no network in this harness'); },
  setTimeout,
  atob: (s) => Buffer.from(s, 'base64').toString('binary'),
  TextDecoder,
};

const exported = ['extractTransactionHash', 'checkUrlForTransactionData', 'state'];
const source = readFileSync(join(web, 'app.js'), 'utf8');
const fns = new Function(...Object.keys(globals), `${source}\nreturn { ${exported.join(', ')} };`)(
  ...Object.values(globals)
);

let failures = 0;
const check = (name, got, want) => {
  const ok = String(got) === String(want);
  if (!ok) { failures++; console.log(`FAIL  ${name}\n        got  ${got}\n        want ${want}`); }
  else console.log(`PASS  ${name}`);
};

// A supplied ?tx=/?txz= payload is forwarded to the QR encoder as the verbatim text that was
// decoded, never parsed and re-serialized. These two checks record why: JSON.parse cannot hold an
// integer above 2^53-1, so a round-trip puts a value in the QR codes that the link never carried.
const roundTripped = (json) => JSON.stringify(JSON.parse(json));

check('a round-trip corrupts a large integer, which is why the text is forwarded instead',
  roundTripped('{"value":1000000000000000001}'), '{"value":1000000000000000000}');
check('a round-trip also rewrites exponent notation',
  roundTripped('{"value":1e18}'), '{"value":1000000000000000000}');

// The property that matters: the text decoded from the link is what reaches the QR encoder.
for (const payload of [
  '{"value":1000000000000000001,"nonce":3}',
  '{"value":115792089237316195423570985008687907853269984665640564039457584007913129639935}',
  '{"value":1e18}',
  '{"meta":{"value":1000000000000000001},"value":2}',
  '{ "value" : 42 }',
]) {
  globals.window.location.search = `?tx=${Buffer.from(payload).toString('base64')}`;
  fns.state.directTransactionJson = null;
  fns.checkUrlForTransactionData();
  check(`forwarded verbatim: ${payload.slice(0, 40)}`, fns.state.directTransactionJson, payload);
}

// Safe UI links carry the hash after the last underscore of the id parameter.
check('extracts the hash from a Safe UI link',
  fns.extractTransactionHash('https://app.safe.global/transactions/tx?safe=eth:0xAAA&id=multisig_0xAAA_0xbeef'),
  '0xbeef');
check('passes a bare hash through', fns.extractTransactionHash('  0xbeef  '), '0xbeef');

console.log(failures ? `\n${failures} failure(s)` : '\nweb-test: all checks passed');
process.exit(failures ? 1 : 0);
