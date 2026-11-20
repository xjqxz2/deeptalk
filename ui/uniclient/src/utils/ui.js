/**
 * 跨端 UI 封装。
 * H5 调试与小程序的可视能力差很多：H5 端使用 Vant 的 Toast / Dialog 保持与 Web 端一致的观感，
 * 小程序与 nvue 端退回 uni 原生组件（没有 DOM，无法运行 Vant）。
 */
// #ifdef H5
import { showToast, showConfirmDialog } from 'vant'
// #endif

/** 轻提示。 */
export function toast(message, duration = 1800) {
  if (!message) return
  // #ifdef H5
  showToast({ message, duration, position: 'bottom' })
  // #endif
  // #ifndef H5
  uni.showToast({ title: message, icon: 'none', duration })
  // #endif
}

/** 二次确认，返回是否确认。 */
export function confirm(title, message) {
  // #ifdef H5
  return showConfirmDialog({ title, message })
    .then(() => true)
    .catch(() => false)
  // #endif
  // #ifndef H5
  return new Promise((resolve) => {
    uni.showModal({
      title,
      content: message,
      success: (res) => resolve(Boolean(res.confirm)),
      fail: () => resolve(false),
    })
  })
  // #endif
}
