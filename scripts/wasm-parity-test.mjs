#!/usr/bin/env node
// Proves the wasm build hashes identically to Go, so the page and the CLI can never silently
// diverge. Vector and expectations are the "Ethereum Mainnet Safe Tx 1" case in core/hashing_test.go.
import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const wasmDir = join(dirname(fileURLToPath(import.meta.url)), "..", "site", "wasm");

// wasm_exec.js is a classic script that assigns globalThis.Go, so it is evaluated, not imported.
new Function(readFileSync(join(wasmDir, "wasm_exec.js"), "utf8")).call(globalThis);

const go = new globalThis.Go();
const { instance } = await WebAssembly.instantiate(readFileSync(join(wasmDir, "main.wasm")), go.importObject);
// Not awaited: Go's main blocks forever to keep the export callable, so run() never resolves.
go.run(instance).catch((err) => {
  console.error("parity: wasm exited unexpectedly:", err);
  process.exit(1);
});

if (typeof globalThis.txvVerify !== "function") {
  console.error("parity: wasm did not register globalThis.txvVerify");
  process.exit(1);
}

const SAFE = "0x847B5c174615B1B7fDF770882256e2D3E95b9D92";
const ZERO = "0x0000000000000000000000000000000000000000";
const tx = {
  safe: SAFE,
  safe_version: "1.4.1",
  chain: 1,
  to: "0xcA11bde05977b3631167028862bE2a173976CA11",
  value: 0,
  data: "0x82ad56cb0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000200000000000000000000000005a0aae59d09fccbddb6c6cceb07b7279367c3d2a000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000000000000000024d4d9bdcd493ad64b8f788ed9808c7bf527a10a017d9f263bb7889868ce18b451d685762d00000000000000000000000000000000000000000000000000000000",
  operation: 1,
  safe_tx_gas: 0,
  base_gas: 0,
  gas_price: 0,
  gas_token: ZERO,
  refund_receiver: ZERO,
  nonce: 15,
};
const expected = {
  domainHash: "0xa4a9c312badf3fcaa05eafe5dc9bee8bd9316c78ee8b0bebe3115bb21b732672",
  messageHash: "0xa6d60aba6b1426cec097593348a9d36ed42ddd1ac52f1b05a5f46a9c0401a11a",
};

const out = globalThis.txvVerify(JSON.stringify(tx));
if (out?.error) {
  console.error(`parity: txvVerify returned an error: ${out.error}`);
  process.exit(1);
}
// The export may hand back the VerificationResult as an object or as a JSON string.
const result = typeof out?.result === "string" ? JSON.parse(out.result) : out?.result;

const failures = [];
for (const [field, want] of Object.entries(expected)) {
  const got = result?.[field];
  if (got !== want) failures.push(`${field}:\n  want ${want}\n  got  ${got}`);
}

// Pinned against `cast calldata "encodeTransactionData(...)"` for this vector. Asserting only that
// it is 0x-prefixed would let a wrong selector or a swapped safeTxGas/baseGas reach a signer as an
// unexplained on-chain mismatch.
const EXPECTED_CALLDATA =
  "0xe86637db000000000000000000000000ca11bde05977b3631167028862be2a173976ca1100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000140000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000f000000000000000000000000000000000000000000000000000000000000012482ad56cb0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000200000000000000000000000005a0aae59d09fccbddb6c6cceb07b7279367c3d2a000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000000000000000024d4d9bdcd493ad64b8f788ed9808c7bf527a10a017d9f263bb7889868ce18b451d685762d0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000";

const check = out?.contractChecks?.[0];
if (!check) {
  failures.push("contractChecks[0]: missing");
} else {
  if (check.calldata !== EXPECTED_CALLDATA) {
    failures.push(`contractChecks[0].calldata:\n  want ${EXPECTED_CALLDATA}\n  got  ${check.calldata}`);
  }
  if (check.target?.toLowerCase() !== SAFE.toLowerCase()) {
    failures.push(`contractChecks[0].target:\n  want ${SAFE}\n  got  ${check.target}`);
  }
  for (const field of ["domainHash", "messageHash", "decode", "params", "safeVersion"]) {
    if (!check[field]) failures.push(`contractChecks[0].${field}: missing`);
  }
  if (check.domainHash !== expected.domainHash || check.messageHash !== expected.messageHash) {
    failures.push("contractChecks[0] hashes disagree with the verification result");
  }
}

// A nested transaction is the superchain-ops norm and the most error-prone path: the "ledger" check
// must describe the child Safe that signs the approveHash, and "inner" the parent action. Getting
// this backwards would show a signer the wrong pair, so the ordering is pinned here.
const nestedTx = {
  ...tx,
  nested: {
    safe: "0x5a0Aae59D09fccBdDb6C6CcEB07B7279367C3d2A",
    safe_version: "1.3.0",
    nonce: 9,
    data: "0xd4d9bdcd493ad64b8f788ed9808c7bf527a10a017d9f263bb7889868ce18b451d685762d",
    operation: 0,
    to: SAFE,
  },
};
const nestedOut = globalThis.txvVerify(JSON.stringify(nestedTx));
if (nestedOut?.error) {
  failures.push(`nested vector returned an error: ${nestedOut.error}`);
} else {
  const [ledger, inner] = nestedOut.contractChecks ?? [];
  if (nestedOut.contractChecks?.length !== 2) {
    failures.push(`nested vector: want 2 contractChecks, got ${nestedOut.contractChecks?.length}`);
  } else {
    if (ledger.label !== "ledger" || inner.label !== "inner") {
      failures.push(`nested labels: want ledger,inner got ${ledger.label},${inner.label}`);
    }
    if (ledger.target.toLowerCase() !== nestedTx.nested.safe.toLowerCase()) {
      failures.push(`nested ledger.target:\n  want ${nestedTx.nested.safe}\n  got  ${ledger.target}`);
    }
    if (inner.target.toLowerCase() !== SAFE.toLowerCase()) {
      failures.push(`nested inner.target:\n  want ${SAFE}\n  got  ${inner.target}`);
    }
    if (ledger.messageHash === inner.messageHash) {
      failures.push("nested: ledger and inner share a message hash, so one pair is wrong");
    }
    if (ledger.safeVersion !== nestedTx.nested.safe_version) {
      failures.push(`nested ledger.safeVersion: want ${nestedTx.nested.safe_version} got ${ledger.safeVersion}`);
    }
  }
}

if (failures.length > 0) {
  console.error(`parity: ${failures.length} mismatch(es) against core/hashing_test.go\n${failures.join("\n")}`);
  process.exit(1);
}
console.log("parity: wasm matches core/hashing_test.go (domainHash, messageHash, contractChecks[0])");
process.exit(0);
