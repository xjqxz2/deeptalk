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
// 套餐限额：月消费 + 5 小时 / 7 天 / 30 天请求次数
const limits = computed(() => props.usage?.limits ?? null)
const spend = computed(() => ((limits.value?.spend_cents ?? 0) / 100).toFixed(2))
const spendLimit = computed(() => ((limits.value?.spend_limit_cents ?? 0) / 100).toFixed(2))

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

    <template v-if="limits">
      <div class="usage-limits">
        <div class="usage-limit-row">
          <span class="usage-limit-label">本月消费</span>
          <span class="usage-limit-value">¥{{ spend }} / ¥{{ spendLimit }}</span>
        </div>
        <div class="usage-limit-row">
          <span class="usage-limit-label">5 小时请求</span>
          <span class="usage-limit-value">{{ limits.requests_short }}</span>
        </div>
        <div class="usage-limit-row">
          <span class="usage-limit-label">本周请求</span>
          <span class="usage-limit-value">{{ limits.requests_week }}</span>
        </div>
        <div class="usage-limit-row">
          <span class="usage-limit-label">本月请求</span>
          <span class="usage-limit-value">{{ limits.requests_month }}</span>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.usage-card {
  border: 1px solid var(--dt-line);
  border-radius: 10px;
  background: var(--dt-surface);
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
  background: var(--dt-line);
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
  color: var(--dt-danger-text);
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

.usage-limits {
  margin-top: 8px;
  padding-top: 8px;
  border-top: 1px dashed var(--dt-line);
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.usage-limit-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 11px;
}

.usage-limit-label {
  color: var(--dt-text-muted);
}

.usage-limit-value {
  color: var(--dt-text-secondary);
  font-variant-numeric: tabular-nums;
}
</style>
