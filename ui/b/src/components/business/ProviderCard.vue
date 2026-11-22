<script setup lang="ts">
/** 厂商卡片：状态胶囊、模型/超时/并发、启停开关与默认标记。 */
import { computed } from 'vue'
import type { ProviderInfo } from '@deeptalk/shared'
import ProviderIcon from '@/components/base/ProviderIcon.vue'

const props = defineProps<{ provider: ProviderInfo; testing: boolean }>()

const emit = defineEmits<{
  (event: 'edit', provider: ProviderInfo): void
  (event: 'toggle', provider: ProviderInfo, enabled: boolean): void
  (event: 'test', provider: ProviderInfo): void
}>()

/** 状态文案。 */
const STATUS_LABEL: Record<ProviderInfo['status'], string> = {
  active: '已启用',
  pending: '待验证',
  failed: '连接失败',
  disabled: '已停用',
}

const apiStyleLabel = computed(() => {
  switch (props.provider.api_style) {
    case 'responses':
      return 'Responses API'
    case 'anthropic':
      return 'Anthropic'
    default:
      return 'Chat Completions'
  }
})

const statusLabel = computed(() => STATUS_LABEL[props.provider.status])
const busy = computed(() => props.testing)

/** 开关变化：父组件负责二次确认与请求。 */
function onToggle(checked: boolean): void {
  emit('toggle', props.provider, checked)
}
</script>

<template>
  <article class="provider-card" @click="emit('edit', provider)">
    <div class="card-head">
      <ProviderIcon :kind="provider.kind" :name="provider.name" :size="34" />
      <div class="identity">
        <strong>{{ provider.name }}</strong>
        <small :title="provider.base_url">{{ provider.base_url }}</small>
        <span class="api-style">{{ apiStyleLabel }}</span>
      </div>
      <span class="status" :class="provider.status">
        <i />
        {{ statusLabel }}
      </span>
    </div>

    <div class="card-stats">
      <div>
        <span>模型</span>
        <strong>{{ provider.model_count }} 个</strong>
      </div>
      <div>
        <span>超时</span>
        <strong>{{ provider.timeout_sec }}s</strong>
      </div>
      <div>
        <span>并发</span>
        <strong>{{ provider.max_concurrency }}</strong>
      </div>
    </div>

    <div class="card-foot">
      <!-- 用一层容器拦截冒泡，避免点开关时触发整张卡片的“打开配置” -->
      <span class="switch-wrap" @click.stop>
        <a-switch
          :checked="provider.enabled"
          size="small"
          :loading="busy"
          :aria-label="`${provider.name} 启用开关`"
          @change="onToggle"
        />
      </span>
      <span v-if="provider.is_default" class="tag default">默认</span>
      <span v-else-if="!provider.is_builtin" class="tag custom">自定义</span>
      <span class="spacer" />
      <button
        type="button"
        class="card-link"
        :disabled="busy"
        @click.stop="provider.status === 'active' || provider.status === 'disabled' ? emit('edit', provider) : emit('test', provider)"
      >
        <a-spin v-if="busy" size="small" />
        <template v-else>
          {{ provider.status === 'active' || provider.status === 'disabled' ? '配置' : '测试连接' }}
          <span aria-hidden="true">›</span>
        </template>
      </button>
    </div>
  </article>
</template>

<style scoped>
.provider-card {
  flex: 1 1 260px;
  min-width: 240px;
  /* 只有一个厂商时不要被拉成一整行那么宽 */
  max-width: 420px;
  display: flex;
  flex-direction: column;
  min-height: 158px;
  padding: 14px 16px 12px;
  overflow: visible;
  border: 1px solid var(--dt-line);
  border-radius: 14px;
  background: var(--dt-surface);
  box-shadow: var(--dt-shadow-card);
  cursor: pointer;
  transition: border-color 0.16s ease, box-shadow 0.16s ease, transform 0.16s ease;
  scroll-snap-align: start;
}

.provider-card:hover {
  border-color: var(--dt-focus-ring);
  box-shadow: 0 10px 22px rgba(16, 24, 40, 0.07);
  transform: translateY(-1px);
}

.card-head {
  display: flex;
  align-items: flex-start;
  gap: 9px;
  min-width: 0;
}

.identity {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
  flex: 1;
}

.identity strong {
  color: var(--dt-text);
  font-size: 13px;
  font-weight: 650;
  line-height: 18px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.identity small {
  color: var(--dt-text-faint);
  font-size: 9.6px;
  line-height: 14px;
  font-family: var(--dt-mono);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.status {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: 20px;
  padding: 0 7px;
  border-radius: 999px;
  font-size: 9.4px;
  font-weight: 600;
  white-space: nowrap;
  flex: 0 0 auto;
}

.status i {
  width: 4px;
  height: 4px;
  border-radius: 50%;
  background: currentColor;
}

.status.active {
  background: var(--dt-success-soft);
  color: var(--dt-success-text);
}

.status.pending {
  background: var(--dt-warning-soft);
  color: var(--dt-warning-text);
}

.status.failed {
  background: var(--dt-danger-soft);
  color: var(--dt-danger-text);
}

.status.disabled {
  background: var(--dt-hover);
  color: var(--dt-text-muted);
}

.card-stats {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
  margin: 14px 0 12px;
  padding-top: 12px;
  border-top: 1px solid var(--dt-line-soft);
}

.card-stats div {
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.card-stats span {
  color: var(--dt-text-faint);
  font-size: 9.4px;
}

.card-stats strong {
  color: var(--dt-text-secondary);
  font-size: 12px;
  font-weight: 650;
}

.card-foot {
  display: flex;
  align-items: center;
  gap: 7px;
  margin-top: auto;
  min-height: 24px;
}

.switch-wrap {
  display: inline-flex;
  align-items: center;
}

.tag {
  display: inline-flex;
  align-items: center;
  height: 18px;
  padding: 0 6px;
  border-radius: 4px;
  font-size: 8.8px;
  font-weight: 600;
}

.tag.default {
  background: var(--dt-brand-soft);
  color: var(--dt-brand-hover);
}

.tag.custom {
  background: var(--dt-hover);
  color: var(--dt-text-muted);
}

.spacer {
  flex: 1;
}

.card-link {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 0;
  border: none;
  background: transparent;
  color: var(--dt-brand);
  font-size: 10.2px;
  font-weight: 550;
  cursor: pointer;
}

.card-link:hover {
  color: var(--dt-brand-hover);
}

.card-link:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.api-style {
  display: inline-block;
  margin-top: 2px;
  padding: 1px 6px;
  border-radius: 4px;
  background: var(--dt-hover, #f2f4f7);
  color: var(--dt-text-faint, #667085);
  font-size: 11px;
}
</style>
