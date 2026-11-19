/**
 * B 端路由实例与管理员守卫。
 * 非管理员账号登录后停在登录页，并提示“仅平台管理员可访问”。
 */
import { createRouter, createWebHistory } from 'vue-router'
import { appRoutes } from '@config/routes'
import { useUserStore } from '@/stores/user'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: appRoutes,
  scrollBehavior: () => ({ top: 0 }),
})

router.beforeEach(async (to) => {
  const users = useUserStore()
  if (!users.initialized) {
    try {
      await users.bootstrap()
    } catch {
      // 网络异常按未登录处理
    }
  }
  if (to.meta.public) {
    if (users.isAdmin && to.name === 'login') {
      return { name: 'overview' }
    }
    return true
  }
  if (!users.isLoggedIn) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }
  if (!users.isAdmin) {
    return { name: 'login', query: { forbidden: '1' } }
  }
  return true
})

router.afterEach((to) => {
  document.title = to.meta.title ? `${to.meta.title} · DeepTalk Admin` : 'DeepTalk Admin'
})

export default router
