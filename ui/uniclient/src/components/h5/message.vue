<template>
  <div class="message" :class="[message.role, message.status]">
    <template v-if="message.role === 'user'">
      <div v-if="userAttachments.length" class="user-attachments">
        <img
          v-for="item in userAttachments"
          :key="item.id"
          class="user-image"
          :src="item.url"
          :alt="item.name"
          width="112"
          height="84"
        />
      </div>
      <div v-if="message.content" class="user-bubble">{{ message.content }}</div>
      <div class="user-time">{{ time }}</div>
    </template>

    <template v-else>
      <div class="assistant-head-row">
        <div class="assistant-avatar">
          <BrandMark :size="24" :radius="7" />
        </div>
        <div class="assistant-head">
          <span class="assistant-name">小橘树洞</span>
          <span class="assistant-model">{{ message.model_name || 'DeepSeek' }}</span>
        </div>
      </div>

      <div class="assistant-main">
        <Thinking
          v-if="message.reasoning || (streaming && activity) || viewedSources.length"
          :reasoning="message.reasoning"
          :streaming="streaming"
          :duration-ms="message.duration_ms || 0"
          :activity="activity"
          :sources="message.sources"
        />

        <A2uiRenderer
          v-if="message.a2ui && message.a2ui.length"
          :messages="message.a2ui"
          @action="emit('a2ui-action', $event)"
        />

        <div v-if="message.notice" class="dt-inline-notice notice">
          <WarningOutlined />
          {{ message.notice }}
        </div>

        <div v-if="message.content" ref="proseRef" class="dt-prose prose" v-html="html" />
        <div v-else-if="streaming" class="typing" aria-label="正在生成回答"><span /><span /><span /></div>
        <span v-if="streaming && message.content" class="stream-cursor" aria-hidden="true" />

        <div v-if="message.error" class="error-box" role="alert">
          <WarningOutlined class="error-icon" />
          <span class="error-text">{{ message.error }}</span>
          <Btn type="button" class="error-retry" @click="emit('retry')">重试</Btn>
        </div>

        <div v-if="hasActions" class="message-actions">
          <Btn v-if="message.content" type="button" class="action" @click="onCopy">
            <CheckOutlined v-if="copied" />
            <CopyOutlined v-else />
            {{ copied ? '已复制' : '复制' }}
          </Btn>
          <Btn type="button" class="action" :disabled="busy" @click="emit('regenerate')">
            <ReloadOutlined />
            重新生成
          </Btn>
          <Btn
            type="button"
            class="action icon"
            :class="{ active: message.feedback === 'up' }"
            aria-label="赞"
            @click="emit('rate', 'up')"
          >
            <LikeOutlined />
          </Btn>
          <Btn
            type="button"
            class="action icon"
            :class="{ active: message.feedback === 'down' }"
            aria-label="踩"
            @click="emit('rate', 'down')"
          >
            <DislikeOutlined />
          </Btn>
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

<script setup>
/**
 * H5 端消息组件：结构与 class 名与 PWA 的 MessageItem 一致，
 * 正文直接复用 PWA 的 Markdown 管线（v-html + dt-prose 样式）。
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
import Thinking from './thinking.vue'
import Btn from './btn.vue'
import BrandMark from './brand-mark.vue'
import A2uiRenderer from '@/components/a2ui/renderer.nvue'
import { enhanceMarkdown, renderMarkdown } from '@/utils/pwa-markdown.js'

const props = defineProps({
  message: { type: Object, required: true },
  busy: { type: Boolean, default: false },
  activity: { type: Object, default: null },
})
const emit = defineEmits(['regenerate', 'rate', 'retry', 'a2ui-action'])

const proseRef = ref(null)
const copied = ref(false)

const streaming = computed(() => props.message.status === 'streaming')
const sources = computed(() => props.message.sources || [])
const viewedSources = computed(() => sources.value.filter((item) => item.viewed))
const userAttachments = computed(() => (props.message.role === 'user' ? props.message.attachments || [] : []))
const html = computed(() => renderMarkdown(props.message.content || '', sources.value))
const characterCount = computed(() => props.message.character_count ?? (props.message.content || '').length)
const hasActions = computed(
  () => Boolean(props.message.content) || Boolean(props.message.reasoning) || Boolean(props.message.error),
)
const tokensPerSec = computed(() => {
  if (props.message.tokens_per_sec) return props.message.tokens_per_sec
  if (props.message.output_tokens > 0 && props.message.duration_ms > 0) {
    return Math.round(props.message.output_tokens / (props.message.duration_ms / 1000))
  }
  return 0
})
const time = computed(() => {
  const raw = props.message.created_at
  if (!raw) return ''
  const date = new Date(raw)
  if (Number.isNaN(date.getTime())) return ''
  return `${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`
})

function formatNumber(value) {
  return new Intl.NumberFormat('zh-CN').format(value || 0)
}

async function onCopy() {
  try {
    await navigator.clipboard.writeText(props.message.content || '')
    copied.value = true
    setTimeout(() => { copied.value = false }, 1400)
  } catch {
    /* 忽略剪贴板权限问题 */
  }
}

onMounted(() => nextTick(() => enhanceMarkdown(proseRef.value)))
watch(html, () => nextTick(() => enhanceMarkdown(proseRef.value)))
</script>
