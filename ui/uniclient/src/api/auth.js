/** 账号相关接口。 */
import { request } from './http'

/** 登录，返回 { user, token }。 */
export function login(email, password) {
  return request({ url: '/api/v1/c/auth/login', method: 'POST', data: { email, password }, silent: true })
}

/** 注册，返回 { user, token }。 */
export function register(email, password) {
  return request({ url: '/api/v1/c/auth/register', method: 'POST', data: { email, password }, silent: true })
}

/** 当前登录用户。 */
export function fetchMe() {
  return request({ url: '/api/v1/c/auth/me', silent: true })
}

/** 退出登录。 */
export function logout() {
  return request({ url: '/api/v1/c/auth/logout', method: 'POST', silent: true })
}

/** 注册开关等公开配置。 */
export function fetchConfig() {
  return request({ url: '/api/v1/c/auth/config', silent: true })
}

/** 用量（底部用量卡片）。 */
export function fetchUsage() {
  return request({ url: '/api/v1/c/usage', silent: true })
}

/** 小程序端绑定 openid：内容安全检测按用户维度调用微信接口。 */
export function bindWeChat(code) {
  return request({ url: '/api/v1/c/auth/wechat/bind', method: 'POST', data: { code }, silent: true })
}
