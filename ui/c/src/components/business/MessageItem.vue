<script setup lang="ts">
/**
 * 单条消息：用户右对齐气泡；助手左对齐，含深度思考、Markdown、代码块、操作栏。
 * 操作栏 hover 才出现，键盘聚焦时也可见。
 */
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import {
  CheckOutlined,
  CopyOutlined,
  DislikeOutlined,
  LikeOutlined,
  ReloadOutlined,
  WarningOutlined,
} from '@ant-design/icons-vue'
import { BrandMark, copyText, enhanceMarkdown, formatNumber, formatTime, renderMarkdown } from '@deeptalk/shared'
import type { ChatMessage, SearchActivity } from '@deeptalk/shared'
import A2uiRenderer from './A2uiRenderer.vue'
import ThinkingPanel from './ThinkingPanel.vue'

const props = withDefaults(
  defineProps<{
    message: ChatMessage
    busy: boolean
    /** 只读模式（分享页）：隐藏重新生成与赞踩 */
    readonly?: boolean
  }>(),
  { readonly: false },
)

const emit = defineEmits<{
  (event: 'regenerate'): void
  (event: 'feedback', rating: 'up' | 'down'): void
}>()

/** Markdown 容器，用于绑定代码复制委托。 */
const proseRef = ref<HTMLElement | null>(null)
/** 复制成功后的短暂状态。 */
const copied = ref(false)

/** 联网检索来源。 */
const sources = computed(() => props.message.sources ?? [])
/** 模型实际打开过的网页；没有打开过正文时不展示面板，避免噪音。 */
const viewedSources = computed(() => sources.value.filter((item) => item.viewed))
/** 正在搜索但没有收到新活动时，用旧 search 事件兜底，避免只显示省略号。 */
const thinkingActivity = computed<SearchActivity | null>(() => {
  if (props.message.activity) return props.message.activity
  if (props.message.searching) return { status: 'searching', label: '正在搜索资料…' }
  return null
})
/** 渲染后的安全 HTML；正文里的 [1] 会按来源列表变成可点击上标。 */
const html = computed(() => renderMarkdown(props.message.content, sources.value))
/** 是否处于生成中。 */
const streaming = computed(() => props.message.status === 'streaming')
/** 字数展示：优先使用服务端统计。 */
const characterCount = computed(() => props.message.character_count ?? props.message.content.length)
/** 用户消息附带的图片。 */
const userAttachments = computed(() => (props.message.role === 'user' ? (props.message.attachments ?? []) : []))
/** 是否展示操作栏。 */
const hasActions = computed(
  () =>
    props.message.content.length > 0 ||
    props.message.reasoning.length > 0 ||
    Boolean(props.message.error) ||
    props.message.status === 'stopped' ||
    props.message.status === 'error',
)
/** 生成速度：优先使用服务端统计，历史消息按耗时反推。 */
const tokensPerSec = computed(() => {
  if (props.message.tokens_per_sec) return props.message.tokens_per_sec
  if (props.message.output_tokens > 0 && props.message.duration_ms > 0) {
    return Math.round(props.message.output_tokens / (props.message.duration_ms / 1000))
  }
  return 0
})

/** 复制正文。 */
async function onCopy(): Promise<void> {
  const ok = await copyText(props.message.content)
  if (!ok) return
  copied.value = true
  window.setTimeout(() => {
    copied.value = false
  }, 1400)
}

/** 切换赞 / 踩，已选中的再次点击取消。 */
function onFeedback(rating: 'up' | 'down'): void {
  emit('feedback', rating)
}

onMounted(() => nextTick(() => enhanceMarkdown(proseRef.value)))

watch(html, () => {
  nextTick(() => enhanceMarkdown(proseRef.value))
})
</script>

