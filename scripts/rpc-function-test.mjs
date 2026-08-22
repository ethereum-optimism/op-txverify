#!/usr/bin/env node
import assert from 'node:assert/strict';
import test from 'node:test';
import { readFileSync } from 'node:fs';
import relay, { config, createRelay, __test } from '../netlify/functions/rpc.mjs';

const upstreams = {
  RPC_UPSTREAM_ETHEREUM: 'https://ethereum.example/rpc',
  RPC_UPSTREAM_OP_MAINNET: 'https://op.example/rpc',
  RPC_UPSTREAM_BASE: 'https://base.example/rpc',
  RPC_UPSTREAM_SEPOLIA: 'https://sepolia.example/rpc',
};
const SELECTOR = '0xe86637db';

function rpcRequest(chain, body, options = {}) {
  const method = options.method || 'POST';
  const init = { method, headers: { 'content-type': 'application/json' }, ...options };
  if (method !== 'GET' && method !== 'HEAD') init.body = JSON.stringify(body);
  return [new Request(`https://op-txverify.example/rpc/${chain}`, init), { params: { chain: String(chain) } }];
}

async function withEnv(overrides, run) {
  const previous = new Map();
  for (const [key, value] of Object.entries({ ...upstreams, ...overrides })) {
    previous.set(key, process.env[key]);
    if (value === undefined) delete process.env[key]; else process.env[key] = value;
  }
  try { return await run(); } finally {
    for (const [key, value] of previous) {
      if (value === undefined) delete process.env[key]; else process.env[key] = value;
    }
  }
}

async function withFetch(fake, run) {
  const previous = globalThis.fetch;
  globalThis.fetch = fake;
  try { return await run(); } finally { globalThis.fetch = previous; }
}

test('uses the custom POST-only, rate-limited path with no legacy redirect', () => {
  assert.equal(config.path, '/rpc/:chain');
  assert.equal(config.method, 'POST');
  assert.deepEqual(config.rateLimit, { windowLimit: 60, windowSize: 60, aggregateBy: ['ip', 'domain'] });
  assert.equal(readFileSync(new URL('../netlify.toml', import.meta.url), 'utf8').includes('[[redirects]]'), false);
  assert.equal(readFileSync(new URL('../netlify/functions/rpc.mjs', import.meta.url), 'utf8').includes('console.'), false);
});

test('uses only configured HTTPS upstreams for supported chains', () => {
  assert.equal(__test.configuredUpstream(1, upstreams), upstreams.RPC_UPSTREAM_ETHEREUM);
  assert.equal(__test.configuredUpstream(10, upstreams), upstreams.RPC_UPSTREAM_OP_MAINNET);
  assert.equal(__test.configuredUpstream(8453, upstreams), upstreams.RPC_UPSTREAM_BASE);
  assert.equal(__test.configuredUpstream(11155111, upstreams), upstreams.RPC_UPSTREAM_SEPOLIA);
  assert.equal(__test.configuredUpstream(999, upstreams), null);
});

test('reports missing configuration as availability and rejects unsupported chains', async () => {
  await withEnv({ RPC_UPSTREAM_OP_MAINNET: undefined }, async () => {
    assert.equal((await relay(...rpcRequest(10, { jsonrpc: '2.0', id: 1, method: 'eth_chainId', params: [] }))).status, 503);
  });
  await withEnv({}, async () => {
    assert.equal((await relay(...rpcRequest(999, { jsonrpc: '2.0', id: 1, method: 'eth_chainId', params: [] }))).status, 404);
  });
});

test('returns only the upstream result and sends no redirects', async () => {
  await withEnv({}, async () => withFetch(async (url, options) => {
    assert.equal(url, upstreams.RPC_UPSTREAM_OP_MAINNET);
    assert.equal(options.redirect, 'error');
    return Response.json({ jsonrpc: '2.0', id: 1, result: '0xa' });
  }, async () => {
    const response = await relay(...rpcRequest(10, { jsonrpc: '2.0', id: 1, method: 'eth_chainId', params: [] }));
    assert.equal(response.status, 200);
    assert.deepEqual(await response.json(), { result: '0xa' });
  }));
});

test('allows only encodeTransactionData eth_call', async () => {
  await withEnv({}, async () => {
    const invalid = await relay(...rpcRequest(10, { jsonrpc: '2.0', id: 1, method: 'eth_call', params: [{ to: `0x${'a'.repeat(40)}`, data: '0x12345678' }, 'latest'] }));
    assert.equal(invalid.status, 400);
    await withFetch(async () => Response.json({ jsonrpc: '2.0', id: 1, result: '0x' }), async () => {
      const valid = await relay(...rpcRequest(10, { jsonrpc: '2.0', id: 1, method: 'eth_call', params: [{ to: `0x${'a'.repeat(40)}`, data: `${SELECTOR}00` }, 'latest'] }));
      assert.equal(valid.status, 200);
    });
  });
});

