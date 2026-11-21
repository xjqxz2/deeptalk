/**
 * B 端路由表：侧栏菜单、面包屑与页面标题的唯一数据源。
 * 说明：meta.menu=true 的路由会自动出现在左侧控制台菜单中。
 */
import type { Component } from 'vue'
import type { RouteRecordRaw } from 'vue-router'
import {
  ApiOutlined,
  AppstoreOutlined,
  BarChartOutlined,
  SettingOutlined,
  TeamOutlined,
} from '@ant-design/icons-vue'

/** 菜单图标与标题的类型扩展。 */
export interface AdminRouteMeta {
  title?: string
  public?: boolean
  menu?: boolean
  icon?: Component
}

export const appRoutes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'login',
    component: () => import('@/pages/auth/LoginPage.vue'),
    meta: { title: '管理员登录', public: true },
  },
  {
    path: '/',
    component: () => import('@/layouts/BasicLayout.vue'),
    redirect: '/overview',
    children: [
      {
        path: 'overview',
        name: 'overview',
        component: () => import('@/pages/overview/OverviewPage.vue'),
        meta: { title: '概览', menu: true, icon: AppstoreOutlined },
      },
      {
        path: 'providers',
        name: 'providers',
        component: () => import('@/pages/providers/ProvidersPage.vue'),
        meta: { title: '厂商与模型', menu: true, icon: ApiOutlined },
      },
      {
        path: 'plans',
        name: 'plans',
        component: () => import('@/pages/plans/PlansPage.vue'),
        meta: { title: '套餐与限额', menu: true, icon: SettingOutlined },
      },
      {
        path: 'users',
        name: 'users',
        component: () => import('@/pages/users/UsersPage.vue'),
        meta: { title: '用户管理', menu: true, icon: TeamOutlined },
      },
      {
        path: 'usage',
        name: 'usage',
        component: () => import('@/pages/usage/UsagePage.vue'),
        meta: { title: '用量统计', menu: true, icon: BarChartOutlined },
      },
      {
        path: 'settings',
        name: 'settings',
        component: () => import('@/pages/settings/SettingsPage.vue'),
        meta: { title: '系统设置', menu: true, icon: SettingOutlined },
      },
    ],
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'not-found',
    component: () => import('@/pages/NotFoundPage.vue'),
    meta: { title: '页面不存在', public: true },
  },
]