<template>
  <div class="message" :class="[message.role, message.status]">
    <template v-if="message.role === 'user'">
      <div v-if="userAttachments.length" class="user-attachments">
        <a-image
          v-for="item in userAttachments"
          :key="item.id"
          class="user-image"
          :src="item.url"
          :alt="item.name"
          :width="112"
          :height="84"
        />
      </div>
      <div v-if="message.content" class="user-bubble">
        <span v-if="message.interjection" class="interject-tag">插话</span>
        {{ message.content }}
      </div>
      <div class="user-time">
        <span v-if="message.status === 'queued'" class="interject-pending">排队中，本轮结束后处理 · </span>
        {{ formatTime(message.created_at) }}
      </div>
    </template>

    <template v-else>
      <div class="assistant-head-row">
        <div class="assistant-avatar">
          <BrandMark :size="24" :radius="7" />
        </div>
        <div class="assistant-head">
          <span class="assistant-name">小橘树洞</span>
          <span class="assistant-model">
            {{ message.model_name || 'DeepSeek' }}
          </span>
        </div>
      </div>
      <div class="assistant-main">

        <ThinkingPanel
          v-if="message.reasoning || thinkingActivity || viewedSources.length"
          :reasoning="message.reasoning"
          :streaming="streaming"
          :duration-ms="message.duration_ms"
          :activity="thinkingActivity"
          :sources="sources"
        />

        <div v-if="message.toolBlocks?.length" class="tool-timeline">
          <div v-for="block in message.toolBlocks" :key="block.id" class="tool-chip" :class="block.status">
            <span class="tool-dot" aria-hidden="true" />
            <span class="tool-label">
              {{ block.kind === 'search' ? (block.status === 'running' ? '正在搜索' : '已搜索') : (block.status === 'running' ? '正在读取' : '已读取') }}
            </span>
            <span v-for="query in block.queries" :key="query" class="tool-query">{{ query }}</span>
            <span v-if="!block.queries.length && block.urls.length" class="tool-query">{{ block.urls[0] }}</span>
            <span v-if="block.status === 'failed'" class="tool-failed">失败</span>
          </div>
        </div>

        <div v-if="message.notice" class="dt-inline-notice notice">
          <WarningOutlined />
          {{ message.notice }}
        </div>

        <A2uiRenderer v-if="message.a2ui?.length" :messages="message.a2ui" />

        <div v-if="message.content" ref="proseRef" class="dt-prose prose" v-html="html" />
        <div v-else-if="streaming" class="typing" aria-label="正在生成回答">
          <span /><span /><span />
        </div>
        <span v-if="streaming && message.content" class="stream-cursor" aria-hidden="true" />

        <div v-if="message.error" class="error-box" role="alert">
          <WarningOutlined class="error-icon" />
          <span class="error-text">{{ message.error }}</span>
          <a-button size="small" :disabled="busy" @click="emit('regenerate')">重试</a-button>
        </div>

        <div v-if="hasActions" class="message-actions">
          <button v-if="message.content" type="button" class="action" @click="onCopy">
            <CheckOutlined v-if="copied" />
            <CopyOutlined v-else />
            {{ copied ? '已复制' : '复制' }}
          </button>
          <button
            v-if="!readonly"
            type="button"
            class="action"
            :disabled="busy"
            @click="emit('regenerate')"
          >
            <ReloadOutlined />
            重新生成
          </button>
          <button
            v-if="!readonly"
            type="button"
            class="action icon"
            :class="{ active: message.feedback === 'up' }"
            :aria-pressed="message.feedback === 'up'"
            aria-label="赞"
            @click="onFeedback('up')"
          >
            <LikeOutlined />
          </button>
          <button
            type="button"
            class="action icon"
            :class="{ active: message.feedback === 'down' }"
            :aria-pressed="message.feedback === 'down'"
            aria-label="踩"
            @click="onFeedback('down')"
          >
            <DislikeOutlined />
          </button>
          <span class="spacer" />
          <span v-if="message.status === 'stopped'" class="stats">已停止生成</span>
          <span v-else-if="message.status === 'complete'" class="stats">
            已生成 {{ formatNumber(characterCount) }} 字
            <template v-if="tokensPerSec"> · {{ tokensPerSec }} tok/s</template>
          </span>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.message {
  display: flex;
  width: 100%;
}

.message.user {
  flex-direction: column;
  align-items: flex-end;
  gap: 6px;
  padding: 8px 0 18px;
}

.user-attachments {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
  max-width: 78%;
}

.user-image {
  overflow: hidden;
  border: 1px solid var(--dt-line);
  border-radius: 10px;
  background: var(--dt-hover);
}

.user-image :deep(img) {
  object-fit: cover;
}

.user-bubble {
  max-width: 78%;
  padding: 11px 16px;
  border-radius: 14px;
  background: var(--dt-hover);
  color: var(--dt-text);
  font-size: 13.4px;
  line-height: 23px;
  white-space: pre-wrap;
  word-break: break-word;
}

.user-time {
  color: var(--dt-text-faint);
  font-size: 10px;
  padding-right: 4px;
}

.message.assistant {
  /* 用 grid 而不是 flex：桌面上头像占第一列、正文缩进；
     移动端让头部与正文都跨满整行，正文才不会整体偏右。 */
  display: grid;
  grid-template-columns: 24px minmax(0, 1fr);
  column-gap: 12px;
  align-items: start;
  padding: 10px 0 24px;
}

.assistant-head-row {
  display: contents;
}

.assistant-avatar {
  margin-top: 1px;
}

.assistant-main {
  grid-column: 2;
  min-width: 0;
}

