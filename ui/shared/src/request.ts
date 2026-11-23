/**
 * 统一请求层：所有接口都走这里，业务代码只拿到 data。
 * 约定：
 * - 使用 Cookie 会话（credentials: include），不在前端保存 token；
 * - 后端统一信封 { code, msg, data }，code !== 0 抛出 ApiError；
 * - 401/423 由页面或路由守卫统一处理，这里只抛出结构化错误。
 */
import { setSignKey, signRequest } from './sign'
import type { Envelope } from './types'

/** 接口地址前缀，开发期由 Vite 代理到 0.0.0.0:9821。 */
const API_BASE = '/api/v1'

/** 结构化接口错误，页面可直接展示 message。 */
export class ApiError extends Error {
  /** 业务错误码 */
  readonly code: number
  /** HTTP 状态码 */
  readonly status: number

  constructor(message: string, code: number, status: number) {
    super(message)
    this.name = 'ApiError'
    this.code = code
    this.status = status
  }
}

/** 请求选项，与 fetch 保持一致但省略 body 类型限制。 */
export interface RequestOptions {
  method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'
  body?: unknown
  query?: Record<string, string | number | boolean | undefined | null>
  signal?: AbortSignal
  /** 内部使用：签名失败后重试一次，避免无限递归 */
  retried?: boolean
}

/** 拼接查询参数，undefined / null 会被忽略。 */
function withQuery(path: string, query?: RequestOptions['query']): string {
  if (!query) return path
  const params = new URLSearchParams()
  Object.entries(query).forEach(([key, value]) => {
    if (value === undefined || value === null || value === '') return
    params.set(key, String(value))
  })
  const qs = params.toString()
  return qs ? `${path}${path.includes('?') ? '&' : '?'}${qs}` : path
}

/** 重新拉取一次会话签名密钥；失败返回 false。 */
async function refreshSignKey(): Promise<boolean> {
  try {
    const response = await fetch(`${API_BASE}/c/auth/sign-key`, { credentials: 'include' })
    if (!response.ok) return false
    const payload = (await response.json()) as {
      data?: { sign_key?: string; sign_key_id?: string }
    }
    const key = payload.data?.sign_key
    const id = payload.data?.sign_key_id
    if (!key || !id) return false
    setSignKey({ id, key })
    return true
  } catch {
    return false
  }
}

/**
 * 发起 JSON 请求并解包信封。
 * 网络异常与后端错误都会抛出 ApiError，调用方无需再判断 res.ok。
 */
export async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { method = 'GET', body, query, signal } = options
  const retried = Boolean((options as RequestOptions & { retried?: boolean }).retried)
  const url = withQuery(`${API_BASE}${path}`, query)
  const bodyText = body === undefined ? '' : JSON.stringify(body)
  // 写请求需要签名：method + path + query + body 摘要 + 时间戳 + nonce
  const queryText = url.includes('?') ? url.slice(url.indexOf('?') + 1) : ''
  const pathText = url.split('?')[0]
  const signHeaders = await signRequest(method, pathText, queryText, bodyText)
  let response: Response
  try {
    response = await fetch(url, {
      method,
      credentials: 'include',
      headers: {
        ...(body === undefined ? {} : { 'Content-Type': 'application/json' }),
        ...signHeaders,
      },
      body: body === undefined ? undefined : bodyText,
      signal,
    })
  } catch (error) {
    if (error instanceof DOMException && error.name === 'AbortError') throw error
    throw new ApiError('网络异常，请确认服务已启动', 2001, 0)
  }
  const contentType = response.headers.get('content-type') ?? ''
  if (!contentType.includes('application/json')) {
    throw new ApiError('服务返回了非 JSON 响应', 2001, response.status)
  }
  const envelope = (await response.json()) as Envelope<T>
  if (!response.ok || envelope.code !== 0) {
    const message = envelope.msg || '请求失败，请稍后重试'
    // 签名密钥可能已经过期（服务端会话轮换、页面长时间挂起）：
    // 取一次新密钥后自动重试，用户不必手动刷新。
    if (message.includes('签名') && !retried) {
      const refreshed = await refreshSignKey()
      if (refreshed) {
        return request<T>(path, { ...options, retried: true } as RequestOptions)
      }
    }
    throw new ApiError(message, envelope.code, response.status)
  }
  return envelope.data
}

/** GET 快捷方法。 */
export function get<T>(path: string, query?: RequestOptions['query'], signal?: AbortSignal): Promise<T> {
  return request<T>(path, { query, signal })
}

/** POST 快捷方法。 */
export function post<T>(path: string, body?: unknown, signal?: AbortSignal): Promise<T> {
  return request<T>(path, { method: 'POST', body, signal })
}

/** PUT 快捷方法。 */
export function put<T>(path: string, body?: unknown): Promise<T> {
  return request<T>(path, { method: 'PUT', body })
}

/** PATCH 快捷方法。 */
export function patch<T>(path: string, body?: unknown): Promise<T> {
  return request<T>(path, { method: 'PATCH', body })
}

/** DELETE 快捷方法。 */
export function del<T>(path: string): Promise<T> {
  return request<T>(path, { method: 'DELETE' })
}

/**
 * 下载文件（CSV 等）。
 * 说明：文件名优先取 Content-Disposition，缺失时使用调用方提供的默认名。
 */
export async function download(path: string, fallbackName: string): Promise<void> {
  const response = await fetch(`${API_BASE}${path}`, { credentials: 'include' })
  if (!response.ok) {
    let message = '导出失败，请稍后重试'
    try {
      const envelope = (await response.json()) as Envelope<null>
      message = envelope.msg || message
    } catch {
      // 非 JSON 错误体时保留默认提示
    }
    throw new ApiError(message, 2001, response.status)
  }
  const disposition = response.headers.get('content-disposition') ?? ''
  const match = /filename="?([^";]+)"?/.exec(disposition)
  const filename = match?.[1] ?? fallbackName
  const blob = await response.blob()
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = filename
  anchor.click()
  URL.revokeObjectURL(url)
}

/**
 * 上传 multipart 表单并解包信封。
 * 不手动设置 Content-Type，交给浏览器生成 boundary。
 */
export async function upload<T>(path: string, form: FormData, signal?: AbortSignal): Promise<T> {
  let response: Response
  try {
    response = await fetch(`${API_BASE}${path}`, {
      method: 'POST',
      credentials: 'include',
      body: form,
      signal,
    })
  } catch (error) {
    if (error instanceof DOMException && error.name === 'AbortError') throw error
    throw new ApiError('网络异常，请确认服务已启动', 2001, 0)
  }
  const contentType = response.headers.get('content-type') ?? ''
  if (!contentType.includes('application/json')) {
    throw new ApiError('服务返回了非 JSON 响应', 2001, response.status)
  }
  const envelope = (await response.json()) as Envelope<T>
  if (!response.ok || envelope.code !== 0) {
    throw new ApiError(envelope.msg || '上传失败，请稍后重试', envelope.code, response.status)
  }
  return envelope.data
}
