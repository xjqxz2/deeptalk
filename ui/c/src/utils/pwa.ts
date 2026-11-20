/**
 * PWA 平台判断与 Service Worker 注册。
 * 安装引导是渐进增强：不支持 beforeinstallprompt 的浏览器仍可通过系统菜单安装。
 */

/** Chromium 的安装事件，目前不在标准 DOM 类型里。 */
export interface BeforeInstallPromptEvent extends Event {
  prompt: () => Promise<void>
  readonly userChoice: Promise<{ outcome: 'accepted' | 'dismissed'; platform: string }>
}

/** iOS Safari 的独立窗口标记。 */
interface NavigatorWithStandalone extends Navigator {
  standalone?: boolean
}

/** 是否已经从主屏幕启动。 */
export function isStandaloneMode(): boolean {
  if (typeof window === 'undefined') return false
  const displayMode = window.matchMedia('(display-mode: standalone)').matches
  return displayMode || (navigator as NavigatorWithStandalone).standalone === true
}

/** 是否 iPhone / iPad（含 iPadOS 桌面版 UA）。 */
export function isIosDevice(): boolean {
  if (typeof navigator === 'undefined') return false
  const iosUa = /iPad|iPhone|iPod/.test(navigator.userAgent)
  const iPadDesktopUa = navigator.platform === 'MacIntel' && navigator.maxTouchPoints > 1
  return iosUa || iPadDesktopUa
}

/** 是否 Android 设备。 */
export function isAndroidDevice(): boolean {
  return typeof navigator !== 'undefined' && /Android/i.test(navigator.userAgent)
}

/** iOS 上只有 Safari 提供“添加到主屏幕”，其他浏览器需提示换 Safari。 */
export function isIosSafari(): boolean {
  if (!isIosDevice()) return false
  return !/CriOS|FxiOS|EdgiOS|OPiOS|DuckDuckGo|GSA/i.test(navigator.userAgent)
}

/** 注册 C 端 Service Worker；失败只降级为普通网页，不阻塞应用启动。 */
export function registerServiceWorker(): void {
  if (!import.meta.env.PROD || !('serviceWorker' in navigator)) return

  const register = (): void => {
    const swUrl = `${import.meta.env.BASE_URL}sw.js`
    navigator.serviceWorker.register(swUrl, { scope: import.meta.env.BASE_URL }).catch((error: unknown) => {
      console.warn('[pwa] Service Worker 注册失败', error)
    })
  }

  if (document.readyState === 'complete') register()
  else window.addEventListener('load', register, { once: true })
}