@media (max-width: 768px) {
  .message.assistant {
    grid-template-columns: minmax(0, 1fr);
    row-gap: 6px;
  }

  /* 手机上把头像和模型名并成一行，正文接着占满整宽，视觉上左右对称 */
  .assistant-head-row {
    display: flex;
    grid-column: 1 / -1;
    align-items: center;
    gap: 8px;
  }

  .assistant-main {
    grid-column: 1 / -1;
  }
}

.assistant-head {
  display: flex;
  align-items: baseline;
  gap: 8px;
  margin-bottom: 8px;
  min-width: 0;
}

.assistant-name {
  color: var(--dt-text);
  font-size: 13px;
  font-weight: 680;
}

.assistant-model {
  color: var(--dt-text-faint);
  font-size: 10.4px;
  font-weight: 400;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.prose {
  min-width: 0;
}

.typing {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  height: 26px;
}

.typing span {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #c5cbda;
  animation: typing 1.1s infinite ease-in-out;
}

.typing span:nth-child(2) {
  animation-delay: 0.14s;
}

.typing span:nth-child(3) {
  animation-delay: 0.28s;
}

@keyframes typing {
  0%,
  70%,
  100% {
    opacity: 0.35;
    transform: translateY(0);
  }
  35% {
    opacity: 1;
    transform: translateY(-3px);
  }
}

.stream-cursor {
  display: inline-block;
  width: 7px;
  height: 14px;
  margin-left: 2px;
  vertical-align: -2px;
  border-radius: 2px;
  background: var(--dt-brand);
  animation: blink 0.9s steps(2, start) infinite;
}

@keyframes blink {
  50% {
    opacity: 0;
  }
}

.notice {
  margin-top: 10px;
}

.error-box {
  display: flex;
  align-items: center;
  gap: 9px;
  margin-top: 12px;
  padding: 9px 12px;
  border: 1px solid var(--dt-danger-soft);
  border-radius: 10px;
  background: var(--dt-danger-soft);
  color: var(--dt-danger-text);
  font-size: 12px;
  line-height: 18px;
}

.error-icon {
  flex: 0 0 auto;
}

.error-text {
  flex: 1;
  min-width: 0;
}

.message-actions {
  display: flex;
  align-items: center;
  gap: 2px;
  margin-top: 10px;
  min-height: 26px;
  opacity: 0;
  transition: opacity 0.16s ease;
}

.message:hover .message-actions,
.message:focus-within .message-actions {
  opacity: 1;
}

.action {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  height: 26px;
  padding: 0 7px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--dt-text-faint);
  font-size: 10.8px;
  cursor: pointer;
}

.action:hover:not(:disabled) {
  background: var(--dt-hover);
  color: var(--dt-text-secondary);
}

.action:disabled {
  cursor: not-allowed;
  opacity: 0.45;
}

.action.icon {
  width: 26px;
  justify-content: center;
  padding: 0;
}

.action.active {
  color: var(--dt-brand);
  background: var(--dt-brand-soft);
}

.spacer {
  flex: 1;
}

.stats {
  color: var(--dt-text-faint);
  font-size: 10.2px;
  white-space: nowrap;
}

@media (max-width: 768px) {
  .user-attachments {
    max-width: 88%;
  }

  .user-bubble {
    max-width: 88%;
  }

  .message-actions {
    opacity: 1;
  }

  .assistant {
    gap: 9px;
  }
}

.interject-tag {
  display: inline-flex;
  align-items: center;
  height: 18px;
  padding: 0 6px;
  border-radius: 5px;
  background: var(--dt-brand-soft);
  color: var(--dt-brand);
  font-size: 11px;
  font-weight: 600;
}

.interject-pending {
  color: var(--dt-text-muted);
  font-size: 11.5px;
}

.tool-timeline {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin: 8px 0;
}

.tool-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
  align-self: flex-start;
  padding: 5px 9px;
  border: 1px solid var(--dt-line);
  border-radius: 8px;
  background: var(--dt-surface);
  font-size: 12px;
  color: var(--dt-text-secondary);
}

.tool-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--dt-brand);
}

.tool-chip.running .tool-dot {
  animation: tool-pulse 1s ease-in-out infinite;
}

.tool-chip.failed .tool-dot {
  background: #d92d20;
}

.tool-label {
  font-weight: 600;
}

.tool-query {
  padding: 1px 6px;
  border-radius: 4px;
  background: var(--dt-hover, #f2f4f7);
  color: var(--dt-text-faint);
}

.tool-failed {
  color: #d92d20;
}

@keyframes tool-pulse {
  0%, 100% { opacity: 0.35; }
  50% { opacity: 1; }
}
</style>
