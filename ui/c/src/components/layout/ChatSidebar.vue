<script setup lang="ts">
/**
 * 对话侧栏：新建、搜索、按今天/7 天/30 天分组的历史、用量与账号。
 * 1024 以下折叠为 64px 图标条；768 以下变为抽屉式覆盖层。
 */
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { message, Modal } from 'ant-design-vue'
import {
  DeleteOutlined,
  EditOutlined,
  LogoutOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  MessageOutlined,
  MoreOutlined,
  PlusOutlined,
  SearchOutlined,
  UserOutlined,
} from '@ant-design/icons-vue'
import { BrandMark, formatDate, planLabel } from '@deeptalk/shared'
import type { Conversation } from '@deeptalk/shared'
import { useAppStore } from '@/stores/app'
import { useChatStore } from '@/stores/chat'
import { useUserStore } from '@/stores/user'
import BrandBlock from '@/components/base/BrandBlock.vue'
import UsageCard from '@/components/business/UsageCard.vue'
import AccountSettingsModal from '@/components/business/AccountSettingsModal.vue'

const app = useAppStore()
const chat = useChatStore()
const userStore = useUserStore()
const router = useRouter()
const route = useRoute()

/** 搜索关键词，300ms 防抖后请求服务端。 */
const keyword = ref(chat.searchKeyword)
const searchRef = ref<{ focus: () => void } | null>(null)
let searchTimer: ReturnType<typeof setTimeout> | null = null

/** 重命名弹窗状态。 */
const renameOpen = ref(false)
const renameTarget = ref<Conversation | null>(null)
const renameTitle = ref('')
const renaming = ref(false)
/** 账号弹窗。 */
const accountOpen = ref(false)

const collapsed = computed(() => app.sidebarCollapsed && !app.isMobile)
const user = computed(() => userStore.user)
const usage = computed(() => userStore.usage)
const currentId = computed(() => chat.current?.id ?? 0)

/** 生成中禁止切换会话，与设计约定一致。 */
const navigationLocked = computed(() => chat.generating)

watch(keyword, (value) => {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    void chat.search(value)
  }, 300)
})

// 新建草稿会清空 store 的搜索词，侧栏输入框需要同步回空
watch(
  () => chat.searchKeyword,
  (value) => {
    if (value !== keyword.value) keyword.value = value
  },
)

onBeforeUnmount(() => {
  if (searchTimer) clearTimeout(searchTimer)
})

/** 新建对话并进入会话页。 */
async function newChat(): Promise<void> {
  if (navigationLocked.value) {
    message.warning('正在生成回答，请先停止生成')
    return
  }
  chat.startNewDraft()
  app.closeMobileSidebar()
  await router.push({ name: 'chat-empty' })
}

/** 打开历史会话。 */
async function openConversation(item: Conversation): Promise<void> {
  if (navigationLocked.value) {
    message.warning('正在生成回答，请先停止生成')
    return
  }
  app.closeMobileSidebar()
  const ref = item.public_id || String(item.id)
  if (String(route.params.id) === ref) return
  await router.push({ name: 'chat-session', params: { id: ref } })
}

/** 展开侧栏并聚焦搜索框。 */
async function expandSearch(): Promise<void> {
  app.sidebarCollapsed = false
  await nextTick()
  searchRef.value?.focus()
}

/** 打开重命名弹窗。 */
function openRename(item: Conversation): void {
  renameTarget.value = item
  renameTitle.value = item.title
  renameOpen.value = true
}

/** 提交重命名。 */
async function submitRename(): Promise<void> {
  const target = renameTarget.value
  const title = renameTitle.value.trim()
  if (!target) return
  if (!title || title.length > 60) {
    message.warning('标题需为 1–60 个字符')
    return
  }
  renaming.value = true
  try {
    await chat.renameConversation(target.id, title)
    renameOpen.value = false
    message.success('已重命名')
  } catch {
    message.error('重命名失败，请稍后重试')
  } finally {
    renaming.value = false
  }
}

/** 删除会话（二次确认）。 */
function confirmDelete(item: Conversation): void {
  Modal.confirm({
    title: '删除这个对话？',
    content: `「${item.title}」及其消息将被永久删除，无法恢复。`,
    okText: '删除',
    okType: 'danger',
    cancelText: '取消',
    async onOk() {
      await chat.deleteConversation(item.id)
      if (String(route.params.id) === (item.public_id || String(item.id))) {
        await router.push({ name: 'chat-empty' })
      }
      message.success('对话已删除')
    },
  })
}

