<script setup lang="ts">
/**
 * 输入区：1–8 行自适应，Enter 发送 / Shift+Enter 换行。
 * 生成中切换为“停止生成”，并保留用户已输入的内容。
 */
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import type { Attachment } from '@deeptalk/shared'
import ModelSelect from './ModelSelect.vue'
import { ArrowUpOutlined, CloseOutlined, LoadingOutlined, PaperClipOutlined } from '@ant-design/icons-vue'

const props = withDefaults(
  defineProps<{
    text: string
    attachments?: Attachment[]
    uploading?: boolean
    generating: boolean
    disabled?: boolean
    placeholder?: string
    autofocus?: boolean
  }>(),
  {
    disabled: false,
    attachments: () => [],
    uploading: false,
    placeholder: '给 小橘 发送消息…',
    autofocus: false,
  },
)

const emit = defineEmits<{
  (event: 'update:text', value: string): void
  (event: 'add-files', files: File[]): void
  (event: 'remove-attachment', id: number): void
  (event: 'send'): void
  (event: 'stop'): void
}>()

/** textarea 引用，用于自适应高度与聚焦。 */
const textarea = ref<HTMLTextAreaElement | null>(null)
/** 文件选择框引用。 */
const fileInput = ref<HTMLInputElement | null>(null)
/** 中文输入法组合态，避免候选词回车被当成发送。 */
const composing = ref(false)

const canSend = computed(
  () =>
    (props.text.trim().length > 0 || (props.attachments?.length ?? 0) > 0) &&
    !props.uploading &&
    !props.disabled,
)

/** 生成中按发送是"插话"，会等这一轮结束再处理。 */
const sendLabel = computed(() => (props.generating ? '插话' : '发送消息'))

/** 自适应高度：最小一行，最多 8 行（约 192px）。 */
function resize(): void {
  const element = textarea.value
  if (!element) return
  // 移动端首帧可能宽度为 0；此时按宽度 0 计算 scrollHeight 会把占位文案折成多行，
  // 留下一个异常高的输入框。先给最小高度，等真实布局完成后再计算。
  if (element.clientWidth === 0) {
    element.style.height = '52px'
    return
  }
  element.style.height = 'auto'
  element.style.height = `${Math.min(element.scrollHeight, 192)}px`
}

/** 把输入框聚焦到末尾。 */
function focus(): void {
  nextTick(() => {
    textarea.value?.focus()
    const length = textarea.value?.value.length ?? 0
    textarea.value?.setSelectionRange(length, length)
  })
}

/** 输入时同步到父组件并重算高度。 */
function onInput(event: Event): void {
  const value = (event.target as HTMLTextAreaElement).value
  emit('update:text', value)
  resize()
}

/** 打开系统文件选择框。 */
function pickFiles(): void {
  fileInput.value?.click()
}

/** 文件选择后交给页面/Store 上传，并清空 input 以便重复选择同一文件。 */
function onFileChange(event: Event): void {
  const input = event.target as HTMLInputElement
  const files = Array.from(input.files ?? [])
  input.value = ''
  if (files.length) emit('add-files', files)
}

/** 粘贴图片：只拦截图片，文本粘贴仍走浏览器默认行为。 */
function onPaste(event: ClipboardEvent): void {
  const files = Array.from(event.clipboardData?.files ?? []).filter((file) => file.type.startsWith('image/'))
  if (!files.length) return
  event.preventDefault()
  emit('add-files', files)
}

/** 键盘规则：Enter 发送、Shift+Enter 换行、输入法组合态回车不发送。 */
function onKeydown(event: KeyboardEvent): void {
  if (event.key !== 'Enter') return
  if (event.shiftKey) return
  if (composing.value || event.isComposing) return
  event.preventDefault()
  if (!canSend.value) return
  emit('send')
}

/** 点击发送前再次校验空内容。 */
function submit(): void {
  if (!canSend.value) return
  emit('send')
}

watch(
  () => props.text,
  () => {
    nextTick(resize)
  },
)

function onViewportResize(): void {
  resize()
}

onMounted(() => {
  // 等浏览器完成首帧布局后再量高度；iOS Safari 首帧宽度经常是 0
  requestAnimationFrame(() => {
    resize()
    if (props.autofocus) focus()
  })
  window.addEventListener('resize', onViewportResize)
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', onViewportResize)
})

defineExpose({ focus })
</script>

