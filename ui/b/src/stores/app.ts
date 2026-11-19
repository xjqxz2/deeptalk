/** B 端外壳状态：移动端抽屉与全局搜索框。 */
import { ref } from 'vue'
import { defineStore } from 'pinia'

export const useAppStore = defineStore('app', () => {
  /** 移动端侧栏是否打开。 */
  const mobileSidebarOpen = ref(false)
  /** 当前是否小于 1024。 */
  const isMobile = ref(false)
  /** 管理员指南弹窗。 */
  const guideOpen = ref(false)

  /** 同步断点。 */
  function syncViewport(width: number): void {
    isMobile.value = width <= 1024
    if (width >= 1024) mobileSidebarOpen.value = false
  }

  /** 打开 / 关闭移动端侧栏。 */
  function toggleSidebar(): void {
    mobileSidebarOpen.value = !mobileSidebarOpen.value
  }

  /** 关闭侧栏。 */
  function closeSidebar(): void {
    mobileSidebarOpen.value = false
  }

  return { mobileSidebarOpen, isMobile, guideOpen, syncViewport, toggleSidebar, closeSidebar }
})
