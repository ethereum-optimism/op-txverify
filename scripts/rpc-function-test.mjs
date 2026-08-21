#!/usr/bin/env node
// Dependency-free regression tests for the Netlify RPC allowlist function.
import assert from 'node:assert/strict';
import test from 'node:test';
import { createRequire } from 'node:module';
import { readFileSync } from 'node:fs';

const require = createRequire(import.meta.url);
const { handler, __test } = require('../netlify/functions/rpc.js');

const upstreams = {
  RPC_UPSTREAM_ETHEREUM: 'https://ethereum.example/rpc',
  RPC_UPSTREAM_OP_MAINNET: 'https://op.example/rpc',
  RPC_UPSTREAM_BASE: 'https://base.example/rpc',
  RPC_UPSTREAM_SEPOLIA: 'https://sepolia.example/rpc',
};

function event(chain, body, extra = {}) {
  return {
    httpMethod: 'POST',
    queryStringParameters: { chain: String(chain) },
    body: JSON.stringify(body),
    ...extra,
  };
}

async function withEnvironment(overrides, run) {
  const previous = new Map();
  for (const [key, value] of Object.entries({ ...upstreams, ...overrides })) {
    previous.set(key, process.env[key]);
    if (value === undefined) delete process.env[key];
    else process.env[key] = value;
  }
  try {
    return await run();
  } finally {
    for (const [key, value] of previous) {
      if (value === undefined) delete process.env[key];
      else process.env[key] = value;
    }
  }
}

async function withFetch(fake, run) {
  const previous = globalThis.fetch;
  globalThis.fetch = fake;
  try {
    return await run();
  } finally {
    globalThis.fetch = previous;
  }
}

test('only exposes configured upstreams for supported chains', () => {
  assert.equal(__test.configuredUpstream(1, upstreams), upstreams.RPC_UPSTREAM_ETHEREUM);
  assert.equal(__test.configuredUpstream(10, upstreams), upstreams.RPC_UPSTREAM_OP_MAINNET);
  assert.equal(__test.configuredUpstream(8453, upstreams), upstreams.RPC_UPSTREAM_BASE);
  assert.equal(__test.configuredUpstream(11155111, upstreams), upstreams.RPC_UPSTREAM_SEPOLIA);
  assert.equal(__test.configuredUpstream(999, upstreams), null);
});

test('Netlify routes the fixed same-origin path and documents every deployment environment variable', () => {
  const config = readFileSync(new URL('../netlify.toml', import.meta.url), 'utf8');
  assert.match(config, /from = "\/rpc\/:chain"/);
  assert.match(config, /to = "\/\.netlify\/functions\/rpc\?chain=:chain"/);
  for (const name of Object.keys(upstreams)) assert.match(config, new RegExp(name));
});

test('forwards a valid eth_chainId request to the configured chain upstream', async () => {
  await withEnvironment({}, async () => {
    let request;
    await withFetch(async (url, options) => {
      request = { url, options };
      return new Response(JSON.stringify({ jsonrpc: '2.0', id: 1, result: '0xa' }), { status: 200 });
    }, async () => {
      const response = await handler(event(10, {
        jsonrpc: '2.0', id: 1, method: 'eth_chainId', params: [],
      }));
      assert.equal(response.statusCode, 200);
      assert.equal(request.url, upstreams.RPC_UPSTREAM_OP_MAINNET);
      assert.equal(JSON.parse(request.options.body).method, 'eth_chainId');
      assert.equal(JSON.parse(response.body).result, '0xa');
    });
  });
});

test('treats a missing deployment upstream as availability so the browser may use its fallback', async () => {
  await withEnvironment({ RPC_UPSTREAM_OP_MAINNET: undefined }, async () => {
    const response = await handler(event(10, {
      jsonrpc: '2.0', id: 1, method: 'eth_chainId', params: [],
    }));
    assert.equal(response.statusCode, 503);
  });
});

test('rejects methods and eth_call shapes outside the small read-only allowlist', async () => {
  await withEnvironment({}, async () => {
    const forbidden = await handler(event(10, {
      jsonrpc: '2.0', id: 1, method: 'eth_sendRawTransaction', params: ['0xdead'],
    }));
    assert.equal(forbidden.statusCode, 400);

    const malformedCall = await handler(event(10, {
      jsonrpc: '2.0', id: 1, method: 'eth_call', params: [{ to: 'not-an-address', data: '0x' }, 'latest'],
    }));
    assert.equal(malformedCall.statusCode, 400);
  });
});

test('rejects non-POST, oversized, and non-JSON requests before contacting an upstream', async () => {
  await withEnvironment({}, async () => {
    let called = false;
    await withFetch(async () => { called = true; throw new Error('must not fetch'); }, async () => {
      assert.equal((await handler(event(10, {}, { httpMethod: 'GET' }))).statusCode, 405);
      assert.equal((await handler(event(10, {}, { body: '{' }))).statusCode, 400);
      assert.equal((await handler(event(10, {}, { body: 'x'.repeat(__test.MAX_BODY_BYTES + 1) }))).statusCode, 413);
      assert.equal(called, false);
    });
  });
});

test('caps oversized upstream responses and never returns a configured credential or error detail', async () => {
  const secret = 'https://user:secret@example.invalid/rpc';
  await withEnvironment({ RPC_UPSTREAM_OP_MAINNET: secret }, async () => {
    await withFetch(async () => new Response('x'.repeat(__test.MAX_RESPONSE_BYTES + 1), { status: 200 }), async () => {
      const response = await handler(event(10, {
        jsonrpc: '2.0', id: 1, method: 'eth_chainId', params: [],
      }));
      assert.equal(response.statusCode, 502);
      assert.equal(response.body.includes(secret), false);
      assert.equal(response.body.includes('secret'), false);
    });
  });
});

test('maps a timed-out upstream to a generic availability response', async () => {
  await withEnvironment({}, async () => {
    await withFetch(async () => { throw new DOMException('timed out', 'AbortError'); }, async () => {
      const response = await handler(event(10, {
        jsonrpc: '2.0', id: 1, method: 'eth_chainId', params: [],
      }));
      assert.equal(response.statusCode, 504);
      assert.equal(response.body.includes('timed out'), false);
    });
  });
});
