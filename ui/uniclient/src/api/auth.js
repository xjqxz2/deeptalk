/** 账号相关接口。 */
import { request } from './http'
import { setSignKey } from '@/store/session'

/** 把服务端下发的签名密钥存到本地，供后续写请求使用。 */
export function applySignKey(data) {
  if (data && data.sign_key && data.sign_key_id) {
    setSignKey({ id: data.sign_key_id, key: data.sign_key })
  }
}

/** 登录，返回 { user, token } 与本次会话的签名密钥。 */
export async function login(email, password) {
  const data = await request({ url: '/api/v1/c/auth/login', method: 'POST', data: { email, password }, silent: true })
  applySignKey(data)
  return data
}

/** 注册，返回 { user, token } 与签名密钥。 */
export async function register(email, password) {
  const data = await request({ url: '/api/v1/c/auth/register', method: 'POST', data: { email, password }, silent: true })
  applySignKey(data)
  return data
}

/** 重新取回当前会话的签名密钥（冷启动、换设备后用）。 */
export async function fetchSignKey() {
  const data = await request({ url: '/api/v1/c/auth/sign-key', silent: true })
  applySignKey({ sign_key: data.sign_key, sign_key_id: data.sign_key_id })
  return data
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
export async function wechatLogin(code) {
  const data = await request({ url: '/api/v1/c/auth/wechat/login', method: 'POST', data: { code } })
  applySignKey(data)
  return data
}

/** 更新昵称与头像。 */
export function updateProfile(payload) {
  return request({ url: '/api/v1/c/auth/profile', method: 'PATCH', data: payload })
}

/** 补全邮箱与密码，之后可在 PWA / PC 端用邮箱登录。 */
export function completeProfile(email, password) {
  return request({ url: '/api/v1/c/auth/profile/complete', method: 'POST', data: { email, password } })
}
