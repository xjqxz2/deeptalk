<script setup lang="ts">
/** 页面标题区：标题 + 一行说明 + 右侧操作。 */
withDefaults(defineProps<{ title: string; subtitle?: string }>(), { subtitle: '' })
</script>

<template>
  <div class="page-header">
    <div class="heading">
      <h1>{{ title }}</h1>
      <p v-if="subtitle">{{ subtitle }}</p>
    </div>
    <div class="actions">
      <slot name="actions" />
    </div>
  </div>
</template>

<style scoped>
.page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
  margin-bottom: 20px;
}

.heading {
  min-width: 0;
}

h1 {
  margin: 0 0 6px;
  color: var(--dt-text);
  font-size: 22px;
  line-height: 1.3;
  font-weight: 730;
  letter-spacing: -0.4px;
}

p {
  margin: 0;
  color: var(--dt-text-muted);
  font-size: 11.8px;
  line-height: 18px;
}

.actions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex: 0 0 auto;
}

@media (max-width: 640px) {
  .page-header {
    flex-direction: column;
    gap: 12px;
  }

  .actions {
    width: 100%;
    flex-wrap: wrap;
  }
}
</style>