<template>
  <div class="composer" :class="{ generating }">
    <div class="composer-box">
      <div v-if="(attachments?.length ?? 0) > 0 || uploading" class="attachment-strip">
        <div v-for="item in attachments" :key="item.id" class="attachment-item">
          <img :src="item.url" :alt="item.name" />
          <button
            type="button"
            class="attachment-remove"
            :aria-label="`移除图片 ${item.name}`"
            @click="emit('remove-attachment', item.id)"
          >
            <CloseOutlined />
          </button>
        </div>
        <div v-if="uploading" class="attachment-uploading">
          <LoadingOutlined spin />
          <span>上传中</span>
        </div>
      </div>
      <input
        ref="fileInput"
        class="file-input"
        type="file"
        accept="image/png,image/jpeg,image/gif,image/webp"
        multiple
        @change="onFileChange"
      />
      <textarea
        ref="textarea"
        class="composer-input"
        :value="text"
        rows="1"
        :placeholder="placeholder"
        :disabled="disabled"
        aria-label="消息输入框"
        @input="onInput"
        @keydown="onKeydown"
        @paste="onPaste"
        @compositionstart="composing = true"
        @compositionend="composing = false"
      />
      <div class="composer-toolbar">
        <div class="toolbar-left">
          <button
            class="icon-btn"
            type="button"
            :disabled="disabled || uploading"
            title="上传图片，或直接粘贴图片"
            aria-label="上传图片"
            @click="pickFiles"
          >
            <PaperClipOutlined />
          </button>
        </div>
        <div class="toolbar-right">
          <ModelSelect />
          <button v-if="generating" type="button" class="stop-btn" @click="emit('stop')">
            <span class="stop-square" aria-hidden="true" />
            停止生成
          </button>
          <button
            type="button"
            class="send-btn"
            :class="{ 'send-btn-interject': generating }"
            :disabled="!canSend && !generating"
            :aria-label="sendLabel"
            @click="submit"
          >
            <ArrowUpOutlined />
          </button>
        </div>
      </div>
    </div>
    <p class="disclaimer">内容由 AI 生成，请仔细甄别</p>
  </div>
</template>

<style scoped>
.composer {
  width: 100%;
}

.file-input {
  display: none;
}

.attachment-strip {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  padding: 4px 2px 10px;
}

.attachment-item {
  position: relative;
  width: 58px;
  height: 58px;
  border: 1px solid var(--dt-line);
  border-radius: 9px;
  overflow: hidden;
  background: var(--dt-hover);
}

.attachment-item img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.attachment-remove {
  position: absolute;
  top: 2px;
  right: 2px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 17px;
  height: 17px;
  border: 0;
  border-radius: 50%;
  background: rgba(16, 24, 40, 0.66);
  color: #fff;
  font-size: 9px;
  cursor: pointer;
}

.attachment-uploading {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 58px;
  padding: 0 12px;
  border: 1px dashed var(--dt-line);
  border-radius: 9px;
  color: var(--dt-text-faint);
  font-size: 11px;
}

.composer-box {
  border: 1px solid var(--dt-line);
  border-radius: 14px;
  background: var(--dt-surface);
  box-shadow: 0 6px 18px rgba(16, 24, 40, 0.05);
  padding: 14px 16px 12px;
  transition: border-color 0.16s ease, box-shadow 0.16s ease;
}

.composer-box:focus-within {
  border-color: var(--dt-focus-ring);
  box-shadow: 0 0 0 2px rgba(201, 212, 255, 0.55), 0 8px 20px rgba(16, 24, 40, 0.06);
}

.composer-input {
  display: block;
  width: 100%;
  min-height: 52px;
  max-height: 192px;
  border: none;
  outline: none;
  resize: none;
  background: transparent;
  color: var(--dt-text);
  font-family: inherit;
  font-size: 13.6px;
  line-height: 24px;
  padding: 0;
}

.composer-input::placeholder {
  color: var(--dt-text-faint);
}

.composer-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-top: 8px;
}

.toolbar-left,
.toolbar-right {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: var(--dt-text-muted);
  font-size: 16px;
  cursor: pointer;
}

.send-btn-interject {
  background: var(--dt-brand-soft);
  color: var(--dt-brand);
}

.icon-btn:disabled {
  color: var(--dt-text-disabled);
  cursor: not-allowed;
}

.send-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  border: none;
  border-radius: 50%;
  background: var(--dt-brand);
  color: #fff;
  font-size: 15px;
  cursor: pointer;
  transition: background 0.16s ease, transform 0.16s ease;
}

.send-btn:hover:not(:disabled) {
  background: var(--dt-brand-hover);
  transform: translateY(-1px);
}

.send-btn:disabled {
  background: #dbe2f8;
  color: #fff;
  cursor: not-allowed;
}

.stop-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 32px;
  padding: 0 12px;
  border: 1px solid #e1e5ee;
  border-radius: 9px;
  background: var(--dt-surface);
  color: var(--dt-text-secondary);
  font-size: 11.8px;
  font-weight: 600;
  cursor: pointer;
}

.stop-btn:hover {
  border-color: var(--dt-focus-ring);
  color: var(--dt-brand-hover);
}

.stop-square {
  width: 9px;
  height: 9px;
  border-radius: 2px;
  background: currentColor;
}

.disclaimer {
  margin: 10px 0 0;
  color: var(--dt-text-faint);
  font-size: 10.2px;
  text-align: center;
}

@media (max-width: 768px) {
  /* iOS Safari 在输入框字号小于 16px 时会自动放大页面；保持 16px 避免聚焦后整页变大 */
  .composer-input {
    min-height: 52px;
    font-size: 16px;
    line-height: 26px;
  }
}

@media (max-width: 640px) {
  .composer-box {
    padding: 12px 12px 10px;
  }

  .model-select {
    width: 126px;
  }
}
</style>
