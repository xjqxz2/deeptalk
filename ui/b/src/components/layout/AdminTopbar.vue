<script setup lang="ts">
/** 后台顶栏：面包屑、全局搜索、通知与账号菜单。 */
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import {
  BellOutlined,
  DownOutlined,
  LogoutOutlined,
  MenuOutlined,
  MoreOutlined,
  QuestionCircleOutlined,
  SearchOutlined,
  SettingOutlined,
  SwapOutlined,
} from '@ant-design/icons-vue'
import { formatRelativeTime } from '@deeptalk/shared'
import type { AuditItem } from '@deeptalk/shared'
import { fetchAuditLogs } from '@/services/admin'
import { useAppStore } from '@/stores/app'
import { useUserStore } from '@/stores/user'

const route = useRoute()
const router = useRouter()
const app = useAppStore()
const userStore = useUserStore()

const query = ref('')
const audits = ref<AuditItem[]>([])

const title = computed(() => route.meta.title ?? '控制台')
const user = computed(() => userStore.user)

onMounted(async () => {
  try {
    const data = await fetchAuditLogs(5)
    audits.value = data.items
  } catch {
    audits.value = []
  }
})

/** 全局搜索：厂商 / 模型关键词去厂商页，其余去用户页。 */
async function submitSearch(): Promise<void> {
  const keyword = query.value.trim()
  if (!keyword) return
  const providerLike = /(deepseek|openai|vllm|ollama|moonshot|gpt|qwen|llama|claude|百炼|厂商|模型)/i.test(keyword)
  const target = providerLike
    ? { path: '/providers', query: { q: keyword } }
    : { path: '/users', query: { q: keyword } }
  await router.push(target)
  query.value = ''
}

/** 账号菜单。 */
function onUserMenu(info: { key: string }): void {
  if (info.key === 'workspace') {
    window.location.href = '/'
  }
  if (info.key === 'settings') {
    void router.push('/settings')
  }
  if (info.key === 'logout') {
    void logout()
  }
}

/** 更多菜单。 */
function onMoreMenu(info: { key: string }): void {
  if (info.key === 'guide') app.guideOpen = true
  if (info.key === 'workspace') window.location.href = '/'
  if (info.key === 'logout') void logout()
}

/** 退出登录。 */
async function logout(): Promise<void> {
  await userStore.logout()
  message.success('已退出登录')
  await router.push({ name: 'login' })
}
</script>

<template>
  <header class="admin-topbar">
    <div class="topbar-left">
      <button type="button" class="menu-btn" aria-label="打开导航" @click="app.toggleSidebar">
        <MenuOutlined />
      </button>
      <nav class="breadcrumb" aria-label="面包屑">
        <span class="crumb">控制台</span>
        <span class="slash">/</span>
        <span class="crumb current">{{ title }}</span>
      </nav>
    </div>

    <div class="topbar-right">
      <a-input
        v-model:value="query"
        class="global-search"
        placeholder="搜索用户、厂商或模型"
        aria-label="全局搜索"
        allow-clear
        @press-enter="submitSearch"
      >
        <template #prefix><SearchOutlined class="search-icon" /></template>
      </a-input>

      <a-popover placement="bottomRight" trigger="click">
        <template #content>
          <div class="notice-panel">
            <div class="notice-head">
              <strong>最近变更</strong>
              <button type="button" class="notice-link" @click="router.push('/providers')">查看全部</button>
            </div>
            <div v-if="audits.length" class="notice-list">
              <div v-for="item in audits" :key="item.id" class="notice-item">
                <span class="notice-dot" />
                <div class="notice-copy">
                  <p>{{ item.detail }}</p>
                  <small>{{ formatRelativeTime(item.created_at) }}</small>
                </div>
              </div>
            </div>
            <p v-else class="notice-empty">暂无变更记录</p>
          </div>
        </template>
        <button type="button" class="icon-btn" aria-label="通知">
          <BellOutlined />
          <span v-if="audits.length" class="notice-badge" />
        </button>
      </a-popover>

      <a-dropdown :trigger="['click']" placement="bottomRight">
        <button type="button" class="account" aria-label="账号菜单">
          <span class="avatar">{{ user?.avatar_text ?? 'A' }}</span>
          <span class="account-name">{{ user?.nickname ?? '管理员' }}</span>
          <DownOutlined class="account-arrow" />
        </button>
        <template #overlay>
          <a-menu @click="onUserMenu">
            <a-menu-item key="workspace">
              <SwapOutlined />
              返回对话工作台
            </a-menu-item>
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

      <a-dropdown :trigger="['click']" placement="bottomRight">
        <button type="button" class="icon-btn more" aria-label="更多">
          <MoreOutlined />
        </button>
        <template #overlay>
          <a-menu @click="onMoreMenu">
            <a-menu-item key="guide">
              <QuestionCircleOutlined />
              管理员指南
            </a-menu-item>
            <a-menu-item key="workspace">
              <SwapOutlined />
              返回对话工作台
            </a-menu-item>
            <a-menu-divider />
            <a-menu-item key="logout" danger>退出登录</a-menu-item>
          </a-menu>
        </template>
      </a-dropdown>
    </div>
  </header>
