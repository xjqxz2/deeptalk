<script setup lang="ts">
/** 套餐标签：颜色只表达层级，不承载业务状态。 */
import { planLabel } from '@deeptalk/shared'

withDefaults(defineProps<{ plan: string }>(), { plan: 'free' })
</script>

<template>
  <span class="plan-tag" :class="plan">{{ planLabel(plan) }}</span>
</template>

<style scoped>
.plan-tag {
  display: inline-flex;
  align-items: center;
  height: 20px;
  padding: 0 7px;
  border-radius: 5px;
  font-size: 9.8px;
  font-weight: 600;
  white-space: nowrap;
}

.plan-tag.free {
  background: var(--dt-hover);
  color: #475467;
}

.plan-tag.pro {
  background: var(--dt-brand-soft);
  color: var(--dt-brand-hover);
}

.plan-tag.team {
  background: var(--dt-success-soft);
  color: #067647;
}

.plan-tag.unlimited {
  background: var(--dt-warning-soft);
  color: var(--dt-danger-text);
}
</style>
