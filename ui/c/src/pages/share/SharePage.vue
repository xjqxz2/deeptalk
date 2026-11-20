<script setup lang="ts">
/**
 * 公开分享页：只读展示一段对话，不需要登录。
 * 这里没有任何输入框或回复入口；想继续对话必须自己登录后新建。
 */
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { GlobalOutlined, LockOutlined } from '@ant-design/icons-vue'
import { BrandMark, formatDate } from '@deeptalk/shared'
import type { SharedConversation } from '@deeptalk/shared'
import * as chatApi from '@/services/chat'
import MessageItem from '@/components/business/MessageItem.vue'

const route = useRoute()
const router = useRouter()

const data = ref<SharedConversation | null>(null)
const loading = ref(true)
const error = ref('')

const ownerLabel = computed(() => data.value?.owner || '一位 DeepTalk 用户')
const sharedAt = computed(() => (data.value ? formatDate(data.value.shared_at) : ''))

/** 访客登录后只能开始自己的对话，不会接管被分享的会话。 */
async function startOwnChat(): Promise<void> {
  await router.push({ name: 'login' })
}

onMounted(async () => {
  const token = String(route.params.token ?? '')
  try {
    data.value = await chatApi.getShared(token)
  } catch (reason) {
    error.value = reason instanceof Error ? reason.message : '分享不存在或已失效'
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div class="share-page">
    <header class="share-header">
      <div class="share-brand">
        <BrandMark :size="26" :radius="8" />
        <span class="share-brand-name">DeepTalk</span>
      </div>
      <a-button size="small" @click="startOwnChat">登录后开始对话</a-button>
    </header>

    <main class="share-main">
      <div v-if="loading" class="share-state">
        <a-spin />
      </div>

      <div v-else-if="error" class="share-state">
        <LockOutlined class="share-state-icon" />
        <p class="share-state-title">{{ error }}</p>
        <span class="share-state-tip">链接可能已被取消分享，或者地址不完整。</span>
      </div>

      <template v-else-if="data">
        <div class="share-title-block">
          <h1>{{ data.title }}</h1>
          <p class="share-meta">
            <GlobalOutlined />
            {{ ownerLabel }} 分享于 {{ sharedAt }} · 只读
          </p>
        </div>

        <div class="share-messages">
          <MessageItem
            v-for="item in data.messages"
            :key="item.id"
            :message="item"
            :busy="true"
            readonly
          />
        </div>

        <div class="share-foot">
          <p>这是只读分享，无法在此继续提问或回复。</p>
          <a-button type="primary" @click="startOwnChat">开始我自己的对话</a-button>
        </div>
      </template>
    </main>
  </div>
</template>

<style scoped>
.share-page {
  display: flex;
  flex-direction: column;
  /* body 在桌面端是 overflow:hidden，分享页必须自己成为滚动容器，
     否则消息超过一屏后无法继续向下滚动 */
  height: 100vh;
  height: 100dvh;
  overflow-y: auto;
  overscroll-behavior-y: contain;
  -webkit-overflow-scrolling: touch;
  background: #fff;
}

.share-header {
  position: sticky;
  top: 0;
  z-index: 10;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  height: calc(56px + env(safe-area-inset-top));
  padding: env(safe-area-inset-top) calc(20px + env(safe-area-inset-right)) 0
    calc(20px + env(safe-area-inset-left));
  border-bottom: 1px solid var(--dt-line-soft);
  background: rgba(255, 255, 255, 0.92);
  backdrop-filter: blur(8px);
}

.share-brand {
  display: inline-flex;
  align-items: center;
  gap: 9px;
}

.share-brand-name {
  color: var(--dt-text);
  font-size: 14px;
  font-weight: 720;
  letter-spacing: -0.2px;
}

.share-main {
  flex: 0 0 auto;
  width: 100%;
  max-width: min(var(--dt-chat-max, 780px), 90vw);
  margin: 0 auto;
  padding: 28px calc(20px + env(safe-area-inset-right)) calc(40px + env(safe-area-inset-bottom))
    calc(20px + env(safe-area-inset-left));
}

.share-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  padding: 80px 20px;
  text-align: center;
}

.share-state-icon {
  color: var(--dt-text-faint);
  font-size: 24px;
}

.share-state-title {
  margin: 0;
  color: var(--dt-text-secondary);
  font-size: 14px;
  font-weight: 600;
}

.share-state-tip {
  color: var(--dt-text-faint);
  font-size: 12px;
}

.share-title-block {
  padding-bottom: 18px;
  border-bottom: 1px solid var(--dt-line-soft);
  margin-bottom: 18px;
}

.share-title-block h1 {
  margin: 0;
  color: var(--dt-text);
  font-size: 21px;
  line-height: 1.4;
  font-weight: 720;
  letter-spacing: -0.3px;
}

.share-meta {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 8px 0 0;
  color: var(--dt-text-faint);
  font-size: 11.6px;
}

.share-messages {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.share-foot {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  margin-top: 36px;
  padding-top: 22px;
  border-top: 1px solid var(--dt-line-soft);
}

.share-foot p {
  margin: 0;
  color: var(--dt-text-faint);
  font-size: 11.8px;
}
</style>