test('sanitizes valid-sized error envelopes containing an exact environment secret', async () => {
  const secret = 'https://user:exact-env-secret@example.invalid/rpc';
  await withEnv({ RPC_UPSTREAM_OP_MAINNET: secret }, async () => withFetch(async () => Response.json({
    jsonrpc: '2.0', id: 1, error: { code: 3, message: `execution reverted ${secret}`, data: secret },
  }), async () => {
    const response = await relay(...rpcRequest(10, { jsonrpc: '2.0', id: 1, method: 'eth_call', params: [{ to: `0x${'a'.repeat(40)}`, data: `${SELECTOR}00` }, 'latest'] }));
    assert.equal(response.status, 422);
    const text = await response.text();
    assert.equal(text.includes(secret), false);
    assert.equal(text.includes('execution reverted'), false);
  }));
});

test('maps only documented transient JSON-RPC errors to availability', async () => {
  await withEnv({}, async () => {
    for (const code of [-32002, -32005, -32603]) {
      await withFetch(async () => Response.json({ jsonrpc: '2.0', id: 1, error: { code, message: 'retry' } }), async () => {
        const response = await relay(...rpcRequest(10, { jsonrpc: '2.0', id: 1, method: 'eth_chainId', params: [] }));
        assert.equal(response.status, 503);
      });
    }
  });
});

test('treats all chainId errors and HTTP 408, 429, and 5xx as availability', async () => {
  await withEnv({}, async () => withFetch(async () => Response.json({
    jsonrpc: '2.0', id: 1, error: { code: 3, message: 'chain unavailable' },
  }), async () => {
    assert.equal((await relay(...rpcRequest(10, { jsonrpc: '2.0', id: 1, method: 'eth_chainId', params: [] }))).status, 503);
  }));
  for (const status of [408, 429, 503]) {
    await withEnv({}, async () => withFetch(async () => new Response('', { status }), async () => {
      assert.equal((await relay(...rpcRequest(10, { jsonrpc: '2.0', id: 1, method: 'eth_chainId', params: [] }))).status, status);
    }));
  }
});

test('rejects malformed and non-POST requests before reaching an upstream', async () => {
  await withEnv({}, async () => withFetch(async () => { throw new Error('must not fetch'); }, async () => {
    const [get, context] = rpcRequest(10, {}, { method: 'GET' });
    assert.equal((await relay(get, context)).status, 405);
    const invalid = await relay(...rpcRequest(10, { jsonrpc: '2.0', id: 1, method: 'eth_sendRawTransaction', params: [] }));
    assert.equal(invalid.status, 400);
  }));
});

test('enforces request and response size caps without exposing a configured secret', async () => {
  await withEnv({}, async () => withFetch(async () => { throw new Error('must not fetch'); }, async () => {
    const large = new Request('https://op-txverify.example/rpc/10', { method: 'POST', body: 'x'.repeat(__test.MAX_BODY_BYTES + 1) });
    assert.equal((await relay(large, { params: { chain: '10' } })).status, 413);
  }));

  const secret = 'https://user:response-secret@example.invalid/rpc';
  await withEnv({ RPC_UPSTREAM_OP_MAINNET: secret }, async () => withFetch(async () =>
    new Response('x'.repeat(__test.MAX_RESPONSE_BYTES + 1)), async () => {
      const response = await relay(...rpcRequest(10, { jsonrpc: '2.0', id: 1, method: 'eth_chainId', params: [] }));
      assert.equal(response.status, 502);
      assert.equal((await response.text()).includes(secret), false);
    }));
});

test('maps transport failures to generic availability without exposing upstream details', async () => {
  const secret = 'https://user:transport-secret@example.invalid/rpc';
  await withEnv({ RPC_UPSTREAM_OP_MAINNET: secret }, async () => withFetch(async () => {
    throw new Error(`cannot reach ${secret}`);
  }, async () => {
    const response = await relay(...rpcRequest(10, { jsonrpc: '2.0', id: 1, method: 'eth_chainId', params: [] }));
    assert.equal(response.status, 502);
    assert.equal((await response.text()).includes(secret), false);
  }));
});

test('uses an injected short timeout and waits for its AbortSignal', async () => {
  let aborted = false;
  const fastRelay = createRelay({ timeoutMs: 1 });
  await withEnv({}, async () => withFetch(async (_url, options) => new Promise((_resolve, reject) => {
    options.signal.addEventListener('abort', () => {
      aborted = true;
      reject(new DOMException('timed out', 'AbortError'));
    });
  }), async () => {
    const response = await fastRelay(...rpcRequest(10, { jsonrpc: '2.0', id: 1, method: 'eth_chainId', params: [] }));
    assert.equal(response.status, 504);
    assert.equal(aborted, true);
  }));
});
