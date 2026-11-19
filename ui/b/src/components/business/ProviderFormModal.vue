<script setup lang="ts">
/**
 * 厂商配置弹窗：预置模板 / 自定义兼容 API、模型列表、并发与连接测试。
 * 安全约定：已保存的 Key 只显示掩码，输入为空表示保留原值。
 */
import { computed, reactive, ref, watch } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { DeleteOutlined, PlusOutlined } from '@ant-design/icons-vue'
import { ApiError } from '@deeptalk/shared'
import type { ProviderInfo, ProviderPayload, TestResult } from '@deeptalk/shared'
import { createProvider, deleteProvider, testDraft, updateProvider } from '@/services/admin'

const open = defineModel<boolean>('open', { default: false })
const props = defineProps<{ provider: ProviderInfo | null }>()
const emit = defineEmits<{ (event: 'saved'): void; (event: 'deleted'): void }>()

/** 模型草稿。 */
interface ModelDraft {
  model_id: string
  display_name: string
  description: string
  context_window: number
  max_concurrency: number
  enabled: boolean
}

/** 预置厂商模板。 */
const PRESETS: Array<{ kind: ProviderInfo['kind']; name: string; base_url: string; models: string[] }> = [
  { kind: 'deepseek', name: 'DeepSeek', base_url: 'https://api.deepseek.com', models: ['deepseek-flash'] },
]

const mode = ref<'preset' | 'custom'>('custom')
const presetCode = ref<string>('deepseek')
const testing = ref(false)
const saving = ref(false)
const testResult = ref<TestResult | null>(null)
const error = ref('')
const newModelId = ref('')

const form = reactive({
  name: '',
  kind: 'custom' as ProviderInfo['kind'],
  base_url: '',
  api_key: '',
  timeout_sec: 60,
  max_concurrency: 4,
  enabled: true,
  is_default: false,
  models: [] as ModelDraft[],
})

const isEditing = computed(() => props.provider !== null)
const canDelete = computed(() => Boolean(props.provider && !props.provider.is_builtin))
const keyPlaceholder = computed(() => {
  if (props.provider?.has_api_key) return `已配置 ${props.provider.api_key_masked}（留空表示不修改）`
  return 'sk-...'
})
const modeOptions = [
  { label: '预置厂商', value: 'preset' },
  { label: '自定义兼容 API', value: 'custom' },
]
const presetOptions = PRESETS.map((item) => ({ label: item.name, value: item.kind }))

/** 用模板填充名称、地址与模型。 */
function applyPreset(kind: string): void {
  const preset = PRESETS.find((item) => item.kind === kind)
  if (!preset) return
  presetCode.value = kind
  form.kind = preset.kind
  form.name = preset.name
  form.base_url = preset.base_url
  if (form.models.length === 0) {
    form.models = preset.models.map((id) => ({
      model_id: id,
      display_name: id,
      description: '',
      context_window: 128000,
      max_concurrency: 4,
      enabled: true,
    }))
  }
}

/** 重置表单。 */
function reset(): void {
  const provider = props.provider
  testResult.value = null
  error.value = ''
  newModelId.value = ''
  if (provider) {
    form.name = provider.name
    form.kind = provider.kind
    form.base_url = provider.base_url
    form.api_key = ''
    form.timeout_sec = provider.timeout_sec
    form.max_concurrency = provider.max_concurrency
    form.enabled = provider.enabled
    form.is_default = provider.is_default
    form.models = provider.models.map((item) => ({
      model_id: item.model_id,
      display_name: item.display_name,
      description: item.description,
      context_window: item.context_window,
      max_concurrency: item.max_concurrency,
      enabled: item.enabled,
    }))
    mode.value = provider.is_builtin ? 'preset' : 'custom'
    presetCode.value = PRESETS.some((item) => item.kind === provider.kind) ? provider.kind : 'deepseek'
  } else {
    form.name = ''
    form.kind = 'custom'
    form.base_url = ''
    form.api_key = ''
    form.timeout_sec = 60
    form.max_concurrency = 4
    form.enabled = true
    form.is_default = false
    form.models = []
    mode.value = 'custom'
    presetCode.value = 'deepseek'
  }
}

watch(
  () => [open.value, props.provider?.id ?? 0],
  () => {
    if (open.value) reset()
  },
  { immediate: true },
)

