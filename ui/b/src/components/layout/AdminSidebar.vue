<script setup lang="ts">
/**
 * 后台侧栏：品牌、控制台菜单、存储用量、管理员指南与账号。
 * 菜单项完全由路由表派生，新增页面只需改 config/routes.ts。
 */
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import {
  ArrowLeftOutlined,
  BookOutlined,
  CloudServerOutlined,
  LogoutOutlined,
  MoreOutlined,
  SettingOutlined,
} from '@ant-design/icons-vue'
import { BrandMark, formatBytes, formatNumber } from '@deeptalk/shared'
import { appRoutes } from '@config/routes'
import { useAdminStore } from '@/stores/admin'
import { useAppStore } from '@/stores/app'
import { useUserStore } from '@/stores/user'

const route = useRoute()
const router = useRouter()
const app = useAppStore()
const admin = useAdminStore()
const userStore = useUserStore()

/** 只保留 meta.menu 的页面。 */
const menuRoutes = computed(() => {
  const layout = appRoutes.find((item) => item.path === '/')
  return (layout?.children ?? []).filter((item) => item.meta?.menu)
})

const user = computed(() => userStore.user)
const percentage = computed(() => {
  const used = admin.settings?.storage_used_bytes ?? 0
  const total = admin.settings?.storage_total_bytes ?? 1
  return Math.max(0, Math.min(100, Math.round((used / total) * 100)))
})

/** 侧栏菜单点击后关闭移动端抽屉。 */
function onNavigate(): void {
  app.closeSidebar()
}

/** 账号菜单。 */
function onUserMenu(info: { key: string }): void {
  if (info.key === 'settings') {
    void router.push('/settings')
  }
  if (info.key === 'logout') {
    void logout()
  }
}

/** 退出登录。 */
async function logout(): Promise<void> {
  await userStore.logout()
  message.success('已退出登录')
  await router.push({ name: 'login' })
}
</script>

<template>
  <aside class="admin-sidebar" aria-label="控制台导航">
    <div class="sidebar-brand">
      <BrandMark :size="30" :radius="8.4" />
      <span class="brand-name">小橘树洞</span>
      <span class="admin-badge">ADMIN</span>
    </div>

    <nav class="sidebar-nav">
      <p class="nav-label">控制台</p>
      <router-link
        v-for="item in menuRoutes"
        :key="item.path"
        :to="`/${item.path}`"
        class="nav-item"
        :class="{ active: route.path === `/${item.path}` }"
        @click="onNavigate"
      >
        <component :is="item.meta?.icon" class="nav-icon" />
        <span>{{ item.meta?.title }}</span>
      </router-link>

      <a class="nav-item back-link" href="/">
        <ArrowLeftOutlined class="nav-icon" />
        <span>返回对话工作台</span>
      </a>
    </nav>

    <div class="sidebar-bottom">
      <button type="button" class="storage-card" @click="void router.push('/settings')">
        <div class="storage-head">
          <span class="storage-title">存储用量</span>
          <span class="storage-percent">{{ percentage }}%</span>
        </div>
        <div class="storage-bar">
          <span :style="{ width: `${Math.max(percentage, 1)}%` }" />
        </div>
        <div class="storage-foot">
          <span>{{ formatBytes(admin.settings?.storage_used_bytes ?? 0) }} / {{ formatBytes(admin.settings?.storage_total_bytes ?? 0) }}</span>
        </div>
        <div class="storage-meta">
          <span><CloudServerOutlined /> SQLite · 单实例部署</span>
          <span>本月对话 {{ formatNumber(admin.overview?.monthly_messages ?? 0) }} 条</span>
        </div>
      </button>

      <button type="button" class="guide-card" @click="app.guideOpen = true">
        <span class="guide-icon"><BookOutlined /></span>
        <span class="guide-copy">
          <strong>管理员指南</strong>
          <small>查看密钥托管与路由说明</small>
          <small class="guide-version">v0.3 · MVP</small>
        </span>
      </button>

      <a-dropdown :trigger="['click']" placement="topRight">
        <button type="button" class="admin-user" aria-label="管理员账号菜单">
          <span class="avatar">{{ user?.avatar_text ?? 'A' }}</span>
          <span class="user-copy">
            <strong>{{ user?.nickname ?? '管理员' }}</strong>
            <small>平台管理员</small>
          </span>
          <MoreOutlined class="user-more" />
        </button>
        <template #overlay>
          <a-menu @click="onUserMenu">
            <a-menu-item key="settings">
              <SettingOutlined />
              系统设置
            </a-menu-item>
            <a-menu-divider />
            <a-menu-item key="logout" danger>
              <LogoutOutlined />
              退出登录
            </a-menu-item>
          </a-menu>
        </template>
      </a-dropdown>
    </div>
  </aside>
</template>

<style scoped>
.admin-sidebar {
  display: flex;
  flex-direction: column;
  width: var(--dt-admin-sidebar);
  flex: 0 0 var(--dt-admin-sidebar);
  height: 100vh;
  height: 100dvh;
  border-right: 1px solid var(--dt-line);
  background: var(--dt-surface);
  z-index: 40;
}

