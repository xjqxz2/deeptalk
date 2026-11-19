/**
 * 登录用户 Store：保存当前账号与本月用量。
 * 只在这里处理用户态；页面不得直接读 Cookie 或 localStorage 中的会话信息。
 */
import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { ApiError } from '@deeptalk/shared'
import type { UsageInfo, User } from '@deeptalk/shared'
import * as authApi from '@/services/auth'

export const useUserStore = defineStore('user', () => {
  /** 当前登录用户；未登录为 null。 */
  const user = ref<User | null>(null)
  /** 本月用量，登录后异步刷新。 */
  const usage = ref<UsageInfo | null>(null)
  /** 是否已完成首次会话探测，避免每次路由跳转都请求 /me。 */
  const initialized = ref(false)
  /** 登录 / 注册提交中。 */
  const submitting = ref(false)

  const isLoggedIn = computed(() => user.value !== null)
  const isAdmin = computed(() => user.value?.role === 'admin')

  /** 首次进入应用时探测登录态；401 视为未登录，其余错误向上抛给页面。 */
  async function bootstrap(): Promise<void> {
    if (initialized.value) return
    try {
      const data = await authApi.fetchMe()
      user.value = data.user
      await refreshUsage()
    } catch (error) {
      if (error instanceof ApiError && (error.code === 1002 || error.status === 401)) {
        user.value = null
      } else {
        throw error
      }
    } finally {
      initialized.value = true
    }
  }

  /** 登录并刷新用量。 */
  async function login(email: string, password: string): Promise<User> {
    submitting.value = true
    try {
      const data = await authApi.login(email, password)
      user.value = data.user
      initialized.value = true
      await refreshUsage()
      return data.user
    } finally {
      submitting.value = false
    }
  }

  /** 注册并直接进入工作台。 */
  async function register(email: string, password: string): Promise<User> {
    submitting.value = true
    try {
      const data = await authApi.register(email, password)
      user.value = data.user
      initialized.value = true
      await refreshUsage()
      return data.user
    } finally {
      submitting.value = false
    }
  }

  /** 退出登录并清空本地用户态。 */
  async function logout(): Promise<void> {
    try {
      await authApi.logout()
    } finally {
      user.value = null
      usage.value = null
    }
  }

  /** 刷新用量；失败时保留旧值，不打断对话。 */
  async function refreshUsage(): Promise<void> {
    if (!user.value) return
    try {
      usage.value = await authApi.fetchUsage()
      // 同步账号上的额度，避免侧栏与账号卡片口径不一致
      user.value.used_tokens = usage.value.used_tokens
      user.value.quota_tokens = usage.value.quota_tokens
      user.value.used_percent = usage.value.used_percent
    } catch {
      // 用量是辅助信息，失败时静默保留旧值
    }
  }

  return {
    user,
    usage,
    initialized,
    submitting,
    isLoggedIn,
    isAdmin,
    bootstrap,
    login,
    register,
    logout,
    refreshUsage,
  }
})