</template>

<style scoped>
.admin-topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  height: 60px;
  flex: 0 0 60px;
  padding: 0 28px 0 24px;
  border-bottom: 1px solid var(--dt-line);
  background: #fff;
}

.topbar-left {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.menu-btn {
  display: none;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border: none;
  border-radius: 7px;
  background: transparent;
  color: var(--dt-text-muted);
  font-size: 15px;
  cursor: pointer;
}

.breadcrumb {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--dt-text-faint);
  font-size: 11.6px;
  white-space: nowrap;
}

.crumb.current {
  color: var(--dt-text-secondary);
  font-weight: 550;
}

.slash {
  color: #d0d5dd;
}

.topbar-right {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.global-search {
  width: 360px;
  height: 34px;
  border-radius: 9px;
  background: #f7f8fa;
}

.global-search :deep(.ant-input-affix-wrapper) {
  height: 34px;
  border-color: transparent;
  border-radius: 9px;
  background: #f7f8fa;
  padding-inline: 11px;
}

.global-search :deep(input) {
  background: transparent;
  font-size: 11.4px;
}

.search-icon {
  color: var(--dt-text-faint);
  font-size: 12px;
}

.icon-btn {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  border: none;
  border-radius: 9px;
  background: transparent;
  color: var(--dt-text-muted);
  font-size: 15px;
  cursor: pointer;
}

.icon-btn:hover {
  background: #f5f7fb;
}

.notice-badge {
  position: absolute;
  top: 8px;
  right: 8px;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--dt-danger);
  box-shadow: 0 0 0 2px #fff;
}

.account {
  display: flex;
  align-items: center;
  gap: 7px;
  height: 36px;
  padding: 0 8px 0 4px;
  border: none;
  border-radius: 9px;
  background: transparent;
  cursor: pointer;
}

.account:hover {
  background: #f5f7fb;
}

.avatar {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: var(--dt-brand-soft);
  color: var(--dt-brand-hover);
  font-size: 11.8px;
  font-weight: 700;
}

.account-name {
  color: var(--dt-text-secondary);
  font-size: 11.6px;
  font-weight: 550;
  max-width: 90px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.account-arrow {
  color: var(--dt-text-faint);
  font-size: 8px;
}

.more {
  width: 26px;
  font-size: 10px;
}

.notice-panel {
  width: 300px;
}

.notice-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 2px 4px 10px;
  border-bottom: 1px solid var(--dt-line-soft);
}

.notice-head strong {
  color: var(--dt-text);
  font-size: 12.4px;
}

.notice-link {
  border: none;
  background: transparent;
  color: var(--dt-brand);
  font-size: 10.6px;
  cursor: pointer;
}

.notice-list {
  display: flex;
  flex-direction: column;
  gap: 9px;
  padding: 10px 4px 2px;
}

.notice-item {
  display: flex;
  gap: 8px;
}

.notice-dot {
  width: 6px;
  height: 6px;
  margin-top: 6px;
  border-radius: 50%;
  background: var(--dt-success);
  flex: 0 0 auto;
}

.notice-copy p {
  margin: 0 0 2px;
  color: var(--dt-text-secondary);
  font-size: 11.4px;
  line-height: 17px;
}

.notice-copy small {
  color: var(--dt-text-faint);
  font-size: 9.8px;
}

.notice-empty {
  margin: 16px 4px;
  color: var(--dt-text-faint);
  font-size: 11px;
  text-align: center;
}

@media (max-width: 1280px) {
  .global-search {
    width: 260px;
  }
}

@media (max-width: 1024px) {
  .admin-topbar {
    padding: 0 16px;
  }

  .menu-btn {
    display: inline-flex;
  }

  .global-search {
    width: 220px;
  }

  .account-name {
    display: none;
  }
}

@media (max-width: 720px) {
  .global-search {
    display: none;
  }

  .more {
    display: none;
  }
}
</style>
