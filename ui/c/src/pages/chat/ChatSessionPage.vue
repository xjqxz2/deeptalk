<script setup lang="ts">
/**
 * 会话进行页：消息流、自动跟随滚动、回到底部、停止生成与重新生成。
 * 滚动策略：仅当用户停留在底部 80px 内才自动跟随；手动上滚后暂停。
 */
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { ArrowDownOutlined, WarningOutlined } from '@ant-design/icons-vue'
import type { ChatMessage } from '@deeptalk/shared'
import { useChatStore } from '@/stores/chat'
import Composer from '@/components/business/Composer.vue'
import MessageItem from '@/components/business/MessageItem.vue'

const route = useRoute()
const router = useRouter()
const chat = useChatStore()

/** 消息滚动容器。 */
const scroller = ref<HTMLElement | null>(null)
/** 消息列，用于监听内容高度变化（代码块、字体加载都会改变高度）。 */
const columnRef = ref<HTMLElement | null>(null)
let resizeObserver: ResizeObserver | null = null
/** 输入框内容。 */
const prompt = ref('')
/** 是否贴近底部。 */
const atBottom = ref(true)
/** 首次加载骨架。 */
const booting = ref(true)
/** 加载失败提示。 */
const loadError = ref('')

const lastMessage = computed<ChatMessage | null>(() => chat.messages[chat.messages.length - 1] ?? null)

/** 滚动到底部。 */
function scrollToBottom(smooth = false): void {
  const element = scroller.value
  if (!element) return
  element.scrollTo({ top: element.scrollHeight, behavior: smooth ? 'smooth' : 'auto' })
  atBottom.value = true
}

/** 监听内容高度变化；用户停留底部时保持贴底。 */
function observeColumn(): void {
  resizeObserver?.disconnect()
  if (!columnRef.value) return
  resizeObserver = new ResizeObserver(() => {
    if (atBottom.value) scrollToBottom()
  })
  resizeObserver.observe(columnRef.value)
}

/** 距离底部 80px 内视为“跟随中”。 */
function onScroll(): void {
  const element = scroller.value
  if (!element) return
  atBottom.value = element.scrollHeight - element.scrollTop - element.clientHeight < 80
}

/** 选择/粘贴图片后上传；失败时保留已上传的图片并提示。 */
async function handleAddFiles(files: File[]): Promise<void> {
  try {
    await chat.uploadFiles(files)
  } catch (error) {
    message.error(error instanceof Error ? error.message : '图片上传失败')
  }
}

/** 发送消息。 */
async function handleSend(): Promise<void> {
  const text = prompt.value.trim()
  if ((!text && chat.attachments.length === 0) || chat.generating) return
  prompt.value = ''
  atBottom.value = true
  try {
    await chat.send(text)
  } catch (error) {
    // 发送前置校验失败（如无可用模型）时把内容还给用户
    prompt.value = text
    message.error(error instanceof Error ? error.message : '发送失败，请稍后重试')
  }
}

/** 重新生成最后一条回答。 */
async function handleRegenerate(): Promise<void> {
  if (chat.generating) return
  atBottom.value = true
  try {
    await chat.regenerate()
  } catch (error) {
    message.error(error instanceof Error ? error.message : '重新生成失败，请稍后重试')
  }
}

/** 消息赞 / 踩。 */
async function handleFeedback(item: ChatMessage, rating: 'up' | 'down'): Promise<void> {
  try {
    await chat.feedback(item, rating)
  } catch (error) {
    message.error(error instanceof Error ? error.message : '操作失败')
  }
}

/** 加载会话；消费从空态带过来的首条消息。 */
async function load(): Promise<void> {
  const id = Number(route.params.id)
  if (!Number.isFinite(id) || id <= 0) {
    await router.replace({ name: 'chat-empty' })
    return
  }
  booting.value = true
  loadError.value = ''
  try {
    await chat.loadConversation(id)
    const pending = chat.consumePendingPrompt()
    // 先结束骨架：首条消息的流式请求不能在骨架状态下等待，否则整段生成都被遮住
    booting.value = false
    await nextTick()
    if (pending !== null) {
      prompt.value = pending
      void handleSend()
    } else {
      scrollToBottom()
    }
  } catch (error) {
    loadError.value = error instanceof Error ? error.message : '对话加载失败'
    message.error(loadError.value)
  } finally {
    booting.value = false
  }
}

onMounted(load)

watch(booting, async (value) => {
  if (value) return
  await nextTick()
  observeColumn()
  // 字体与代码高亮会在首帧后补一次布局，延迟再贴底一次
  window.setTimeout(() => {
    if (atBottom.value) scrollToBottom()
  }, 140)
})

onBeforeUnmount(() => {
  resizeObserver?.disconnect()
})

