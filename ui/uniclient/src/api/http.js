/**
 * 统一的接口请求层：带 token、统一错误提示、登录失效自动跳转。
 */
import { BASE_URL, CLIENT_PLATFORM } from '@/config/env'
import { getToken, clearSession } from '@/store/session'
import { goLogin } from '@/utils/nav'

// 并发请求同时 401 时只跳一次登录页，避免连续 reLaunch 把页面来回重建
let redirecting = false

/** 当前是否已经在登录页。 */
function onLoginPage() {
  const pages = typeof getCurrentPages === 'function' ? getCurrentPages() : []
  const top = pages[pages.length - 1]
  return Boolean(top && String(top.route || '').includes('login'))
}

/** 跳转登录页（单次）。 */
function gotoLogin() {
  if (redirecting || onLoginPage()) return
  redirecting = true
  clearSession()
  goLogin()
}

/**
 * 发起请求。
 * @param {object} options
 * @param {string} options.url 以 /api/v1 开头的路径
 * @param {string} [options.method] 请求方法
 * @param {object} [options.data] 请求体
 * @param {boolean} [options.silent] 失败时不弹提示
 */
export function request({ url, method = 'GET', data, silent = false }) {
  return new Promise((resolve, reject) => {
    const token = getToken()
    const header = { 'Content-Type': 'application/json', 'X-Client': CLIENT_PLATFORM }
    if (token) header.Authorization = `Bearer ${token}`
    uni.request({
      url: `${BASE_URL}${url}`,
      method,
      data,
      header,
      timeout: 30000,
      success: (res) => {
        const body = res.data || {}
        if (res.statusCode === 401 || body.code === 1002) {
          gotoLogin()
          reject(new Error(body.msg || '登录已失效，请重新登录'))
          return
        }
        if (res.statusCode >= 200 && res.statusCode < 300 && body.code === 0) {
          resolve(body.data)
          return
        }
        const message = body.msg || `请求失败（${res.statusCode}）`
        if (!silent) uni.showToast({ title: message, icon: 'none' })
        reject(new Error(message))
      },
      fail: (err) => {
        const message = err.errMsg?.includes('timeout') ? '请求超时，请重试' : '网络异常，请检查连接'
        if (!silent) uni.showToast({ title: message, icon: 'none' })
        reject(new Error(message))
      },
    })
  })
}

/** 上传图片附件，返回附件信息。 */
export function uploadFile(filePath) {
  return new Promise((resolve, reject) => {
    const token = getToken()
    uni.uploadFile({
      url: `${BASE_URL}/api/v1/c/attachments`,
      filePath,
      name: 'file',
      header: { 'X-Client': CLIENT_PLATFORM, ...(token ? { Authorization: `Bearer ${token}` } : {}) },
      success: (res) => {
        try {
          const body = JSON.parse(res.data)
          if (body.code === 0) {
            resolve(body.data.attachment)
            return
          }
          reject(new Error(body.msg || '上传失败'))
        } catch {
          reject(new Error('上传失败'))
        }
      },
      fail: () => reject(new Error('上传失败，请重试')),
    })
  })
}
