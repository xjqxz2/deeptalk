/**
 * 应用外壳状态：侧栏折叠与移动端抽屉。
 * 断点由布局监听，页面只消费状态，不各自监听 resize。
 */
import { ref } from 'vue'
import { defineStore } from 'pinia'

export const useAppStore = defineStore('app', () => {
  /** 桌面端侧栏是否折叠为 64px。 */
  const sidebarCollapsed = ref(false)
  /** 移动端侧栏抽屉是否打开。 */
  const mobileSidebarOpen = ref(false)
  /** 当前视口是否小于 768。 */
  const isMobile = ref(false)
  /** 当前视口是否小于 1024。 */
  const isNarrow = ref(false)

  /** 按窗口宽度同步断点；小于 1024 自动折叠，小于 768 使用抽屉。 */
  function syncViewport(width: number): void {
    // 移动端阈值 768（抽屉），窄屏阈值 1024（侧栏折叠为 64px 图标条）
    isMobile.value = width < 768
    isNarrow.value = width <= 1024
    if (width <= 1024) sidebarCollapsed.value = true
    if (width > 1024) sidebarCollapsed.value = false
  }

  /** 手动切换侧栏。 */
  function toggleSidebar(): void {
    if (isMobile.value) {
      mobileSidebarOpen.value = !mobileSidebarOpen.value
      return
    }
    sidebarCollapsed.value = !sidebarCollapsed.value
  }

  /** 关闭移动端抽屉。 */
  function closeMobileSidebar(): void {
    mobileSidebarOpen.value = false
  }

  return { sidebarCollapsed, mobileSidebarOpen, isMobile, isNarrow, syncViewport, toggleSidebar, closeMobileSidebar }
})