/** 会话更多操作。 */
function onItemMenu(info: { key: string }, item: Conversation): void {
  if (info.key === 'rename') openRename(item)
  if (info.key === 'delete') confirmDelete(item)
}

/** 账号菜单。 */
function onUserMenu(info: { key: string }): void {
  if (info.key === 'account') accountOpen.value = true
  if (info.key === 'logout') void logout()
}

/** 退出登录。 */
async function logout(): Promise<void> {
  await userStore.logout()
  chat.reset()
  app.closeMobileSidebar()
  await router.push({ name: 'login' })
}

/** 取会话标题首字符作为折叠态图标。 */
function titleInitial(title: string): string {
  return title.trim().slice(0, 1) || '话'
}

/** 会话时间提示。 */
function conversationTime(item: Conversation): string {
  return formatDate(item.updated_at)
}
</script>

<template>
  <aside
    class="chat-sidebar"
    :class="{ collapsed, mobile: app.isMobile, 'mobile-open': app.mobileSidebarOpen }"
    aria-label="对话历史"
  >
    <div class="sidebar-head">
      <button v-if="collapsed" type="button" class="brand-mini" aria-label="展开侧栏" @click="app.toggleSidebar">
        <BrandMark :size="28" :radius="8" />
      </button>
      <BrandBlock v-else :size="30" subtitle="" />
      <button
        v-if="!collapsed"
        type="button"
        class="collapse-btn"
        :aria-label="app.isMobile ? '关闭侧栏' : '收起侧栏'"
        @click="app.toggleSidebar"
      >
        <MenuFoldOutlined v-if="!app.isMobile" />
        <MenuUnfoldOutlined v-else />
      </button>
    </div>

    <template v-if="!collapsed">
      <div class="sidebar-body">
        <button type="button" class="new-chat" :class="{ locked: navigationLocked }" @click="newChat">
          <PlusOutlined />
          <span class="new-label">新建对话</span>
          <kbd>Ctrl K</kbd>
        </button>

        <a-input
          ref="searchRef"
          v-model:value="keyword"
          class="search-input"
          placeholder="搜索对话"
          allow-clear
          aria-label="搜索对话"
        >
          <template #prefix>
            <SearchOutlined class="search-icon" />
          </template>
        </a-input>

        <div class="history">
          <template v-if="chat.groupedConversations.length">
            <div v-for="group in chat.groupedConversations" :key="group.label" class="history-group">
              <div class="group-label">{{ group.label }}</div>
              <div
                v-for="item in group.items"
                :key="item.id"
                class="history-item"
                :class="{ active: currentId === item.id, locked: navigationLocked }"
                role="button"
                tabindex="0"
                @click="openConversation(item)"
                @keydown.enter="openConversation(item)"
              >
                <MessageOutlined class="item-icon" />
                <span class="item-title" :title="item.title">{{ item.title }}</span>
                <a-dropdown :trigger="['click']" placement="bottomRight">
                  <button type="button" class="more" aria-label="会话操作" @click.stop>
                    <MoreOutlined />
                  </button>
                  <template #overlay>
                    <a-menu @click="onItemMenu($event, item)">
                      <a-menu-item key="rename">
                        <EditOutlined />
                        重命名
                      </a-menu-item>
                      <a-menu-item key="delete" danger>
                        <DeleteOutlined />
                        删除
                      </a-menu-item>
                    </a-menu>
                  </template>
                </a-dropdown>
              </div>
            </div>
          </template>
          <div v-else class="history-empty">
            <span v-if="chat.loadingConversations">正在加载…</span>
            <span v-else-if="keyword">没有匹配的对话</span>
            <span v-else>还没有对话，开始第一段吧</span>
          </div>
        </div>
      </div>

      <div class="sidebar-foot">
        <UsageCard :usage="usage" />
        <a-dropdown placement="topRight" :trigger="['click']">
          <button type="button" class="user-card" aria-label="账号菜单">
            <span class="user-avatar">{{ user?.avatar_text ?? 'U' }}</span>
            <span class="user-meta">
              <span class="user-name">{{ user?.nickname ?? '未登录' }}</span>
              <span class="user-email" :title="user?.email">{{ user?.email }}</span>
            </span>
            <MoreOutlined class="user-more" />
          </button>
          <template #overlay>
            <a-menu @click="onUserMenu">
              <div class="user-menu-head">
                <span class="user-menu-name">{{ user?.nickname }}</span>
                <span class="user-menu-plan">{{ planLabel(user?.plan ?? 'free') }}</span>
              </div>
              <a-menu-divider />
              <a-menu-item key="account">
                <UserOutlined />
                账号与密码
              </a-menu-item>
              <a-menu-item key="logout">
                <LogoutOutlined />
                退出登录
              </a-menu-item>
            </a-menu>
          </template>
        </a-dropdown>
      </div>
    </template>

    <template v-else>
      <div class="collapsed-body">
        <a-tooltip title="新建对话" placement="right">
          <button type="button" class="rail-btn primary" aria-label="新建对话" @click="newChat">
            <PlusOutlined />
          </button>
        </a-tooltip>
        <a-tooltip title="搜索对话" placement="right">
          <button type="button" class="rail-btn" aria-label="搜索对话" @click="expandSearch">
            <SearchOutlined />
          </button>
        </a-tooltip>
        <div class="rail-divider" />
        <a-tooltip
          v-for="item in chat.conversations.slice(0, 7)"
          :key="item.id"
          :title="`${item.title} · ${conversationTime(item)}`"
          placement="right"
        >
          <button
            type="button"
            class="rail-conv"
            :class="{ active: currentId === item.id }"
            :aria-label="item.title"
            @click="openConversation(item)"
          >
            {{ titleInitial(item.title) }}
          </button>
        </a-tooltip>
      </div>
      <div class="collapsed-foot">
        <a-tooltip :title="user?.nickname" placement="right">
          <button type="button" class="rail-avatar" aria-label="账号设置" @click="accountOpen = true">
            {{ user?.avatar_text ?? 'U' }}
          </button>
        </a-tooltip>
      </div>
    </template>

    <AccountSettingsModal v-model:open="accountOpen" />

    <a-modal
      v-model:open="renameOpen"
      title="重命名对话"
      :confirm-loading="renaming"
      ok-text="保存"
      cancel-text="取消"
      @ok="submitRename"
    >
      <a-input v-model:value="renameTitle" :maxlength="60" placeholder="请输入对话标题" @press-enter="submitRename" />
    </a-modal>
  </aside>
