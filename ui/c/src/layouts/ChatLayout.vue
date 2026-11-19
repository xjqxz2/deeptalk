<script setup lang="ts">
/**
 * 对话主壳：侧栏 + 顶栏 + 内容区。
 * 负责一次性加载模型与会话、响应式断点、以及全局快捷键（Ctrl/⌘+K、Esc、Ctrl/⌘+B）。
 */
import { onBeforeUnmount, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { useAppStore } from '@/stores/app'
import { useChatStore } from '@/stores/chat'
import { fetchAuthConfig } from '@/services/auth'
import ChatSidebar from '@/components/layout/ChatSidebar.vue'
import ChatTopbar from '@/components/layout/ChatTopbar.vue'

const app = useAppStore()
const chat = useChatStore()
const router = useRouter()

/** 窗口宽度变化时同步断点。 */
function onResize(): void {
  app.syncViewport(window.innerWidth)
}

/** 全局快捷键：新建、停止、折叠侧栏。 */
function onKeydown(event: KeyboardEvent): void {
  const meta = event.metaKey || event.ctrlKey
  if (meta && event.key.toLowerCase() === 'k') {
    event.preventDefault()
    void newChat()
    return
  }
  if (meta && event.key.toLowerCase() === 'b') {
    event.preventDefault()
    app.toggleSidebar()
    return
  }
  if (event.key === 'Escape' && chat.generating) {
    event.preventDefault()
    chat.stop()
    message.info('已停止生成')
  }
}

/** 新建对话，生成中禁止切换。 */
async function newChat(): Promise<void> {
  if (chat.generating) {
    message.warning('正在生成回答，请先停止生成')
    return
  }
  chat.startNewDraft()
  app.closeMobileSidebar()
  await router.push({ name: 'chat-empty' })
}

onMounted(async () => {
  app.syncViewport(window.innerWidth)
  window.addEventListener('resize', onResize)
  window.addEventListener('keydown', onKeydown)
  try {
    await Promise.all([chat.loadModels(), chat.loadConversations()])
  } catch {
    message.error('初始化失败，请确认后端服务已启动')
  }
  // 联网搜索能力由后端配置决定；失败时保留开关，由服务端返回未启用提示
  try {
    const config = await fetchAuthConfig()
    chat.setWebSearchEnabled(config.web_search_enabled)
  } catch {
    // 公开配置失败不阻断工作台
  }
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', onResize)
  window.removeEventListener('keydown', onKeydown)
})
</script>

<template>
  <div class="chat-shell">
    <ChatSidebar />
    <button
      v-if="app.isMobile && app.mobileSidebarOpen"
      type="button"
      class="sidebar-backdrop"
      aria-label="关闭侧栏"
      @click="app.closeMobileSidebar"
    />
    <div class="chat-main">
      <ChatTopbar />
      <main class="chat-content">
        <router-view />
      </main>
    </div>
  </div>
</template>

<style scoped>
.chat-shell {
  display: flex;
  height: 100vh;
  height: 100svh;
  height: 100dvh;
  overflow: hidden;
  background: #fff;
}

.chat-main {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-width: 0;
  background: #fff;
}

.chat-content {
  position: relative;
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.sidebar-backdrop {
  position: fixed;
  inset: 0;
  z-index: 35;
  border: none;
  background: rgba(16, 24, 40, 0.32);
  backdrop-filter: blur(1px);
  cursor: pointer;
}
</style>
