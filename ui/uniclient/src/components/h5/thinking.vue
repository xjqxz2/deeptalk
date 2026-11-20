<template>
  <div class="thinking" :class="{ expanded, streaming }">
    <Btn
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
    </Btn>

    <div v-if="liveActivity" class="thinking-activity" role="status">
      <LoadingOutlined v-if="liveActivity.status !== 'found'" class="activity-icon" spin />
      <CheckCircleOutlined v-else class="activity-icon done" />
      <span class="activity-label">{{ liveActivity.label }}</span>
      <a
        v-if="liveActivity.url"
        class="activity-detail"
        :href="liveActivity.url"
        target="_blank"
        rel="noopener noreferrer"
      >{{ liveActivity.detail || liveActivity.url }}</a>
      <span v-else-if="liveActivity.detail" class="activity-detail">{{ liveActivity.detail }}</span>
    </div>

    <div v-if="expanded && hasBody" class="thinking-body">
      <div v-if="reasoning" class="thinking-content">{{ reasoning }}</div>
      <div v-if="viewedSources.length" class="thinking-sources" :class="{ divided: Boolean(reasoning) }">
        <a
          v-for="source in viewedSources"
          :key="source.url"
          class="thinking-source"
          :href="source.url"
          target="_blank"
          rel="noopener noreferrer"
          :title="source.title || source.url"
        >
          <GlobalOutlined class="thinking-source-icon" />
          <span class="thinking-source-title">{{ source.title || source.url }}</span>
        </a>
      </div>
    </div>
  </div>
</template>

<script setup>
/** 深度思考面板：与 PWA 的 ThinkingPanel 结构、类名、图标完全一致。 */
import { computed, ref, watch } from 'vue'
import { CheckCircleOutlined, DownOutlined, GlobalOutlined, LoadingOutlined } from '@ant-design/icons-vue'
import Btn from './btn.vue'

const props = defineProps({
  reasoning: { type: String, default: '' },
  streaming: { type: Boolean, default: false },
  durationMs: { type: Number, default: 0 },
  activity: { type: Object, default: null },
  sources: { type: Array, default: () => [] },
})

const expanded = ref(false)

watch(
  () => props.streaming,
  (value) => {
    if (value) expanded.value = true
  },
  { immediate: true },
)

const viewedSources = computed(() => props.sources.filter((item) => item.viewed))
const liveActivity = computed(() => (props.streaming ? props.activity : null))
const hasBody = computed(() => Boolean(props.reasoning) || viewedSources.value.length > 0)

const label = computed(() => {
  if (props.streaming) return props.activity?.label || '正在思考…'
  const seconds = props.durationMs > 0 ? Math.max(1, Math.round(props.durationMs / 1000)) : Math.max(2, Math.ceil(props.reasoning.length / 12))
  return `已思考 ${seconds} 秒`
})

function toggle() {
  expanded.value = !expanded.value
}
</script>
