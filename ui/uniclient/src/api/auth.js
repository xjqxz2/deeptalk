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

/** 小程序微信一键登录：用 wx.login 的 code 换会话（首次自动建号）。 */
export function wechatLogin(code) {
  return request({ url: '/api/v1/c/auth/wechat/login', method: 'POST', data: { code } })
}

/** 更新昵称与头像。 */
export function updateProfile(payload) {
  return request({ url: '/api/v1/c/auth/profile', method: 'PATCH', data: payload })
}

/** 补全邮箱与密码，之后可在 PWA / PC 端用邮箱登录。 */
export function completeProfile(email, password) {
  return request({ url: '/api/v1/c/auth/profile/complete', method: 'POST', data: { email, password } })
}
