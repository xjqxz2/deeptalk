/**
 * 请求签名（方案 B）：每个会话一把派生密钥，服务端只存 key id。
 *
 * 客户端对写请求计算
 *   HMAC-SHA256(key, METHOD\nPATH\nSORTED_QUERY\nSHA256(BODY)\nTS\nNONCE)
 * 并把 ts / nonce / key id / 签名放进 X-DT-* 请求头。
 *
 * 密钥保存在内存 + sessionStorage：刷新页面后由 /auth/sign-key 重新取回，
 * 关闭标签页即失效。
 */

/** 签名请求头。 */
export const SIGN_HEADERS = {
  ts: 'X-DT-Ts',
  nonce: 'X-DT-Nonce',
  key: 'X-DT-Key',
  sign: 'X-DT-Sign',
} as const

/** 需要签名的方法；GET / HEAD 不签。 */
const SIGNED_METHODS = new Set(['POST', 'PUT', 'PATCH', 'DELETE'])

const STORAGE_KEY = 'dt-sign-key'

export interface SignKey {
  id: string
  key: string
}

let cached: SignKey | null = null

/** 保存会话签名密钥。 */
export function setSignKey(key: SignKey | null): void {
  cached = key
  try {
    if (key) sessionStorage.setItem(STORAGE_KEY, JSON.stringify(key))
    else sessionStorage.removeItem(STORAGE_KEY)
  } catch {
    // 隐私模式下 sessionStorage 可能不可用，仅保留内存副本
  }
}

/** 读取签名密钥（内存优先，其次 sessionStorage）。 */
export function getSignKey(): SignKey | null {
  if (cached) return cached
  try {
    const raw = sessionStorage.getItem(STORAGE_KEY)
    if (!raw) return null
    cached = JSON.parse(raw) as SignKey
    return cached
  } catch {
    return null
  }
}

/** 是否需要对这次请求签名。 */
export function needsSignature(method: string): boolean {
  return SIGNED_METHODS.has(method.toUpperCase())
}

/** 十六进制字符串转字节。 */
function hexToBytes(hex: string): Uint8Array {
  const bytes = new Uint8Array(hex.length / 2)
  for (let index = 0; index < bytes.length; index += 1) {
    bytes[index] = Number.parseInt(hex.slice(index * 2, index * 2 + 2), 16)
  }
  return bytes
}

/** 字节转十六进制。 */
function bytesToHex(bytes: ArrayBufferLike): string {
  return Array.from(new Uint8Array(bytes as ArrayBuffer))
    .map((item) => item.toString(16).padStart(2, '0'))
    .join('')
}

/** SHA-256 十六进制摘要。 */
async function sha256Hex(text: string): Promise<string> {
  const digest = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(text))
  return bytesToHex(digest)
}

/** 生成 32 位随机 nonce。 */
function randomNonce(): string {
  const bytes = new Uint8Array(16)
  crypto.getRandomValues(bytes)
  return Array.from(bytes)
    .map((item) => item.toString(16).padStart(2, '0'))
    .join('')
}

/** 查询参数按名字排序后拼回，与服务端保持一致。 */
export function sortedQuery(query: string): string {
  if (!query) return ''
  const params = new URLSearchParams(query)
  const keys = Array.from(new Set(Array.from(params.keys()))).sort()
  return keys
    .map((key) => {
      const values = params.getAll(key).sort()
      return values.map((value) => `${encodeURIComponent(key)}=${encodeURIComponent(value)}`).join('&')
    })
    .join('&')
}

/** 为一次请求生成签名头；没有密钥时返回空对象。 */
export async function signRequest(
  method: string,
  path: string,
  query: string,
  body: string,
): Promise<Record<string, string>> {
  if (!needsSignature(method)) return {}
  const key = getSignKey()
  if (!key) return {}
  const ts = String(Math.floor(Date.now() / 1000))
  const nonce = randomNonce()
  const canonical = [
    method.toUpperCase(),
    path,
    sortedQuery(query),
    await sha256Hex(body),
    ts,
    nonce,
  ].join('\n')
  const cryptoKey = await crypto.subtle.importKey(
    'raw',
    hexToBytes(key.key) as unknown as BufferSource,
    { name: 'HMAC', hash: 'SHA-256' },
    false,
    ['sign'],
  )
  const signature = await crypto.subtle.sign(
    'HMAC',
    cryptoKey,
    new TextEncoder().encode(canonical) as unknown as BufferSource,
  )
  return {
    [SIGN_HEADERS.ts]: ts,
    [SIGN_HEADERS.nonce]: nonce,
    [SIGN_HEADERS.key]: key.id,
    [SIGN_HEADERS.sign]: bytesToHex(signature),
  }
}

/**
 * 供自行发起的 fetch 使用：根据完整路径生成签名头。
 * 流式接口（SSE）不经过 request 层，需要单独调用。
 */
export async function signatureHeadersFor(
  method: string,
  url: string,
  body: string,
): Promise<Record<string, string>> {
  const [path, query = ''] = url.split('?')
  return signRequest(method, path, query, body)
}
