/** 管理员账号接口：与 C 端共用账号体系，仅校验管理员角色。 */
import { get, post } from '@deeptalk/shared'
import type { User } from '@deeptalk/shared'

/** 登录并下发 HttpOnly Cookie，同时返回本次会话的签名密钥。 */
export interface AuthResult {
  user: User
  token?: string
  sign_key?: string
  sign_key_id?: string
}

export function login(email: string, password: string): Promise<AuthResult> {
  return post<AuthResult>('/c/auth/login', { email, password })
}

/** 重新取回当前会话的签名密钥。 */
export function fetchSignKey(): Promise<{ sign_key?: string; sign_key_id?: string }> {
  return get<{ sign_key?: string; sign_key_id?: string }>('/c/auth/sign-key')
}

/** 当前登录用户。 */
export function fetchMe(): Promise<{ user: User }> {
  return get<{ user: User }>('/c/auth/me')
}

/** 退出登录。 */
export function logout(): Promise<null> {
  return post<null>('/c/auth/logout')
}
