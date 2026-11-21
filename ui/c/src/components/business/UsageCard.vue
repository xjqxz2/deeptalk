<script setup lang="ts">
/**
 * 侧栏用量卡片：展示套餐与两类限额的当前用量。
 * 旧版按 token 计量的进度条已下线，改为「月消费 + 5 小时 / 本周 / 本月请求次数」。
 */
import { computed } from 'vue'
import type { UsageInfo } from '@deeptalk/shared'

const props = defineProps<{ usage: UsageInfo | null }>()

const limits = computed(() => props.usage?.limits ?? null)
const planName = computed(() => limits.value?.plan?.name || '免费版')
const spend = computed(() => ((limits.value?.spend_cents ?? 0) / 100).toFixed(2))
const spendLimit = computed(() => ((limits.value?.spend_limit_cents ?? 0) / 100).toFixed(2))
const rows = computed(() => {
  const value = limits.value
  if (!value) return []
  return [
    { label: '5 小时请求', used: value.requests_short },
    { label: '本周请求', used: value.requests_week },
    { label: '本月请求', used: value.requests_month },
  ]
})
</script>

<template>
  <div class="usage-card">
    <div class="usage-head">
      <span class="usage-title">{{ planName }}</span>
      <span class="usage-percent">本月 ¥{{ spend }} / ¥{{ spendLimit }}</span>
    </div>

    <div v-if="rows.length" class="usage-limits">
      <div v-for="row in rows" :key="row.label" class="usage-limit-row">
        <span class="usage-limit-label">{{ row.label }}</span>
        <span class="usage-limit-value">{{ row.used }}</span>
      </div>
    </div>
    <div v-else class="usage-unlimited">用量统计加载中…</div>
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
  align-items: baseline;
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
  font-size: 11.2px;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
}

.usage-limits {
  margin-top: 8px;
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

.usage-unlimited {
  margin-top: 8px;
  color: var(--dt-text-faint);
  font-size: 11px;
}
</style>
