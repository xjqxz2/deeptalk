/**
 * 登录态：token 与用户信息持久化在本地存储。
 * 小程序 / APP 没有可靠的 Cookie，统一用 token + Authorization 头。
 */
import { reactive } from 'vue'

const TOKEN_KEY = 'dt_token'
const USER_KEY = 'dt_user'

export const session = reactive({
  token: uni.getStorageSync(TOKEN_KEY) || '',
  user: uni.getStorageSync(USER_KEY) || null,
})

/** 读取 token，供请求层与 WebSocket 使用。 */
export function getToken() {
  return session.token || uni.getStorageSync(TOKEN_KEY) || ''
}

/** 保存登录态。 */
export function setSession(token, user) {
  session.token = token || ''
  session.user = user || null
  if (token) uni.setStorageSync(TOKEN_KEY, token)
  if (user) uni.setStorageSync(USER_KEY, user)
}

/** 更新用户信息（额度变化等）。 */
const SIGN_KEY = 'dt_sign_key'

/** 读取会话签名密钥：{ id, key }。 */
export function getSignKey() {
  return uni.getStorageSync(SIGN_KEY) || null
}

/** 保存会话签名密钥。 */
export function setSignKey(value) {
  if (!value || !value.key || !value.id) {
    uni.removeStorageSync(SIGN_KEY)
    return
  }
  uni.setStorageSync(SIGN_KEY, { id: value.id, key: value.key })
}

export function patchUser(patch) {
  session.user = { ...(session.user || {}), ...patch }
  uni.setStorageSync(USER_KEY, session.user)
}

/** 清空登录态。 */
export function clearSession() {
  session.token = ''
  session.user = null
  uni.removeStorageSync(TOKEN_KEY)
  uni.removeStorageSync(USER_KEY)
  uni.removeStorageSync('dt_sign_key')
}

/** 是否已登录。 */
export function isLoggedIn() {
  return Boolean(getToken())
}
