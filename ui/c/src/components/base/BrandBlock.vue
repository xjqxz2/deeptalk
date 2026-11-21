<script setup lang="ts">
/** 品牌标识块：图标 + 名称 + 可选副标题，侧栏与认证页共用。 */
import { BrandMark } from '@deeptalk/shared'

withDefaults(
  defineProps<{
    size?: number
    name?: string
    subtitle?: string
  }>(),
  { size: 30, name: '小橘树洞', subtitle: '' },
)
</script>

<template>
  <div class="brand-block">
    <BrandMark :size="size" :radius="Math.round(size * 0.28)" />
    <div class="brand-text">
      <span class="brand-name">{{ name }}</span>
      <span v-if="subtitle" class="brand-sub">{{ subtitle }}</span>
    </div>
  </div>
</template>

<style scoped>
.brand-block {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.brand-text {
  display: flex;
  flex-direction: column;
  min-width: 0;
  gap: 1px;
}

.brand-name {
  color: var(--dt-text);
  font-size: 15px;
  font-weight: 720;
  letter-spacing: -0.2px;
  line-height: 20px;
}

.brand-sub {
  color: var(--dt-text-faint);
  font-size: 8.6px;
  font-weight: 600;
  letter-spacing: 1.1px;
  line-height: 12px;
  white-space: nowrap;
}
</style>