/** 切换接入方式：仅在用户主动操作时套用模板，避免重置表单时覆盖已保存的地址。 */
function onModeChange(value: unknown): void {
  const next = value as 'preset' | 'custom'
  mode.value = next
  if (next === 'preset') applyPreset(presetCode.value)
  if (next === 'custom') form.kind = 'custom'
}

/** 添加模型 chip。 */
function addModel(): void {
  const id = newModelId.value.trim()
  if (!id) return
  if (form.models.some((item) => item.model_id === id)) {
    message.warning('该模型已存在')
    return
  }
  form.models.push({
    model_id: id,
    display_name: id,
    description: '',
    context_window: 128000,
    max_concurrency: 4,
    enabled: true,
  })
  newModelId.value = ''
}

/** 移除模型 chip。 */
function removeModel(id: string): void {
  form.models = form.models.filter((item) => item.model_id !== id)
}

/** 校验表单，返回可读错误。 */
function validate(): string {
  if (!form.name.trim()) return '请填写显示名称'
  if (!form.base_url.trim()) return '请填写 Base URL'
  if (!/^https?:\/\//i.test(form.base_url.trim())) return 'Base URL 必须以 http:// 或 https:// 开头'
  if (!isEditing.value && !form.api_key.trim()) return '新建厂商必须填写 API Key'
  if (form.timeout_sec < 5 || form.timeout_sec > 300) return '超时需在 5–300 秒之间'
  if (form.max_concurrency < 1 || form.max_concurrency > 10000) return '并发需在 1–10000 之间'
  return ''
}

/** 组装请求体。 */
function payload(): ProviderPayload {
  return {
    name: form.name.trim(),
    kind: form.kind,
    base_url: form.base_url.trim(),
    api_key: form.api_key.trim(),
    timeout_sec: form.timeout_sec,
    max_concurrency: form.max_concurrency,
    enabled: isEditing.value ? form.enabled : true,
    is_default: form.is_default,
    models: form.models.map((item) => ({
      model_id: item.model_id,
      display_name: item.display_name || item.model_id,
      description: item.description,
      context_window: item.context_window,
      max_concurrency: item.max_concurrency,
      enabled: item.enabled,
    })),
  }
}

/** 连接测试。 */
async function runTest(): Promise<void> {
  const reason = validate()
  if (reason) {
    error.value = reason
    return
  }
  error.value = ''
  testing.value = true
  testResult.value = null
  try {
    testResult.value = await testDraft(payload(), props.provider?.id)
  } catch (err) {
    testResult.value = {
      ok: false,
      latency_ms: 0,
      models_found: 0,
      models: [],
      message: err instanceof ApiError ? err.message : '测试失败，请稍后重试',
    }
  } finally {
    testing.value = false
  }
}

/** 保存厂商。 */
async function save(): Promise<void> {
  const reason = validate()
  if (reason) {
    error.value = reason
    return
  }
  if (testResult.value && !testResult.value.ok) {
    const proceed = await new Promise<boolean>((resolve) => {
      Modal.confirm({
        title: '连接测试未通过',
        content: '当前配置的连通性测试失败，仍要保存吗？保存后可随时重新测试。',
        okText: '仍然保存',
        cancelText: '返回修改',
        onOk: () => resolve(true),
        onCancel: () => resolve(false),
      })
    })
    if (!proceed) return
  }
  error.value = ''
  saving.value = true
  try {
    if (props.provider) {
      await updateProvider(props.provider.id, payload())
      message.success('厂商配置已保存')
    } else {
      await createProvider(payload())
      message.success('厂商已创建')
    }
    emit('saved')
    open.value = false
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : '保存失败，请稍后重试'
  } finally {
    saving.value = false
  }
}

/** 删除自定义厂商。 */
function removeProvider(): void {
  const provider = props.provider
  if (!provider) return
  Modal.confirm({
    title: `删除厂商「${provider.name}」？`,
    content: '该厂商下的模型会一并删除，使用这些模型的用户将无法继续选择。',
    okText: '删除',
    okType: 'danger',
    cancelText: '取消',
    async onOk() {
      await deleteProvider(provider.id)
      message.success('厂商已删除')
      emit('deleted')
      open.value = false
    },
  })
}

/** 关闭弹窗。 */
function close(): void {
  open.value = false
}
</script>

<template>
  <a-modal
    v-model:open="open"
    :title="isEditing ? '厂商配置' : '新建厂商'"
    :width="620"
    :footer="null"
    destroy-on-close
    @cancel="close"
  >
    <div class="provider-form">
      <div class="form-scroll">
        <div class="mode-row">
        <span class="mode-label">接入方式</span>
        <a-segmented :value="mode" :options="modeOptions" size="small" @change="onModeChange" />
      </div>

      <a-form layout="vertical" class="form-grid">
        <a-form-item v-if="mode === 'preset'" label="厂商模板">
          <a-select :value="presetCode" :options="presetOptions" @change="applyPreset" />
        </a-form-item>

        <div class="two-col">
          <a-form-item label="显示名称" required>
            <a-input v-model:value="form.name" placeholder="My Gateway" :maxlength="64" />
          </a-form-item>
          <a-form-item label="超时（秒）">
            <a-input-number v-model:value="form.timeout_sec" :min="5" :max="300" style="width: 100%" />
          </a-form-item>
        </div>

        <a-form-item label="Base URL" required>
          <a-input v-model:value="form.base_url" placeholder="https://gateway.example.com/v1" :maxlength="255" />
          <p class="field-hint">需兼容 /v1/chat/completions 与 /v1/models；生产环境必须 HTTPS。</p>
        </a-form-item>

        <a-form-item label="API Key">
          <a-input-password v-model:value="form.api_key" :placeholder="keyPlaceholder" autocomplete="new-password" />
          <p class="field-hint">Key 使用 AES-GCM 加密入库，前端永不回显；留空表示保留原密钥。</p>
        </a-form-item>

        <a-form-item label="模型列表">
          <div class="chips">
            <span v-for="item in form.models" :key="item.model_id" class="chip">
              {{ item.model_id }}
              <button type="button" class="chip-remove" :aria-label="`移除 ${item.model_id}`" @click="removeModel(item.model_id)">
                ×
              </button>
            </span>
            <span v-if="!form.models.length" class="chip-empty">尚未添加模型</span>
          </div>
          <div class="add-row">
            <a-input
              v-model:value="newModelId"
              size="small"
              placeholder="输入模型 ID，如 gpt-4.1"
              @press-enter="addModel"
            />
            <a-button size="small" @click="addModel">
              <PlusOutlined />
              添加模型
            </a-button>
          </div>
          <p class="field-hint">仅启用的模型会出现在用户侧；上下文与并发可在模型路由表继续编辑。</p>
        </a-form-item>

        <div class="two-col">
          <a-form-item label="最大并发">
            <a-input-number v-model:value="form.max_concurrency" :min="1" :max="10000" style="width: 100%" />
          </a-form-item>
          <div class="switch-field">
            <div>
              <span class="switch-label">启用该厂商</span>
              <p class="field-hint">保存后立即参与模型路由。</p>
            </div>
            <a-switch v-model:checked="form.enabled" />
          </div>
        </div>

        <div class="switch-field default-row">
          <div>
            <span class="switch-label">设为平台默认</span>
            <p class="field-hint">新用户默认使用该厂商的默认模型。</p>
          </div>
          <a-switch v-model:checked="form.is_default" />
        </div>

        <div class="test-box" :class="{ ok: testResult?.ok, fail: testResult && !testResult.ok }">
          <div class="test-head">
            <div>
              <strong>连接测试</strong>
              <p>由服务端发起，10 秒超时。</p>
            </div>
            <a-button size="small" :loading="testing" @click="runTest">测试连接</a-button>
          </div>
          <div v-if="testResult" class="test-result">
            <span class="test-dot" :class="{ off: !testResult.ok }" />
            <span class="test-message">{{ testResult.message }}</span>
            <span v-if="testResult.ok" class="test-meta">
              {{ testResult.latency_ms }} ms · {{ testResult.models_found }} 个模型
            </span>
          </div>
          <p v-else class="field-hint">保存前建议先测试 /models；失败不会覆盖现有配置。</p>
        </div>

        <div v-if="error" class="form-error" role="alert">{{ error }}</div>
      </a-form>
      </div>

      <div class="modal-footer">
        <a-button v-if="canDelete" type="text" danger class="delete-btn" @click="removeProvider">
          <DeleteOutlined />
          删除厂商
        </a-button>
        <span class="footer-spacer" />
        <a-button @click="close">取消</a-button>
        <a-button type="primary" :loading="saving" @click="save">
          {{ isEditing ? '保存修改' : '保存并启用' }}
        </a-button>
      </div>
    </div>
  </a-modal>
</template>

<style scoped>
.provider-form {
  display: flex;
  flex-direction: column;
  max-height: calc(100vh - 190px);
  padding-top: 4px;
}

.form-scroll {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding-right: 4px;
}

.mode-row {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}

.mode-label {
  color: var(--dt-text-secondary);
  font-size: 11.6px;
  font-weight: 600;
  white-space: nowrap;
}

.form-grid :deep(.ant-form-item) {
  margin-bottom: 13px;
}

.form-grid :deep(.ant-form-item-label) {
  padding-bottom: 5px;
}

.form-grid :deep(.ant-form-item-label > label) {
  color: var(--dt-text-secondary);
  font-size: 11.4px;
  font-weight: 600;
}

.two-col {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14px;
  align-items: start;
}

.field-hint {
  margin: 4px 0 0;
  color: var(--dt-text-faint);
  font-size: 9.8px;
  line-height: 15px;
}

.chips {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
  min-height: 30px;
  padding: 6px 8px;
  border: 1px solid var(--dt-line);
  border-radius: 9px;
  background: #fafbfd;
}

.chip {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  height: 22px;
  padding: 0 7px;
  border: 1px solid #e2e7f3;
  border-radius: 6px;
  background: #fff;
  color: var(--dt-text-secondary);
  font-family: var(--dt-mono);
  font-size: 9.8px;
}

.chip-remove {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 14px;
  height: 14px;
  border: none;
  border-radius: 3px;
  background: transparent;
  color: var(--dt-text-faint);
  font-size: 13px;
  line-height: 1;
  cursor: pointer;
}

.chip-remove:hover {
  background: var(--dt-danger-soft);
  color: var(--dt-danger-text);
}

.chip-empty {
  color: var(--dt-text-faint);
  font-size: 10.4px;
}

.add-row {
  display: flex;
  gap: 8px;
  margin-top: 7px;
}

.add-row :deep(.ant-input) {
  font-size: 11.4px;
}

.switch-field {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  padding: 9px 12px;
  border: 1px solid var(--dt-line);
  border-radius: 9px;
}

.default-row {
  margin-bottom: 13px;
}

.switch-label {
  color: var(--dt-text-secondary);
  font-size: 11.4px;
  font-weight: 600;
}

.test-box {
  padding: 11px 12px;
  border: 1px solid var(--dt-line);
  border-radius: 10px;
  background: #fafbfd;
}

.test-box.ok {
  border-color: #d1fadf;
  background: var(--dt-success-soft);
}

.test-box.fail {
  border-color: #fee4e2;
  background: var(--dt-danger-soft);
}

.test-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.test-head strong {
  color: var(--dt-text-secondary);
  font-size: 11.4px;
  font-weight: 650;
}

.test-head p {
  margin: 2px 0 0;
  color: var(--dt-text-faint);
  font-size: 9.6px;
}

.test-result {
  display: flex;
  align-items: center;
  gap: 7px;
  margin-top: 8px;
  flex-wrap: wrap;
}

.test-dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: var(--dt-success);
}

.test-dot.off {
  background: var(--dt-danger);
}

.test-message {
  color: var(--dt-text-secondary);
  font-size: 10.6px;
}

.test-meta {
  color: var(--dt-text-muted);
  font-family: var(--dt-mono);
  font-size: 9.6px;
}

.form-error {
  margin-bottom: 4px;
  padding: 8px 10px;
  border-radius: 8px;
  background: var(--dt-danger-soft);
  color: var(--dt-danger-text);
  font-size: 11px;
}

.modal-footer {
  display: flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 8px;
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--dt-line-soft);
}

.footer-spacer {
  flex: 1;
}

.delete-btn {
  color: var(--dt-danger-text);
}

@media (max-width: 640px) {
  .two-col {
    grid-template-columns: 1fr;
    gap: 0;
  }
}
</style>
