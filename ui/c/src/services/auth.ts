/** C 端账号接口：注册、登录、会话与用量。 */
import { get, post, put } from '@deeptalk/shared'
import type { AuthConfig, UsageInfo, User } from '@deeptalk/shared'

/** 注册开关等公开配置，不需要登录。 */
export function fetchAuthConfig(): Promise<AuthConfig> {
  return get<AuthConfig>('/c/auth/config')
}

/** 登录，成功后服务端下发 HttpOnly Cookie。 */
export interface AuthResult {
  user: User
  token?: string
  sign_key?: string
  sign_key_id?: string
}

export function login(email: string, password: string): Promise<AuthResult> {
  return post<AuthResult>('/c/auth/login', { email, password })
}

/** 刷新页面后重新取回会话签名密钥（服务端字段为 sign_key / sign_key_id）。 */
export function fetchSignKey(): Promise<{ sign_key?: string; sign_key_id?: string }> {
  return get<{ sign_key?: string; sign_key_id?: string }>('/c/auth/sign-key')
}

/** 注册并直接登录。 */
export function register(email: string, password: string): Promise<AuthResult> {
  return post<AuthResult>('/c/auth/register', { email, password })
}

/** 退出登录。 */
export function logout(): Promise<null> {
  return post<null>('/c/auth/logout')
}

/** 当前登录用户。 */
export function fetchMe(): Promise<{ user: User }> {
  return get<{ user: User }>('/c/auth/me')
}

/** 修改密码；成功后全部会话失效。 */
export function changePassword(oldPassword: string, newPassword: string): Promise<null> {
  return put<null>('/c/auth/password', { old_password: oldPassword, new_password: newPassword })
}

/** 本月用量。 */
export function fetchUsage(): Promise<UsageInfo> {
  return get<UsageInfo>('/c/usage')
}