watch(
  () => route.params.id,
  async (value, previous) => {
    if (value && value !== previous) {
      prompt.value = ''
      await load()
    }
  },
)

// 流式增量到达时，如果用户在底部就持续跟随
watch(
  () => [chat.messages.length, lastMessage.value?.content.length ?? 0, lastMessage.value?.reasoning.length ?? 0],
  async () => {
    if (!atBottom.value) return
    await nextTick()
    scrollToBottom()
  },
)
</script>

<template>
  <div class="chat-session">
    <div ref="scroller" class="messages-scroll" @scroll.passive="onScroll">
      <div v-if="booting" class="skeleton-wrap">
        <div class="skeleton-row">
          <a-skeleton avatar :paragraph="{ rows: 2 }" active />
        </div>
        <div class="skeleton-row">
          <a-skeleton :paragraph="{ rows: 5 }" active />
        </div>
      </div>

      <div v-else-if="loadError" class="load-error">
        <WarningOutlined class="load-error-icon" />
        <p>{{ loadError }}</p>
        <a-button @click="load">重新加载</a-button>
      </div>

      <div v-else ref="columnRef" class="messages-column">
        <MessageItem
          v-for="item in chat.messages"
          :key="item.id"
          :message="item"
          :busy="chat.generating"
          @regenerate="handleRegenerate"
          @feedback="(rating: 'up' | 'down') => handleFeedback(item, rating)"
        />

        <div v-if="!chat.messages.length" class="empty-session">
          <p>这个对话还没有消息</p>
          <span>在下方输入你的问题，开始这次对话。</span>
        </div>
      </div>

      <button v-if="!atBottom" type="button" class="back-to-bottom" @click="scrollToBottom(true)">
        <ArrowDownOutlined />
        回到底部
      </button>
    </div>

    <div class="composer-dock">
      <div class="composer-inner">
        <Composer
          v-model:text="prompt"
          v-model:deep-thinking="chat.deepThinking"
          v-model:web-search="chat.webSearch"
          :web-search-enabled="chat.webSearchEnabled"
          :attachments="chat.attachments"
          :uploading="chat.uploading"
          :generating="chat.generating"
          :placeholder="chat.generating ? '正在生成中，可随时停止…' : '继续追问，或让 DeepTalk 补充更多细节…'"
          @add-files="handleAddFiles"
          @remove-attachment="chat.removeAttachment"
          @send="handleSend"
          @stop="chat.stop"
        />
      </div>
    </div>
  </div>
</template>

<style scoped>
.chat-session {
  display: flex;
  flex-direction: column;
  height: 100%;
  position: relative;
}

.messages-scroll {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  overscroll-behavior: contain;
}

.messages-column {
  flex: 0 0 auto;
  width: 100%;
  max-width: 752px;
  /* 消息从顶部开始排布：短对话不再被推到页面底部，长对话仍会按需滚动 */
  margin: 0 auto;
  padding: 22px 24px 18px;
}

.skeleton-wrap {
  max-width: 752px;
  margin: 0 auto;
  padding: 40px 24px;
}

.skeleton-row {
  margin-bottom: 28px;
}

.load-error {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  padding: 80px 24px;
  color: var(--dt-text-muted);
  font-size: 13px;
}

.load-error-icon {
  color: var(--dt-warning);
  font-size: 24px;
}

.load-error p {
  margin: 0;
}

.empty-session {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  padding: 100px 20px;
  text-align: center;
}

.empty-session p {
  margin: 0;
  color: var(--dt-text-secondary);
  font-size: 15px;
  font-weight: 600;
}

.empty-session span {
  color: var(--dt-text-faint);
  font-size: 12.4px;
}

.back-to-bottom {
  position: absolute;
  left: 50%;
  bottom: 154px;
  transform: translateX(-50%);
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 32px;
  padding: 0 12px;
  border: 1px solid var(--dt-line);
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.96);
  box-shadow: var(--dt-shadow-pop);
  color: var(--dt-text-secondary);
  font-size: 11.4px;
  cursor: pointer;
  z-index: 5;
}

.back-to-bottom:hover {
  color: var(--dt-brand-hover);
  border-color: #cfd9ff;
}

.composer-dock {
  flex: 0 0 auto;
  padding: 8px 24px 12px;
  background: linear-gradient(180deg, rgba(255, 255, 255, 0) 0%, #ffffff 26%);
}

.composer-inner {
  width: 100%;
  max-width: 752px;
  margin: 0 auto;
}

@media (max-width: 768px) {
  .messages-column {
    padding: 16px 16px 12px;
  }

  .composer-dock {
    padding: 6px 12px 8px;
  }

  .back-to-bottom {
    bottom: 148px;
  }
}
</style>
