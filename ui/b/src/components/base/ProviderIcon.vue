<script setup lang="ts">
/** 厂商图标：用品牌首字母组合区分类型，避免引入第三方 logo。 */
import { computed } from 'vue'

const props = withDefaults(defineProps<{ kind: string; name: string; size?: number }>(), { size: 34 })

/** 预置厂商缩写。 */
const PRESET: Record<string, string> = {
  deepseek: 'DS',
  openai: 'OA',
  aliyun: 'AL',
  moonshot: 'MO',
}

/** 自定义厂商取名称中有辨识度的两个字符。 */
const label = computed(() => {
  if (PRESET[props.kind]) return PRESET[props.kind]
  if (/vllm/i.test(props.name)) return 'vL'
  const cleaned = props.name.replace(/[^\p{L}\p{N}]/gu, '')
  return cleaned.slice(0, 2).toUpperCase() || 'AI'
})
</script>

<template>
  <span class="provider-icon" :class="kind" :style="{ width: `${size}px`, height: `${size}px` }">
    {{ label }}
  </span>
</template>

<style scoped>
.provider-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 9px;
  border: 1px solid var(--dt-line-soft);
  background: #f7f8fa;
  color: var(--dt-text-secondary);
  font-family: var(--dt-mono);
  font-size: 11px;
  font-weight: 700;
  letter-spacing: -0.2px;
  flex: 0 0 auto;
}

.provider-icon.deepseek {
  background: #eef2ff;
  border-color: #e0e7ff;
  color: #3b5bdb;
}

.provider-icon.openai {
  background: #f2f4f7;
  border-color: #e7eaef;
  color: #101828;
}

.provider-icon.aliyun {
  background: #fff4ed;
  border-color: #ffe4d5;
  color: #c2410c;
}

.provider-icon.moonshot {
  background: #f5f3ff;
  border-color: #ebe7ff;
  color: #5925dc;
}
</style>
