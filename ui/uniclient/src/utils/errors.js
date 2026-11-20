/**
 * 全局错误记录：H5 端把未捕获异常收集起来，页面顶部给一条可关闭的提示，
 * 这样出问题时有具体错误文案，而不是只看到一张白屏或浏览器崩溃页。
 */
import { reactive } from 'vue'

export const errorState = reactive({
  message: '',
  detail: '',
})

/** 记录一条错误（重复错误只保留最新一条）。 */
export function reportError(message, detail = '') {
  // 只展示人能看懂的错误；uni 的 {errMsg:'xxx:fail'} 之类噪声直接忽略
  const text = typeof message === 'string' ? message.trim() : ''
  if (!text || text === '[object Object]' || /:fail$/.test(text)) return
  errorState.message = text.slice(0, 200)
  errorState.detail = String(detail || '').slice(0, 300)
}

/** 关闭提示。 */
export function clearError() {
  errorState.message = ''
  errorState.detail = ''
}

// #ifdef H5
if (typeof window !== 'undefined') {
  window.addEventListener('error', (event) => {
    if (event?.message) reportError(event.message)
  })
  window.addEventListener('unhandledrejection', (event) => {
    const reason = event?.reason
    reportError(reason?.message || reason || '未处理的异步错误')
  })
}
// #endif
