/**
 * 安装到桌面的跨组件状态。
 * 模块级单例：侧栏、顶栏与安装说明弹窗共享同一个 beforeinstallprompt 事件，
 * 事件只能消费一次，不能在多个组件里各自监听。
 */
import { computed, ref } from 'vue'
import {
  isAndroidDevice,
  isIosDevice,
  isIosSafari,
  isStandaloneMode,
  type BeforeInstallPromptEvent,
} from '@/utils/pwa'

const deferredPrompt = ref<BeforeInstallPromptEvent | null>(null)
const standalone = ref(false)
const installed = ref(false)
const guideOpen = ref(false)
let initialized = false

/** 首次使用时注册浏览器事件，之后所有组件复用同一份状态。 */
function initialize(): void {
  if (initialized || typeof window === 'undefined') return
  initialized = true
  standalone.value = isStandaloneMode()

  window.addEventListener('beforeinstallprompt', (event) => {
    // 阻止 Chrome 的默认迷你提示，改由应用内入口触发
    event.preventDefault()
    deferredPrompt.value = event as BeforeInstallPromptEvent
  })

  window.addEventListener('appinstalled', () => {
    installed.value = true
    deferredPrompt.value = null
    standalone.value = true
    guideOpen.value = false
  })

  const displayMode = window.matchMedia('(display-mode: standalone)')
  const syncDisplayMode = (): void => {
    standalone.value = isStandaloneMode()
  }
  if (typeof displayMode.addEventListener === 'function') {
    displayMode.addEventListener('change', syncDisplayMode)
  } else {
    // 兼容旧版 iOS Safari
    displayMode.addListener(syncDisplayMode)
  }
}

export function usePwaInstall() {
  initialize()

  const ios = computed(isIosDevice)
  const android = computed(isAndroidDevice)
  const iosSafari = computed(isIosSafari)
  const available = computed(
    () => !standalone.value && !installed.value && (deferredPrompt.value !== null || ios.value || android.value),
  )

  /**
   * 触发安装：Android/桌面 Chrome 走系统安装弹窗，iOS 或无安装事件时展示步骤说明。
   * 返回 prompted 表示已调用系统弹窗，guide 表示打开了说明。
   */
  async function install(): Promise<'prompted' | 'guide' | 'unavailable'> {
    initialize()
    if (standalone.value || installed.value) return 'unavailable'

    if (deferredPrompt.value) {
      const prompt = deferredPrompt.value
      deferredPrompt.value = null
      try {
        await prompt.prompt()
        const choice = await prompt.userChoice
        if (choice.outcome === 'accepted') {
          installed.value = true
          return 'prompted'
        }
      } catch {
        // 浏览器拒绝弹窗时退回到说明，用户仍可从系统菜单安装
      }
    }

    guideOpen.value = true
    return 'guide'
  }

  return { available, standalone, ios, android, iosSafari, guideOpen, install }
}
