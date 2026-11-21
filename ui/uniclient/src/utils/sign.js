/**
 * 请求签名（方案 B）：与服务端 internal/security 保持同一套规范串。
 *
 * 小程序没有 WebCrypto，这里用纯 JS 实现 SHA-256 / HMAC-SHA256，
 * 以便 H5、微信小程序、APP 三端共用同一份代码。
 */

/** 需要签名的方法。 */
const SIGNED_METHODS = ['POST', 'PUT', 'PATCH', 'DELETE']

const K = [
  0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5, 0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5,
  0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3, 0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174,
  0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc, 0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da,
  0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7, 0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967,
  0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13, 0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85,
  0xa2bfe8a1, 0xa81a664b, 0xc24b8b70, 0xc76c51a3, 0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070,
  0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5, 0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3,
  0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208, 0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2,
]

/** 32 位右循环。 */
function rotr(value, bits) {
  return (value >>> bits) | (value << (32 - bits))
}

/** UTF-8 编码（小程序没有 TextEncoder）。 */
function utf8Bytes(text) {
  const bytes = []
  for (let index = 0; index < text.length; index += 1) {
    let code = text.codePointAt(index)
    if (code > 0xffff) index += 1
    if (code < 0x80) bytes.push(code)
    else if (code < 0x800) bytes.push(0xc0 | (code >> 6), 0x80 | (code & 0x3f))
    else if (code < 0x10000) bytes.push(0xe0 | (code >> 12), 0x80 | ((code >> 6) & 0x3f), 0x80 | (code & 0x3f))
    else
      bytes.push(
        0xf0 | (code >> 18),
        0x80 | ((code >> 12) & 0x3f),
        0x80 | ((code >> 6) & 0x3f),
        0x80 | (code & 0x3f),
      )
  }
  return bytes
}

/** SHA-256，返回 32 字节数组。 */
export function sha256Bytes(bytes) {
  const length = bytes.length
  const words = []
  const padded = bytes.slice()
  padded.push(0x80)
  while (padded.length % 64 !== 56) padded.push(0)
  const bitLen = length * 8
  for (let shift = 56; shift >= 0; shift -= 8) {
    padded.push(shift >= 32 ? Math.floor(bitLen / 2 ** shift) & 0xff : Math.floor(bitLen / 2 ** shift) & 0xff)
  }

  const hash = [
    0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a, 0x510e527f, 0x9b05688c, 0x1f83d9ab, 0x5be0cd19,
  ]
  const w = new Array(64)

  for (let offset = 0; offset < padded.length; offset += 64) {
    for (let index = 0; index < 16; index += 1) {
      const pos = offset + index * 4
      w[index] = ((padded[pos] << 24) | (padded[pos + 1] << 16) | (padded[pos + 2] << 8) | padded[pos + 3]) | 0
    }
    for (let index = 16; index < 64; index += 1) {
      const s0 = rotr(w[index - 15], 7) ^ rotr(w[index - 15], 18) ^ (w[index - 15] >>> 3)
      const s1 = rotr(w[index - 2], 17) ^ rotr(w[index - 2], 19) ^ (w[index - 2] >>> 10)
      w[index] = (w[index - 16] + s0 + w[index - 7] + s1) | 0
    }
    let [a, b, c, d, e, f, g, h] = hash
    for (let index = 0; index < 64; index += 1) {
      const S1 = rotr(e, 6) ^ rotr(e, 11) ^ rotr(e, 25)
      const ch = (e & f) ^ (~e & g)
      const temp1 = (h + S1 + ch + K[index] + w[index]) | 0
      const S0 = rotr(a, 2) ^ rotr(a, 13) ^ rotr(a, 22)
      const maj = (a & b) ^ (a & c) ^ (b & c)
      const temp2 = (S0 + maj) | 0
      h = g
      g = f
      f = e
      e = (d + temp1) | 0
      d = c
      c = b
      b = a
      a = (temp1 + temp2) | 0
    }
    hash[0] = (hash[0] + a) | 0
    hash[1] = (hash[1] + b) | 0
    hash[2] = (hash[2] + c) | 0
    hash[3] = (hash[3] + d) | 0
    hash[4] = (hash[4] + e) | 0
    hash[5] = (hash[5] + f) | 0
    hash[6] = (hash[6] + g) | 0
    hash[7] = (hash[7] + h) | 0
  }

  const out = []
  hash.forEach((value) => out.push((value >>> 24) & 0xff, (value >>> 16) & 0xff, (value >>> 8) & 0xff, value & 0xff))
  return out
}

/** 十六进制字符串转字节数组。 */
export function hexToBytes(hex) {
  const bytes = []
  for (let index = 0; index < hex.length; index += 2) bytes.push(parseInt(hex.slice(index, index + 2), 16))
  return bytes
}

/** 字节数组转十六进制。 */
export function bytesToHex(bytes) {
  return bytes.map((item) => item.toString(16).padStart(2, '0')).join('')
}

/** HMAC-SHA256，返回十六进制串。 */
export function hmacSha256Hex(keyHex, message) {
  const key = hexToBytes(keyHex)
  let block = key.length > 64 ? sha256Bytes(key) : key.slice()
  while (block.length < 64) block.push(0)
  const inner = block.map((item) => item ^ 0x36)
  const outer = block.map((item) => item ^ 0x5c)
  const innerHash = sha256Bytes(inner.concat(utf8Bytes(message)))
  return bytesToHex(sha256Bytes(outer.concat(innerHash)))
}

/** SHA-256 十六进制摘要。 */
export function sha256Hex(text) {
  return bytesToHex(sha256Bytes(utf8Bytes(text)))
}

/** 查询参数排序，与服务端一致。 */
export function sortedQuery(query) {
  if (!query) return ''
  const pairs = query.split('&').filter(Boolean)
  const map = new Map()
  pairs.forEach((pair) => {
    const index = pair.indexOf('=')
    const key = decodeURIComponent(index === -1 ? pair : pair.slice(0, index))
    const value = index === -1 ? '' : pair.slice(index + 1)
    if (!map.has(key)) map.set(key, [])
    map.get(key).push(value)
  })
  return Array.from(map.keys())
    .sort()
    .map((key) => map.get(key).sort().map((value) => `${encodeURIComponent(key)}=${value}`).join('&'))
    .join('&')
}

/** 随机 nonce（32 位十六进制）。 */
export function randomNonce() {
  const bytes = []
  for (let index = 0; index < 16; index += 1) bytes.push(Math.floor(Math.random() * 256))
  return bytesToHex(bytes)
}

/** 是否需要对这次请求签名。 */
export function needsSignature(method) {
  return SIGNED_METHODS.includes(String(method || 'GET').toUpperCase())
}

/** 计算签名头；没有密钥时返回空对象。 */
export function signRequest(signKey, method, path, query, body) {
  if (!signKey || !signKey.key || !needsSignature(method)) return {}
  const ts = String(Math.floor(Date.now() / 1000))
  const nonce = randomNonce()
  const canonical = [
    String(method).toUpperCase(),
    path,
    sortedQuery(query),
    sha256Hex(body || ''),
    ts,
    nonce,
  ].join('\n')
  return {
    'X-DT-Ts': ts,
    'X-DT-Nonce': nonce,
    'X-DT-Key': signKey.id,
    'X-DT-Sign': hmacSha256Hex(signKey.key, canonical),
  }
}
