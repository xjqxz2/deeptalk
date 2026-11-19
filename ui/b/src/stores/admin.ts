/**
 * 后台外壳数据：侧栏存储用量、概览计数。
 * 页面可以自行刷新，但外壳数据只在这里维护，避免每个页面各取一份。
 */
import { ref } from 'vue'
import { defineStore } from 'pinia'
import type { OverviewData, SettingsInfo } from '@deeptalk/shared'
import { fetchOverview, fetchSettings } from '@/services/admin'

export const useAdminStore = defineStore('admin', () => {
  const overview = ref<OverviewData | null>(null)
  const settings = ref<SettingsInfo | null>(null)
  const loading = ref(false)

  /** 加载侧栏所需数据；失败时保留旧值。 */
  async function loadShell(): Promise<void> {
    loading.value = true
    try {
      const [overviewData, settingsData] = await Promise.all([fetchOverview(), fetchSettings()])
      overview.value = overviewData
      settings.value = settingsData
    } catch {
      // 外壳数据失败不阻断页面，由页面自身的错误态兜底
    } finally {
      loading.value = false
    }
  }

  /** 只刷新概览，用于厂商 / 用户变更后同步侧栏与概览数字。 */
  async function refreshOverview(): Promise<void> {
    try {
      overview.value = await fetchOverview()
    } catch {
      // 静默失败，页面已有各自提示
    }
  }

  return { overview, settings, loading, loadShell, refreshOverview }
})
