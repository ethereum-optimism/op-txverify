// The custom path in config keeps this relay off Netlify's legacy /.netlify/functions/* URL.
const MAX_BODY_BYTES = 32 * 1024;
const MAX_RESPONSE_BYTES = 256 * 1024;
const RPC_TIMEOUT_MS = 4_000;
const ENCODE_TRANSACTION_DATA_SELECTOR = '0xe86637db';
const TRANSIENT_RPC_ERROR_CODES = new Set([-32002, -32005, -32603]);

const UPSTREAM_ENV_BY_CHAIN = {
  1: 'RPC_UPSTREAM_ETHEREUM',
  10: 'RPC_UPSTREAM_OP_MAINNET',
  8453: 'RPC_UPSTREAM_BASE',
  11155111: 'RPC_UPSTREAM_SEPOLIA',
};

export const config = {
  path: '/rpc/:chain',
  method: 'POST',
  rateLimit: { windowLimit: 60, windowSize: 60, aggregateBy: ['ip', 'domain'] },
};

function configuredUpstream(chainId, environment = process.env) {
  const value = environment[UPSTREAM_ENV_BY_CHAIN[chainId]];
  if (typeof value !== 'string' || value === '') return null;
  try {
    return new URL(value).protocol === 'https:' ? value : null;
  } catch {
    return null;
  }
}

function response(status, error) {
  return Response.json({ error }, {
    status,
    headers: { 'cache-control': 'no-store' },
  });
}

function isObject(value) {
  return value !== null && typeof value === 'object' && !Array.isArray(value);
}

function hasOnlyKeys(value, keys) {
  return Object.keys(value).every((key) => keys.includes(key));
}

function validRequest(value) {
  if (!isObject(value) || value.jsonrpc !== '2.0' || !Object.hasOwn(value, 'id') ||
      !['string', 'number'].includes(typeof value.id) || !Array.isArray(value.params)) return false;
  if (value.method === 'eth_chainId') return value.params.length === 0;
  if (value.method !== 'eth_call' || value.params.length !== 2 || value.params[1] !== 'latest') return false;
  const call = value.params[0];
  return isObject(call) && hasOnlyKeys(call, ['to', 'data']) &&
    typeof call.to === 'string' && /^0x[0-9a-fA-F]{40}$/.test(call.to) &&
    typeof call.data === 'string' && /^0xe86637db(?:[0-9a-fA-F]{2})*$/i.test(call.data);
}

async function textWithinLimit(upstreamResponse) {
  const declaredLength = Number(upstreamResponse.headers.get('content-length'));
  if (Number.isFinite(declaredLength) && declaredLength > MAX_RESPONSE_BYTES) return null;
  const reader = upstreamResponse.body?.getReader();
  if (!reader) return '';
  const chunks = [];
  let length = 0;
  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    length += value.byteLength;
    if (length > MAX_RESPONSE_BYTES) {
      await reader.cancel();
      return null;
    }
    chunks.push(value);
  }
  const bytes = new Uint8Array(length);
  let offset = 0;
  for (const chunk of chunks) {
    bytes.set(chunk, offset);
    offset += chunk.byteLength;
  }
  return new TextDecoder().decode(bytes);
}

export function createRelay({ timeoutMs = RPC_TIMEOUT_MS } = {}) {
  return async function relay(request, context) {
  if (request.method !== 'POST') return response(405, 'POST required');
  const chainId = Number(context.params?.chain);
  if (!Object.hasOwn(UPSTREAM_ENV_BY_CHAIN, chainId)) return response(404, 'unsupported RPC chain');
  const upstream = configuredUpstream(chainId);
  if (!upstream) return response(503, 'RPC unavailable');

  let body;
  try {
    const declaredLength = Number(request.headers.get('content-length'));
    if (Number.isFinite(declaredLength) && declaredLength > MAX_BODY_BYTES) return response(413, 'request too large');
    const text = await request.text();
    if (new TextEncoder().encode(text).byteLength > MAX_BODY_BYTES) return response(413, 'request too large');
    body = JSON.parse(text);
  } catch {
    return response(400, 'invalid JSON-RPC request');
  }
  if (!validRequest(body)) return response(400, 'unsupported JSON-RPC request');

  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs);
  try {
    const upstreamResponse = await fetch(upstream, {
      method: 'POST',
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify(body),
      signal: controller.signal,
      redirect: 'error',
    });
    if (!upstreamResponse.ok) {
      return response(upstreamResponse.status === 408 || upstreamResponse.status === 429 || upstreamResponse.status >= 500
        ? upstreamResponse.status
        : 502, 'RPC unavailable');
    }
    const text = await textWithinLimit(upstreamResponse);
    if (text === null) return response(502, 'RPC response too large');
    const json = JSON.parse(text);
    if (!isObject(json) || json.jsonrpc !== '2.0' || json.id !== body.id ||
        (!Object.hasOwn(json, 'result') && !Object.hasOwn(json, 'error'))) return response(502, 'invalid RPC response');
    if (json.error) {
      const unavailable = body.method === 'eth_chainId' || TRANSIENT_RPC_ERROR_CODES.has(json.error.code);
      return response(unavailable ? 503 : 422, unavailable ? 'RPC unavailable' : 'RPC rejected request');
    }
    return Response.json({ result: json.result }, { headers: { 'cache-control': 'no-store' } });
  } catch (error) {
    return response(error?.name === 'AbortError' ? 504 : 502, 'RPC unavailable');
  } finally {
    clearTimeout(timer);
  }
  };
}

export default createRelay();
export const __test = { configuredUpstream, MAX_BODY_BYTES, MAX_RESPONSE_BYTES };