</template>

<style scoped>
.chat-sidebar {
  display: flex;
  flex-direction: column;
  width: var(--dt-chat-sidebar);
  height: 100%;
  flex: 0 0 var(--dt-chat-sidebar);
  background: var(--dt-sidebar);
  border-right: 1px solid var(--dt-line);
  transition: width 0.18s ease, flex-basis 0.18s ease, transform 0.2s ease;
  z-index: 40;
}

.chat-sidebar.collapsed {
  width: var(--dt-chat-sidebar-collapsed);
  flex-basis: var(--dt-chat-sidebar-collapsed);
}

.sidebar-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  height: 64px;
  padding: 0 14px 0 18px;
  flex: 0 0 64px;
}

.collapsed .sidebar-head {
  justify-content: center;
  padding: 0;
}

.brand-mini {
  display: inline-flex;
  padding: 0;
  border: none;
  background: transparent;
  cursor: pointer;
}

.collapse-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: none;
  border-radius: 7px;
  background: transparent;
  color: var(--dt-text-muted);
  font-size: 15px;
  cursor: pointer;
}

.collapse-btn:hover {
  background: #eef1f6;
  color: var(--dt-text);
}

.sidebar-body {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
  padding: 2px 12px 0;
}

.new-chat {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  height: 38px;
  padding: 0 11px;
  border: 1px solid #e4e7ee;
  border-radius: 10px;
  background: #fff;
  color: var(--dt-text-secondary);
  font-size: 12.4px;
  font-weight: 600;
  cursor: pointer;
  transition: border-color 0.16s ease, color 0.16s ease, background 0.16s ease;
}

.new-chat:hover {
  border-color: var(--dt-brand);
  color: var(--dt-brand-hover);
  background: var(--dt-brand-softer);
}

.new-chat.locked {
  opacity: 0.6;
}

.new-label {
  flex: 1;
  text-align: left;
}

.new-chat kbd {
  border: none;
  background: transparent;
  color: var(--dt-text-disabled);
  font-family: var(--dt-mono);
  font-size: 9.4px;
  letter-spacing: 0.2px;
}

.search-input {
  margin: 10px 0 6px;
  height: 34px;
  border-radius: 9px;
  background: #fff;
}

.search-input :deep(.ant-input-affix-wrapper) {
  border-color: #e7eaf0;
  background: #fff;
  border-radius: 9px;
  padding-inline: 10px;
}

.search-input :deep(input) {
  font-size: 11.8px;
}

.search-icon {
  color: var(--dt-text-faint);
  font-size: 12px;
}

.history {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 4px 0 12px;
  margin-right: -4px;
  padding-right: 4px;
}

