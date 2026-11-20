/**
 * 运行环境配置。
 * H5 调试时走 vite 代理，避免跨域；小程序 / APP 必须填生产域名。
 */
const DEV = import.meta.env.DEV

/** 接口地址：H5 开发态同源代理，其它端直连生产域名 */
export const BASE_URL = import.meta.env.VITE_API_BASE || (DEV ? '' : 'https://deeptalk.peon.top')

/** WebSocket 地址：小程序不支持自定义请求头之外的能力，token 走 query */
export function wsURL(token) {
  const base = BASE_URL || window.location.origin
  const url = `${base.replace(/^http/, 'ws')}/api/v1/c/ws?token=${encodeURIComponent(token)}`
  return url
}

/** 分享链接所在站点（PWA），用于复制分享地址 */
export const WEB_BASE = 'https://deeptalk.peon.top'
