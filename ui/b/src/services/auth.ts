/** 管理员账号接口：与 C 端共用账号体系，仅校验管理员角色。 */
import { get, post } from '@deeptalk/shared'
import type { User } from '@deeptalk/shared'

/** 登录并下发 HttpOnly Cookie。 */
export function login(email: string, password: string): Promise<{ user: User }> {
  return post<{ user: User }>('/c/auth/login', { email, password })
}

/** 当前登录用户。 */
export function fetchMe(): Promise<{ user: User }> {
  return get<{ user: User }>('/c/auth/me')
}

/** 退出登录。 */
export function logout(): Promise<null> {
  return post<null>('/c/auth/logout')
}
