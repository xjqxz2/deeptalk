/**
 * C 端路由实例与登录守卫。
 * 说明：会话探测只在首次导航时执行一次，避免每个页面都请求 /me。
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
      // 网络异常时按未登录处理，登录页会给出可读提示
    }
  }
  if (to.meta.public) {
    // 已登录用户访问登录 / 注册页时直接回工作台
    if (users.isLoggedIn && (to.name === 'login' || to.name === 'register')) {
      return { name: 'chat-empty' }
    }
    return true
  }
  if (!users.isLoggedIn) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }
  return true
})

router.afterEach((to) => {
  const title = to.meta.title ? `${to.meta.title} · 小橘树洞` : '小橘树洞 · AI 对话工作台'
  document.title = title
})

export default router
