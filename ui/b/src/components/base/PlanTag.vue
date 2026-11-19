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
  background: #f2f4f7;
  color: #475467;
}

.plan-tag.pro {
  background: #eef2ff;
  color: #3b5bdb;
}

.plan-tag.team {
  background: #ecfdf3;
  color: #067647;
}

.plan-tag.unlimited {
  background: #fff4ed;
  color: #b93815;
}
</style>
