<script setup lang="ts">
/** 模型选择器：只列管理员启用的模型，切换只影响下一条消息。 */
import { computed } from 'vue'
import { useChatStore } from '@/stores/chat'

const chat = useChatStore()

const options = computed(() =>
  chat.models.map((model) => ({
    value: model.id,
    label: model.display_name,
    title: `${model.provider_name} · ${model.description}`,
  })),
)
</script>

<template>
  <a-select
    class="model-select"
    :value="chat.selectedModelId || undefined"
    :options="options"
    :bordered="false"
    size="small"
    placeholder="未配置模型"
    :dropdown-match-select-width="false"
    :disabled="chat.generating"
    aria-label="选择模型"
    @change="(value: number) => chat.setModel(value)"
  />
</template>

<style scoped>
.model-select {
  width: 156px;
}

.model-select :deep(.ant-select-selector) {
  background: transparent !important;
  border: none !important;
  box-shadow: none !important;
  padding: 0 22px 0 8px !important;
}

.model-select :deep(.ant-select-selection-item) {
  color: var(--dt-text-secondary);
  font-size: 11.8px;
  font-weight: 620;
  line-height: 28px;
}

.model-select :deep(.ant-select-arrow) {
  color: var(--dt-text-faint);
  font-size: 9px;
  inset-inline-end: 6px;
}

@media (max-width: 900px) {
  .model-select {
    width: 118px;
  }
}
</style>
