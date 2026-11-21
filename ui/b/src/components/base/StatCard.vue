<script setup lang="ts">
/** 统计卡：数值 + 口径说明，四等分排列。 */
withDefaults(
  defineProps<{
    label: string
    value: string
    sub?: string
    tone?: 'default' | 'warning' | 'danger'
  }>(),
  { sub: '', tone: 'default' },
)
</script>

<template>
  <div class="stat-card" :class="tone">
    <span class="label">{{ label }}</span>
    <strong class="value">{{ value }}</strong>
    <span class="sub">{{ sub }}</span>
  </div>
</template>

<style scoped>
.stat-card {
  display: flex;
  flex-direction: column;
  justify-content: center;
  min-height: 94px;
  padding: 14px 18px;
  border: 1px solid var(--dt-line);
  border-radius: var(--dt-radius-card);
  background: var(--dt-surface);
  box-shadow: var(--dt-shadow-card);
}

.label {
  color: var(--dt-text-muted);
  font-size: 11px;
  font-weight: 500;
  line-height: 16px;
}

.value {
  margin: 3px 0 2px;
  color: var(--dt-text);
  font-size: 26px;
  line-height: 34px;
  font-weight: 730;
  letter-spacing: -0.6px;
}

.sub {
  color: var(--dt-text-faint);
  font-size: 10.2px;
  line-height: 15px;
}

.stat-card.warning .value {
  color: var(--dt-warning-text);
}

.stat-card.danger .value {
  color: var(--dt-danger-text);
}

@media (max-width: 640px) {
  .stat-card {
    min-height: 84px;
    padding: 12px 14px;
  }

  .value {
    font-size: 22px;
    line-height: 30px;
  }
}
</style>
