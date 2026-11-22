<script setup lang="ts">
/** 模型编辑弹窗：上游 ID、展示名、上下文、并发与启停。 */
import { computed, reactive, ref, watch } from 'vue'
import { message } from 'ant-design-vue'
import { ApiError } from '@deeptalk/shared'
import type { AdminModel, ProviderInfo } from '@deeptalk/shared'
import { createModel, updateModel } from '@/services/admin'

const open = defineModel<boolean>('open', { default: false })
const props = defineProps<{ model: AdminModel | null; providers: ProviderInfo[]; defaultProviderId?: number }>()
const emit = defineEmits<{ (event: 'saved'): void }>()

const saving = ref(false)
const error = ref('')

const form = reactive({
  provider_id: 0,
  model_id: '',
  display_name: '',
  description: '',
  context_window: 128000,
  max_concurrency: 4,
  price_input: 0,
  price_cached_input: 0,
  price_output: 0,
  enabled: true,
})

/** 上下文窗口预设，避免用户输入 131072 之类的精确值。 */
const CONTEXT_OPTIONS = [
  { label: '8K', value: 8192 },
  { label: '32K', value: 32768 },
  { label: '64K', value: 64000 },
  { label: '128K', value: 128000 },
  { label: '256K', value: 256000 },
  { label: '1M', value: 1_048_576 },
]

const providerOptions = computed(() =>
  props.providers.map((item) => ({ label: `${item.name}${item.enabled ? '' : '（已停用）'}`, value: item.id })),
)
const isEditing = computed(() => props.model !== null)

watch(
  () => [open.value, props.model?.id ?? 0],
  () => {
    if (!open.value) return
    error.value = ''
    if (props.model) {
      form.provider_id = props.model.provider_id
      form.model_id = props.model.model_id
      form.display_name = props.model.display_name
      form.description = props.model.description
      form.context_window = props.model.context_window
      form.max_concurrency = props.model.max_concurrency
      form.price_input = props.model.price_input ?? 0
      form.price_cached_input = props.model.price_cached_input ?? 0
      form.price_output = props.model.price_output ?? 0
      form.enabled = props.model.enabled
    } else {
      form.provider_id = props.defaultProviderId ?? props.providers[0]?.id ?? 0
      form.model_id = ''
      form.display_name = ''
      form.description = ''
      form.context_window = 128000
      form.max_concurrency = 4
      form.price_input = 0
      form.price_cached_input = 0
      form.price_output = 0
      form.enabled = true
    }
  },
  { immediate: true },
)

/** 保存模型。 */
async function save(): Promise<void> {
  error.value = ''
  if (!form.provider_id) {
    error.value = '请选择所属厂商'
    return
  }
  if (!form.model_id.trim()) {
    error.value = '请填写上游模型 ID'
    return
  }
  if (form.max_concurrency < 1 || form.max_concurrency > 10000) {
    error.value = '并发需在 1–10000 之间'
    return
  }
  saving.value = true
  try {
    const payload = {
      provider_id: form.provider_id,
      model_id: form.model_id.trim(),
      display_name: form.display_name.trim() || form.model_id.trim(),
      description: form.description.trim(),
      context_window: form.context_window,
      price_input: form.price_input,
      price_cached_input: form.price_cached_input,
      price_output: form.price_output,
      max_concurrency: form.max_concurrency,
      enabled: form.enabled,
    }
    if (props.model) {
      await updateModel(props.model.id, payload)
      message.success('模型已更新')
    } else {
      await createModel(payload)
      message.success('模型已添加')
    }
    emit('saved')
    open.value = false
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : '保存失败，请稍后重试'
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <a-modal
    v-model:open="open"
    :title="isEditing ? '编辑模型' : '添加模型'"
    :width="480"
    :footer="null"
    destroy-on-close
  >
    <a-form layout="vertical" class="model-form">
      <a-form-item label="所属厂商" required>
        <a-select v-model:value="form.provider_id" :options="providerOptions" :disabled="isEditing" placeholder="选择厂商" />
      </a-form-item>
      <a-form-item label="上游模型 ID" required>
        <a-input v-model:value="form.model_id" placeholder="deepseek-flash" :maxlength="128" />
      </a-form-item>
      <a-form-item label="展示名称">
        <a-input v-model:value="form.display_name" placeholder="DeepSeek-V4.1-Flash" :maxlength="64" />
      </a-form-item>
      <a-form-item label="用途说明">
        <a-input v-model:value="form.description" placeholder="通用对话" :maxlength="128" />
      </a-form-item>
      <div class="two-col">
        <a-form-item label="上下文窗口">
          <a-select v-model:value="form.context_window" :options="CONTEXT_OPTIONS" />
        </a-form-item>
        <a-form-item label="并发">
          <a-input-number v-model:value="form.max_concurrency" :min="1" :max="10000" style="width: 100%" />
        </a-form-item>
      </div>
      <div class="price-row">
        <a-form-item label="输入价（元/百万）">
          <a-input-number v-model:value="form.price_input" :min="0" :max="10000" :step="0.1" style="width: 100%" />
        </a-form-item>
        <a-form-item label="缓存命中价">
          <a-input-number v-model:value="form.price_cached_input" :min="0" :max="10000" :step="0.1" style="width: 100%" />
        </a-form-item>
        <a-form-item label="输出价（元/百万）">
          <a-input-number v-model:value="form.price_output" :min="0" :max="10000" :step="0.1" style="width: 100%" />
        </a-form-item>
      </div>
      <p class="hint price-hint">上游不返回价格时（如免费模型），按这三个单价自行结算：普通输入、缓存命中输入、输出分别计价；缓存价留空按输入价计。</p>

      <div class="switch-row">
        <div>
          <span class="switch-label">启用该模型</span>
          <p class="hint">停用后不会出现在 C 端模型选择中。</p>
        </div>
        <a-switch v-model:checked="form.enabled" />
      </div>

      <div v-if="error" class="form-error" role="alert">{{ error }}</div>

      <div class="modal-footer">
        <a-button @click="open = false">取消</a-button>
        <a-button type="primary" :loading="saving" @click="save">保存</a-button>
      </div>
    </a-form>
  </a-modal>
</template>

<style scoped>
.model-form :deep(.ant-form-item) {
  margin-bottom: 13px;
}

.model-form :deep(.ant-form-item-label) {
  padding-bottom: 5px;
}

.model-form :deep(.ant-form-item-label > label) {
  color: var(--dt-text-secondary);
  font-size: 11.4px;
  font-weight: 600;
}

.two-col {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14px;
}

.switch-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  padding: 9px 12px;
  border: 1px solid var(--dt-line);
  border-radius: 9px;
}

.switch-label {
  color: var(--dt-text-secondary);
  font-size: 11.4px;
  font-weight: 600;
}

.hint {
  margin: 2px 0 0;
  color: var(--dt-text-faint);
  font-size: 9.8px;
}

.form-error {
  margin-top: 10px;
  padding: 8px 10px;
  border-radius: 8px;
  background: var(--dt-danger-soft);
  color: var(--dt-danger-text);
  font-size: 11px;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 18px;
  padding-top: 14px;
  border-top: 1px solid var(--dt-line-soft);
}

.price-row {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}

.price-hint {
  margin: -8px 0 12px;
}
</style>