.history-group + .history-group {
  margin-top: 10px;
}

.group-label {
  padding: 8px 8px 5px;
  color: var(--dt-text-faint);
  font-size: 10.2px;
  font-weight: 650;
  letter-spacing: 0.5px;
}

.history-item {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 34px;
  padding: 0 7px 0 9px;
  border-radius: 8px;
  color: var(--dt-text-secondary);
  font-size: 12.4px;
  cursor: pointer;
  outline: none;
}

.history-item:hover {
  background: #f1f3f8;
}

.history-item.active {
  background: var(--dt-brand-soft);
  color: var(--dt-brand-hover);
  font-weight: 600;
}

.history-item.locked {
  cursor: not-allowed;
}

.item-icon {
  flex: 0 0 auto;
  font-size: 12.6px;
  color: var(--dt-text-faint);
}

.history-item.active .item-icon {
  color: var(--dt-brand);
}

.item-title {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.more {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border: none;
  border-radius: 5px;
  background: transparent;
  color: var(--dt-text-muted);
  opacity: 0;
  cursor: pointer;
}

.history-item:hover .more,
.history-item:focus-within .more,
.history-item.active .more {
  opacity: 1;
}

.more:hover {
  background: #fff;
}

.history-empty {
  padding: 26px 10px;
  color: var(--dt-text-faint);
  font-size: 11.4px;
  text-align: center;
}

.sidebar-foot {
  flex: 0 0 auto;
  padding: 10px 12px 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.user-card {
  display: flex;
  align-items: center;
  gap: 9px;
  width: 100%;
  padding: 7px 8px;
  border: none;
  border-radius: 9px;
  background: transparent;
  text-align: left;
  cursor: pointer;
}

.user-card:hover {
  background: #f1f3f8;
}

.user-avatar,
.rail-avatar {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border-radius: 50%;
  background: var(--dt-brand-soft);
  color: var(--dt-brand-hover);
  font-size: 12.6px;
  font-weight: 700;
  flex: 0 0 auto;
}

.user-meta {
  display: flex;
  flex-direction: column;
  gap: 1px;
  min-width: 0;
  flex: 1;
}

.user-name {
  color: var(--dt-text);
  font-size: 12.2px;
  font-weight: 620;
  line-height: 16px;
}

.user-email {
  color: var(--dt-text-faint);
  font-family: var(--dt-mono);
  font-size: 9.4px;
  line-height: 13px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.user-more {
  color: var(--dt-text-faint);
  font-size: 13px;
}

.user-menu-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 8px 12px 6px;
}

.user-menu-name {
  color: var(--dt-text);
  font-size: 12.4px;
  font-weight: 650;
}

.user-menu-plan {
  color: var(--dt-text-faint);
  font-size: 10.4px;
}

.collapsed-body {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  flex: 1;
  min-height: 0;
  padding: 10px 0;
  overflow-y: auto;
}

.rail-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 38px;
  height: 38px;
  border: none;
  border-radius: 10px;
  background: transparent;
  color: var(--dt-text-muted);
  font-size: 15px;
  cursor: pointer;
}

.rail-btn:hover {
  background: #eef1f6;
  color: var(--dt-text);
}

.rail-btn.primary {
  background: #fff;
  border: 1px solid #e3e7ef;
  color: var(--dt-brand);
  box-shadow: 0 1px 2px rgba(16, 24, 40, 0.04);
}

.rail-divider {
  width: 24px;
  height: 1px;
  margin: 4px 0;
  background: var(--dt-line);
}

.rail-conv {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border: none;
  border-radius: 9px;
  background: transparent;
  color: var(--dt-text-muted);
  font-size: 11.4px;
  font-weight: 600;
  cursor: pointer;
}

.rail-conv:hover {
  background: #eef1f6;
}

.rail-conv.active {
  background: var(--dt-brand-soft);
  color: var(--dt-brand-hover);
}

.collapsed-foot {
  display: flex;
  justify-content: center;
  padding: 10px 0 12px;
  flex: 0 0 auto;
}

.rail-avatar {
  border: none;
  cursor: pointer;
}

.chat-sidebar.mobile {
  position: fixed;
  inset: 0 auto 0 0;
  height: 100vh;
  height: 100svh;
  height: 100dvh;
  width: 280px;
  flex-basis: 280px;
  transform: translateX(-100%);
  box-shadow: 0 18px 40px rgba(16, 24, 40, 0.18);
}

.chat-sidebar.mobile.mobile-open {
  transform: translateX(0);
}
</style>
