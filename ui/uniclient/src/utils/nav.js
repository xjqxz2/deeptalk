/**
 * 统一的页面跳转。
 *
 * 微信小程序里，如果在 onLaunch / 页面还没挂载完时就 reLaunch，或者短时间内连发两次
 * reLaunch，会报 "routeDone with a webviewId N is not found"。这里做两件事：
 * 1. 去重：同一时刻只允许一次跳转，且目标页已经是当前页就不再跳
 * 2. 延迟一帧：等首次路由完成再跳，避免和启动时的默认页面切换打架
 */
let pending = false

/** 当前页面路径（不含参数）。 */
function currentRoute() {
  if (typeof getCurrentPages !== 'function') return ''
  const pages = getCurrentPages()
  const top = pages[pages.length - 1]
  return String(top?.route || '')
}

/** 跳转到指定页面（reLaunch），带去重与延迟。 */
export function relaunch(url) {
  const target = url.replace(/^\//, '').replace(/\?.*$/, '')
  if (pending) return
  if (currentRoute() === target) return
  pending = true
  // 等首屏路由与上一次跳转动画结束（微信里 0ms 仍可能抢跑）
  setTimeout(() => {
    uni.reLaunch({
      url,
      fail: () => {
        // 路由还没准备好时再兜底重试一次
        setTimeout(() => {
          uni.reLaunch({ url, fail: () => {} })
        }, 60)
      },
      complete: () => {
        setTimeout(() => {
          pending = false
        }, 200)
      },
    })
  }, 50)
}

/** 去登录页。 */
export function goLogin() {
  relaunch('/pages/login/login')
}

/** 去聊天页。 */
export function goChat() {
  relaunch('/pages/chat/chat')
}
