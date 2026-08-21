// A deliberately small RPC relay. The browser can use an operator-managed endpoint without
// exposing its URL or credentials, but this function never accepts an upstream from a requester.
const MAX_BODY_BYTES = 32 * 1024;
const MAX_RESPONSE_BYTES = 256 * 1024;
const RPC_TIMEOUT_MS = 4_000;

const UPSTREAM_ENV_BY_CHAIN = {
    1: 'RPC_UPSTREAM_ETHEREUM',
    10: 'RPC_UPSTREAM_OP_MAINNET',
    8453: 'RPC_UPSTREAM_BASE',
    11155111: 'RPC_UPSTREAM_SEPOLIA',
};

function configuredUpstream(chainId, environment = process.env) {
    const value = environment[UPSTREAM_ENV_BY_CHAIN[chainId]];
    if (typeof value !== 'string' || value === '') return null;
    try {
        const url = new URL(value);
        return url.protocol === 'https:' ? value : null;
    } catch {
        return null;
    }
}

function response(statusCode, message) {
    return {
        statusCode,
        headers: { 'content-type': 'application/json; charset=utf-8', 'cache-control': 'no-store' },
        body: JSON.stringify({ jsonrpc: '2.0', error: { code: -32000, message } }),
    };
}

function isObject(value) {
    return value !== null && typeof value === 'object' && !Array.isArray(value);
}

function hasOnlyKeys(value, keys) {
    return Object.keys(value).every(key => keys.includes(key));
}

function validRequest(request) {
    if (!isObject(request) || request.jsonrpc !== '2.0' || !Object.hasOwn(request, 'id') ||
        !['string', 'number'].includes(typeof request.id) || !Array.isArray(request.params)) {
        return false;
    }
    if (request.method === 'eth_chainId') return request.params.length === 0;
    if (request.method !== 'eth_call' || request.params.length !== 2 || request.params[1] !== 'latest') {
        return false;
    }
    const call = request.params[0];
    return isObject(call) && hasOnlyKeys(call, ['to', 'data']) &&
        typeof call.to === 'string' && /^0x[0-9a-fA-F]{40}$/.test(call.to) &&
        typeof call.data === 'string' && /^0x(?:[0-9a-fA-F]{2})*$/.test(call.data);
}

async function responseTextWithinLimit(upstreamResponse) {
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
    return new TextDecoder().decode(Buffer.concat(chunks));
}

async function handler(event) {
    if (event.httpMethod !== 'POST') {
        return { ...response(405, 'POST required'), headers: { allow: 'POST', 'content-type': 'application/json; charset=utf-8', 'cache-control': 'no-store' } };
    }

    const chainId = Number(event.queryStringParameters?.chain);
    if (!Object.hasOwn(UPSTREAM_ENV_BY_CHAIN, chainId)) return response(404, 'unsupported RPC chain');
    const upstream = configuredUpstream(chainId);
    if (!upstream) return response(503, 'RPC unavailable');

    let body;
    try {
        const encoded = event.body || '';
        body = event.isBase64Encoded ? Buffer.from(encoded, 'base64').toString('utf8') : encoded;
        if (Buffer.byteLength(body, 'utf8') > MAX_BODY_BYTES) return response(413, 'request too large');
        body = JSON.parse(body);
    } catch {
        return response(400, 'invalid JSON-RPC request');
    }
    if (!validRequest(body)) return response(400, 'unsupported JSON-RPC request');

    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), RPC_TIMEOUT_MS);
    try {
        const upstreamResponse = await fetch(upstream, {
            method: 'POST',
            headers: { 'content-type': 'application/json' },
            body: JSON.stringify(body),
            signal: controller.signal,
        });
        if (!upstreamResponse.ok) {
            return response(upstreamResponse.status === 429 || upstreamResponse.status >= 500
                ? upstreamResponse.status
                : 502, 'RPC unavailable');
        }
        const text = await responseTextWithinLimit(upstreamResponse);
        if (text === null) return response(502, 'RPC response too large');
        const json = JSON.parse(text);
        if (!isObject(json) || json.jsonrpc !== '2.0' || json.id !== body.id ||
            (!Object.hasOwn(json, 'result') && !Object.hasOwn(json, 'error'))) {
            return response(502, 'invalid RPC response');
        }
        return {
            statusCode: 200,
            headers: { 'content-type': 'application/json; charset=utf-8', 'cache-control': 'no-store' },
            body: JSON.stringify(json),
        };
    } catch (error) {
        return response(error?.name === 'AbortError' ? 504 : 502, 'RPC unavailable');
    } finally {
        clearTimeout(timer);
    }
}

module.exports = {
    handler,
    __test: { configuredUpstream, MAX_BODY_BYTES, MAX_RESPONSE_BYTES },
};
