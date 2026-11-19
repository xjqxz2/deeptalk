<script setup lang="ts">
/** 本月用量卡片：进度条 + 已用/额度 + 重置天数，对应设计图侧栏底部。 */
import { computed } from 'vue'
import { formatPercent, formatTokens } from '@deeptalk/shared'
import type { UsageInfo } from '@deeptalk/shared'

const props = defineProps<{ usage: UsageInfo | null }>()

const percent = computed(() => props.usage?.used_percent ?? 0)
const unlimited = computed(() => (props.usage?.quota_tokens ?? 0) <= 0)
const used = computed(() => formatTokens(props.usage?.used_tokens ?? 0))
const quota = computed(() => formatTokens(props.usage?.quota_tokens ?? 0))
const resetDays = computed(() => props.usage?.reset_days ?? 0)
</script>

<template>
  <div class="usage-card">
    <div class="usage-head">
      <span class="usage-title">本月用量</span>
      <span class="usage-percent">{{ unlimited ? '不限' : formatPercent(percent) }}</span>
    </div>
    <div v-if="!unlimited" class="usage-bar" role="progressbar" :aria-valuenow="percent" aria-valuemin="0" aria-valuemax="100">
      <span class="usage-bar-fill" :class="{ warning: percent >= 90 }" :style="{ width: `${Math.min(100, percent)}%` }" />
    </div>
    <div v-else class="usage-unlimited">无限版 · 不设月度上限</div>
    <div class="usage-foot">
      <span>{{ used }} / {{ unlimited ? '无限' : quota }} tokens</span>
      <span class="reset">{{ unlimited ? '不限额度' : `重置 ${resetDays} 天` }}</span>
    </div>
  </div>
</template>

<style scoped>
.usage-card {
  border: 1px solid var(--dt-line);
  border-radius: 10px;
  background: #fff;
  padding: 10px 12px 11px;
}

.usage-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.usage-title {
  color: var(--dt-text-secondary);
  font-size: 11.4px;
  font-weight: 600;
}

.usage-percent {
  color: var(--dt-brand);
  font-size: 11.8px;
  font-weight: 720;
}

.usage-bar {
  height: 4px;
  margin: 8px 0 7px;
  border-radius: 999px;
  background: #eef0f4;
  overflow: hidden;
}

.usage-bar-fill {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: var(--dt-brand);
  transition: width 0.4s ease;
}

.usage-unlimited {
  margin: 8px 0 7px;
  color: #b93815;
  font-size: 10.3px;
}

.usage-bar-fill.warning {
  background: var(--dt-warning);
}

.usage-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  color: var(--dt-text-muted);
  font-size: 10.2px;
}

.reset {
  color: var(--dt-text-faint);
  white-space: nowrap;
}
</style>