.sidebar-brand {
  display: flex;
  align-items: center;
  gap: 9px;
  height: 64px;
  padding: 0 18px;
  flex: 0 0 64px;
}

.brand-name {
  color: var(--dt-text);
  font-size: 15px;
  font-weight: 730;
  letter-spacing: -0.2px;
}

.admin-badge {
  padding: 2px 6px;
  border-radius: 5px;
  background: var(--dt-brand-soft);
  color: var(--dt-brand-hover);
  font-size: 8.6px;
  font-weight: 700;
  letter-spacing: 0.6px;
}

.sidebar-nav {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 6px 12px;
}

.nav-label {
  margin: 6px 10px 8px;
  color: var(--dt-text-faint);
  font-size: 10.4px;
  font-weight: 650;
  letter-spacing: 0.6px;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 10px;
  height: 40px;
  padding: 0 11px;
  margin-bottom: 2px;
  border-radius: 9px;
  color: var(--dt-text-secondary);
  font-size: 12.6px;
  font-weight: 500;
  transition: background 0.15s ease, color 0.15s ease;
}

.nav-item:hover {
  background: var(--dt-hover);
  color: var(--dt-text);
}

.nav-item.active {
  background: var(--dt-brand-soft);
  color: var(--dt-brand-hover);
  font-weight: 620;
}

.nav-icon {
  flex: 0 0 auto;
  font-size: 14px;
  color: var(--dt-text-faint);
}

.nav-item.active .nav-icon {
  color: var(--dt-brand);
}

.back-link {
  margin-top: 14px;
  border-top: 1px solid var(--dt-line-soft);
  padding-top: 12px;
  height: 44px;
  border-radius: 0 0 9px 9px;
  color: var(--dt-text-muted);
  font-size: 11.6px;
}

.sidebar-bottom {
  flex: 0 0 auto;
  display: flex;
  flex-direction: column;
  gap: 9px;
  padding: 10px 12px 12px;
}

.storage-card,
.guide-card,
.admin-user {
  width: 100%;
  border: 1px solid var(--dt-line);
  border-radius: 10px;
  background: var(--dt-surface);
  text-align: left;
  cursor: pointer;
  padding: 10px 11px;
}

.storage-card:hover,
.guide-card:hover,
.admin-user:hover {
  border-color: #d8def0;
  background: #fcfdff;
}

.storage-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.storage-title {
  color: var(--dt-text-secondary);
  font-size: 11.2px;
  font-weight: 600;
}

.storage-percent {
  color: var(--dt-brand);
  font-size: 11.6px;
  font-weight: 720;
}

.storage-bar {
  height: 4px;
  margin: 8px 0 6px;
  border-radius: 999px;
  background: var(--dt-line);
  overflow: hidden;
}

.storage-bar span {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: var(--dt-brand);
}

.storage-foot {
  color: var(--dt-text-muted);
  font-size: 10px;
}

.storage-meta {
  display: flex;
  flex-direction: column;
  gap: 3px;
  margin-top: 7px;
  padding-top: 7px;
  border-top: 1px solid var(--dt-line-soft);
  color: var(--dt-text-faint);
  font-size: 9.6px;
}

.storage-meta span {
  display: flex;
  align-items: center;
  gap: 5px;
}

.guide-card {
  display: flex;
  align-items: flex-start;
  gap: 9px;
  padding: 11px;
}

.guide-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border-radius: 50%;
  border: 1px solid #dfe6ff;
  background: var(--dt-brand-softer);
  color: var(--dt-brand);
  font-size: 11px;
  flex: 0 0 auto;
}

.guide-copy {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.guide-copy strong {
  color: var(--dt-text-secondary);
  font-size: 11.2px;
  font-weight: 620;
}

.guide-copy small {
  color: var(--dt-text-faint);
  font-size: 9.6px;
  line-height: 14px;
}

.guide-version {
  margin-top: 2px;
  color: var(--dt-text-disabled) !important;
  font-family: var(--dt-mono);
  font-size: 9px !important;
}

.admin-user {
  display: flex;
  align-items: center;
  gap: 9px;
  padding: 8px 9px;
  border-color: transparent;
  background: transparent;
}

.avatar {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border-radius: 50%;
  background: var(--dt-brand-soft);
  color: var(--dt-brand-hover);
  font-size: 12.4px;
  font-weight: 700;
  flex: 0 0 auto;
}

.user-copy {
  display: flex;
  flex-direction: column;
  min-width: 0;
  flex: 1;
}

.user-copy strong {
  color: var(--dt-text);
  font-size: 12px;
  font-weight: 620;
}

.user-copy small {
  color: var(--dt-text-faint);
  font-size: 9.6px;
}

.user-more {
  color: var(--dt-text-faint);
  font-size: 13px;
}

@media (max-width: 1024px) {
  .admin-sidebar {
    position: fixed;
    inset: 0 auto 0 0;
    transform: translateX(-100%);
    transition: transform 0.2s ease;
    box-shadow: var(--dt-shadow-float);
  }

  .admin-sidebar.mobile-open {
    transform: translateX(0);
  }
}
</style>
