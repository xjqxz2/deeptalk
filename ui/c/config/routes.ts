/**
 * C 端路由表：路由、标题与页面级权限的唯一数据源。
 * 说明：会话页与空态共用 ChatLayout，保证切换会话时侧栏不重建。
 */
import type { RouteRecordRaw } from 'vue-router'

/** 应用路由。 */
export const appRoutes: RouteRecordRaw[] = [
  {
    path: '/',
    component: () => import('@/layouts/ChatLayout.vue'),
    children: [
      {
        path: '',
        name: 'chat-empty',
        component: () => import('@/pages/chat/ChatEmptyPage.vue'),
        meta: { title: '新对话', requiresAuth: true },
      },
      {
        path: 'chat/:id',
        name: 'chat-session',
        component: () => import('@/pages/chat/ChatSessionPage.vue'),
        meta: { title: '对话', requiresAuth: true },
      },
    ],
  },
  {
    path: '/login',
    name: 'login',
    component: () => import('@/pages/auth/LoginPage.vue'),
    meta: { title: '登录', public: true, layout: 'auth' },
  },
  {
    path: '/register',
    name: 'register',
    component: () => import('@/pages/auth/RegisterPage.vue'),
    meta: { title: '注册', public: true, layout: 'auth' },
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'not-found',
    component: () => import('@/pages/NotFoundPage.vue'),
    meta: { title: '页面不存在', public: true },
  },
]
