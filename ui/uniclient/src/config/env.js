/**
 * 运行环境配置。
 * H5 调试时走 vite 代理，避免跨域；小程序 / APP 必须填生产域名。
 */
const DEV = import.meta.env.DEV

/**
 * 接口地址：
 * - 显式设置 VITE_API_BASE 时以它为准（构建 H5 预览时设为空串，走同源代理）
 * - H5 开发态同源代理
 * - 小程序 / APP 直连生产域名
 */
const ENV_BASE = import.meta.env.VITE_API_BASE
export const BASE_URL = ENV_BASE !== undefined ? ENV_BASE : DEV ? '' : 'https://deeptalk.peon.top'

/** WebSocket 地址：小程序不支持自定义请求头之外的能力，token 走 query */
export function wsURL(token) {
  // 小程序 / APP 没有 window，这里必须兜底，否则取 origin 会直接抛错
  const base = BASE_URL || (typeof window !== 'undefined' ? window.location.origin : '')
  const url = `${base.replace(/^http/, 'ws')}/api/v1/c/ws?token=${encodeURIComponent(token)}`
  return url
}

/** 分享链接所在站点（PWA），用于复制分享地址 */
export const WEB_BASE = 'https://deeptalk.peon.top'

/**
 * 客户端平台标识：后端据此决定是否对小程序的输入做内容安全检测。
 * H5 调试页按 web 处理，微信小程序按 mp-weixin 处理。
 */
export const CLIENT_PLATFORM = (() => {
  // #ifdef MP-WEIXIN
  return 'mp-weixin'
  // #endif
  // #ifndef MP-WEIXIN
  return 'web'
  // #endif
})()
