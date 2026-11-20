<script setup lang="ts">
/** 对话顶栏：标题、保存态、新建、模型选择、连接状态与账号菜单。 */
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import {
  LogoutOutlined,
  MenuOutlined,
  MobileOutlined,
  MoreOutlined,
  UserOutlined,
  WarningOutlined,
} from '@ant-design/icons-vue'
import { useAppStore } from '@/stores/app'
import { useChatStore } from '@/stores/chat'
import { usePwaInstall } from '@/composables/usePwaInstall'
import { useUserStore } from '@/stores/user'
import AccountSettingsModal from '@/components/business/AccountSettingsModal.vue'

const app = useAppStore()
const chat = useChatStore()
const userStore = useUserStore()
const router = useRouter()
const route = useRoute()
const { available: installAvailable, install: installApp } = usePwaInstall()

const accountOpen = ref(false)
const shortcutsOpen = ref(false)

const title = computed(() => {
  if (route.name === 'chat-empty') return '新对话'
  return chat.current?.title || '对话'
})

const connected = computed(() => chat.models.length > 0)
const initial = computed(() => userStore.user?.avatar_text ?? 'U')

/** 新建对话；生成中禁止切换。 */
async function newChat(): Promise<void> {
  if (chat.generating) {
    message.warning('正在生成回答，请先停止生成')
    return
  }
  chat.startNewDraft()
  await router.push({ name: 'chat-empty' })
}

/** 头像菜单。 */
function onUserMenu(info: { key: string }): void {
  if (info.key === 'install') void installApp()
  if (info.key === 'account') accountOpen.value = true
  if (info.key === 'logout') void logout()
}

/** 更多菜单。 */
function onMoreMenu(info: { key: string }): void {
  if (info.key === 'install') void installApp()
  if (info.key === 'shortcuts') shortcutsOpen.value = true
  if (info.key === 'new') void newChat()
  if (info.key === 'account') accountOpen.value = true
  if (info.key === 'logout') void logout()
}

/** 退出登录。 */
async function logout(): Promise<void> {
  await userStore.logout()
  chat.reset()
  await router.push({ name: 'login' })
}
</script>

<template>
  <header class="chat-topbar">
    <div class="topbar-left">
      <button
        v-if="app.sidebarCollapsed || app.isMobile"
        type="button"
        class="menu-btn"
        aria-label="展开侧栏"
        @click="app.toggleSidebar"
      >
        <MenuOutlined />
      </button>
      <h1 class="title" :title="title">{{ title }}</h1>
    </div>

    <div class="topbar-right">
      <span class="connection" :class="{ warning: !connected }">
        <span class="dot" />
        {{ connected ? '已连接' : '未配置模型' }}
      </span>
      <a-dropdown :trigger="['click']" placement="bottomRight">
        <button type="button" class="avatar" aria-label="账号菜单">{{ initial }}</button>
        <template #overlay>
          <a-menu @click="onUserMenu">
            <a-menu-item v-if="installAvailable" key="install">
              <MobileOutlined />
              安装到桌面
            </a-menu-item>
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
      <a-dropdown :trigger="['click']" placement="bottomRight">
        <button type="button" class="more-btn" aria-label="更多操作">
          <MoreOutlined />
        </button>
        <template #overlay>
          <a-menu @click="onMoreMenu">
            <a-menu-item v-if="installAvailable" key="install">
              <MobileOutlined />
              安装到桌面
            </a-menu-item>
            <a-menu-item key="shortcuts">键盘快捷键</a-menu-item>
            <a-menu-item key="new">新建对话</a-menu-item>
            <a-menu-divider />
            <a-menu-item key="logout" danger>
              <LogoutOutlined />
              退出登录
            </a-menu-item>
          </a-menu>
        </template>
      </a-dropdown>
    </div>

    <AccountSettingsModal v-model:open="accountOpen" />

    <a-modal v-model:open="shortcutsOpen" title="键盘快捷键" :footer="null" :width="420">
      <div class="shortcuts">
        <div class="shortcut-row"><span>发送消息</span><kbd>Enter</kbd></div>
        <div class="shortcut-row"><span>换行</span><kbd>Shift + Enter</kbd></div>
        <div class="shortcut-row"><span>停止生成</span><kbd>Esc</kbd></div>
        <div class="shortcut-row"><span>新建对话</span><kbd>Ctrl / ⌘ + K</kbd></div>
        <div class="shortcut-row"><span>关闭侧栏</span><kbd>Ctrl / ⌘ + B</kbd></div>
        <p class="shortcut-hint">
          <WarningOutlined />
          生成中切换会话会先停止当前生成，已输出内容会保留。
        </p>
      </div>
    </a-modal>
  </header>
</template>

<style scoped>
.chat-topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  height: calc(var(--dt-chat-topbar) + env(safe-area-inset-top));
  flex: 0 0 calc(var(--dt-chat-topbar) + env(safe-area-inset-top));
  padding: env(safe-area-inset-top) calc(18px + env(safe-area-inset-right)) 0
    calc(20px + env(safe-area-inset-left));
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
  display: inline-flex;
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

.menu-btn:hover {
  background: #f4f6fa;
}

.title {
  margin: 0;
  color: var(--dt-text);
  font-size: 13.6px;
  font-weight: 650;
  letter-spacing: -0.1px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 360px;
}

.topbar-right {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 0 0 auto;
}

.connection {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 26px;
  padding: 0 10px;
  border-radius: 999px;
  background: #f2fbf6;
  color: var(--dt-success-text);
  font-size: 10.2px;
  font-weight: 600;
  white-space: nowrap;
}

.connection.warning {
  background: var(--dt-warning-soft);
  color: var(--dt-warning-text);
}

.dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--dt-success);
}

.connection.warning .dot {
  background: var(--dt-warning);
}

.avatar {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border: none;
  border-radius: 50%;
  background: var(--dt-brand-soft);
  color: var(--dt-brand-hover);
  font-size: 12.6px;
  font-weight: 700;
  cursor: pointer;
}

.more-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 30px;
  border: none;
  border-radius: 7px;
  background: transparent;
  color: var(--dt-text-muted);
  font-size: 15px;
  cursor: pointer;
}

.more-btn:hover {
  background: #f4f6fa;
}

.shortcuts {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.shortcut-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 9px 2px;
  border-bottom: 1px solid var(--dt-line-soft);
  color: var(--dt-text-secondary);
  font-size: 12.4px;
}

.shortcut-row kbd {
  padding: 2px 7px;
  border: 1px solid var(--dt-line);
  border-radius: 6px;
  background: #f9fafc;
  color: var(--dt-text-muted);
  font-family: var(--dt-mono);
  font-size: 10.4px;
}

.shortcut-hint {
  display: flex;
  align-items: flex-start;
  gap: 6px;
  margin: 12px 0 0;
  color: var(--dt-text-faint);
  font-size: 11px;
  line-height: 17px;
}

@media (max-width: 900px) {
  .connection {
    display: none;
  }

  .title {
    max-width: 180px;
  }
}

@media (max-width: 640px) {
  .chat-topbar {
    padding: 0 12px;
  }

  .more-btn {
    display: none;
  }
}
</style>
