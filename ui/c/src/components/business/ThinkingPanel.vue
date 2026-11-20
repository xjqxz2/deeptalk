<script setup lang="ts">
/** 深度思考折叠区：默认收起只显示耗时，展开可查看 reasoning。 */
import { computed, nextTick, ref, watch } from 'vue'
import { CheckCircleOutlined, DownOutlined, LoadingOutlined } from '@ant-design/icons-vue'
import type { SearchActivity } from '@deeptalk/shared'

const props = defineProps<{
  reasoning: string
  streaming: boolean
  durationMs: number
  activity?: SearchActivity | null
}>()

/** 是否展开推理正文。 */
const expanded = ref(false)
/** 推理正文滚动容器，流式期间自动跟到最新一行。 */
const bodyRef = ref<HTMLElement | null>(null)

// 流式思考时自动展开，让用户能看到 reasoning 逐字输出；完成后保持展开
watch(
  () => props.streaming,
  (streaming) => {
    // 流式期间自动展开；完成后保持展开，用户可手动收起
    if (streaming) expanded.value = true
  },
  { immediate: true },
)

// 推理文本增长时让面板内部也保持在底部
watch(
  () => props.reasoning,
  async () => {
    if (!expanded.value) return
    await nextTick()
    const element = bodyRef.value
    if (element) element.scrollTop = element.scrollHeight
  },
)

function toggle(): void {
  expanded.value = !expanded.value
}

/** 耗时展示：优先使用服务端记录的毫秒数，缺失时按文本长度粗估。 */
const seconds = computed(() => {
  // 有真实耗时就用真实耗时（不足 1 秒也显示 1 秒），避免演示流被估算成 7 秒
  if (props.durationMs > 0) return Math.max(1, Math.round(props.durationMs / 1000))
  return Math.max(3, Math.ceil(props.reasoning.length / 12))
})

/** 折叠态标题。 */
const label = computed(() => {
  if (props.streaming) return '正在思考…'
  return `已思考 ${seconds.value} 秒`
})

/** 只有流式期间展示工具活动；完成后不再占用思考面板。 */
const liveActivity = computed(() => (props.streaming ? props.activity : null))
</script>

<template>
  <div class="thinking" :class="{ expanded, streaming }">
    <button
      type="button"
      class="thinking-head"
      :aria-expanded="expanded"
      aria-label="展开或收起深度思考"
      @click="toggle"
    >
      <span class="mark" aria-hidden="true">
        <LoadingOutlined v-if="streaming" spin />
        <svg v-else viewBox="0 0 16 16" width="12" height="12">
          <path
            d="M8 1.2l1.62 4.86L14.8 8l-5.18 1.94L8 14.8 6.38 9.94 1.2 8l5.18-1.94L8 1.2Z"
            fill="currentColor"
          />
        </svg>
      </span>
      <span class="label">{{ label }}</span>
      <span class="spacer" />
      <span class="action">{{ expanded ? '收起' : '展开' }}</span>
      <DownOutlined class="chevron" :class="{ rotated: expanded }" />
    </button>

    <div v-if="liveActivity" class="thinking-activity" role="status" aria-live="polite">
      <CheckCircleOutlined v-if="liveActivity.status === 'found'" class="activity-icon done" />
      <LoadingOutlined v-else class="activity-icon" spin />
      <span class="activity-label">{{ liveActivity.label }}</span>
      <a
        v-if="liveActivity.url"
        class="activity-detail"
        :href="liveActivity.url"
        target="_blank"
        rel="noopener noreferrer"
        :title="liveActivity.detail || liveActivity.url"
      >
        {{ liveActivity.detail || liveActivity.url }}
      </a>
      <span v-else-if="liveActivity.detail" class="activity-detail" :title="liveActivity.detail">
        {{ liveActivity.detail }}
      </span>
    </div>

    <div v-if="expanded && reasoning" class="thinking-body">
      <div ref="bodyRef" class="thinking-content">{{ reasoning }}</div>
    </div>
  </div>
</template>

<style scoped>
.thinking {
  width: 100%;
  margin: 10px 0 12px;
  border: 1px solid var(--dt-line-soft);
  border-radius: 10px;
  background: #fafbfd;
  overflow: hidden;
}

.thinking-head {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  height: 38px;
  padding: 0 12px;
  border: none;
  background: transparent;
  color: var(--dt-text-secondary);
  font-size: 11.6px;
  font-weight: 550;
  cursor: pointer;
  text-align: left;
}

.thinking-head:hover {
  background: #f7f8fb;
}

.mark {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: var(--dt-brand);
  font-size: 12px;
}

.streaming .mark {
  color: var(--dt-brand);
}

.label {
  color: var(--dt-text-secondary);
}

.streaming .label {
  color: var(--dt-brand-hover);
}

.spacer {
  flex: 1;
}

.action {
  color: var(--dt-text-faint);
  font-size: 10.6px;
}

.chevron {
  color: var(--dt-text-faint);
  font-size: 9px;
  transition: transform 0.16s ease;
}

.chevron.rotated {
  transform: rotate(180deg);
}

.thinking-activity {
  display: flex;
  align-items: center;
  gap: 7px;
  min-width: 0;
  padding: 0 14px 10px;
  color: var(--dt-text-muted);
  font-size: 11.4px;
  line-height: 17px;
}

.activity-icon {
  flex: 0 0 auto;
  color: var(--dt-brand);
  font-size: 11px;
}

.activity-icon.done {
  color: var(--dt-success);
}

.activity-label {
  flex: 0 0 auto;
  color: var(--dt-text-secondary);
}

.activity-detail {
  min-width: 0;
  overflow: hidden;
  color: var(--dt-brand-hover);
  text-decoration: none;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.activity-detail:hover {
  text-decoration: underline;
}

.thinking-body {
  border-top: 1px solid var(--dt-line-soft);
  padding: 10px 14px 12px;
}

.thinking-content {
  max-height: 260px;
  overflow-y: auto;
  color: var(--dt-text-muted);
  font-size: 12px;
  line-height: 20px;
  white-space: pre-wrap;
  word-break: break-word;
}
</style>
