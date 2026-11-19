/** 管理员登录态：B 端只允许平台管理员访问。 */
import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { ApiError } from '@deeptalk/shared'
import type { User } from '@deeptalk/shared'
import * as authApi from '@/services/auth'

export const useUserStore = defineStore('user', () => {
  const user = ref<User | null>(null)
  const initialized = ref(false)
  const submitting = ref(false)

  const isLoggedIn = computed(() => user.value !== null)
  const isAdmin = computed(() => user.value?.role === 'admin')

  /** 首次进入探测登录态。 */
  async function bootstrap(): Promise<void> {
    if (initialized.value) return
    try {
      const data = await authApi.fetchMe()
      user.value = data.user
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

  /** 登录；角色校验交给调用方，便于展示“仅管理员”提示。 */
  async function login(email: string, password: string): Promise<User> {
    submitting.value = true
    try {
      const data = await authApi.login(email, password)
      user.value = data.user
      initialized.value = true
      return data.user
    } finally {
      submitting.value = false
    }
  }

  /** 退出登录。 */
  async function logout(): Promise<void> {
    try {
      await authApi.logout()
    } finally {
      user.value = null
    }
  }

  return { user, initialized, submitting, isLoggedIn, isAdmin, bootstrap, login, logout }
})
