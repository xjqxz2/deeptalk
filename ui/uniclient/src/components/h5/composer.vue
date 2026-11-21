<template>
  <div class="composer" :class="{ generating }">
    <div class="composer-box">
      <div v-if="attachments.length" class="attachment-strip">
        <div v-for="item in attachments" :key="item.id" class="attachment-item">
          <img :src="item.url" :alt="item.name" />
          <Btn type="button" class="attachment-remove" aria-label="移除图片" @click="emit('remove-attachment', item.id)">
            <CloseOutlined />
          </Btn>
        </div>
      </div>

      <NativeTextarea
        ref="textareaRef"
        :model-value="modelValue"
        :placeholder="generating ? '生成中可以插话，会在这一轮结束后处理…' : '继续追问，或让 小橘树洞 补充更多细节…'"
        @update:model-value="(value) => emit('update:modelValue', value)"
        @submit="submit"
      />

      <div class="composer-toolbar">
        <div class="toolbar-left">
          <Btn
            class="icon-btn"
            type="button"
            title="上传图片"
            aria-label="上传图片"
            @click="emit('attach')"
          >
            <PaperClipOutlined />
          </Btn>
        </div>
        <div class="toolbar-right">
          <Btn type="button" class="model-trigger" @click="emit('pick-model')">
            {{ modelName }}
            <DownOutlined class="model-arrow" />
          </Btn>
          <Btn v-if="generating" type="button" class="stop-btn" @click="emit('stop')">
            <span class="stop-square" aria-hidden="true" />
            停止生成
          </Btn>
          <Btn
            v-else
            type="button"
            class="send-btn"
            :disabled="!canSend"
            aria-label="发送消息"
            @click="submit"
          >
            <ArrowUpOutlined />
          </Btn>
        </div>
      </div>
    </div>
    <p class="disclaimer">内容由 AI 生成，请仔细甄别</p>
  </div>
</template>

<script setup>
/** H5 输入区：结构与 class 名与 PWA 的 Composer 一致。 */
import { computed, ref } from 'vue'
import { ArrowUpOutlined, CloseOutlined, DownOutlined, PaperClipOutlined } from '@ant-design/icons-vue'
import NativeTextarea from './textarea.vue'
import Btn from './btn.vue'

const props = defineProps({
  modelValue: { type: String, default: '' },
  generating: { type: Boolean, default: false },
  modelName: { type: String, default: '选择模型' },
  attachments: { type: Array, default: () => [] },
})
const emit = defineEmits(['update:modelValue', 'send', 'stop', 'attach', 'pick-model', 'remove-attachment'])

const textareaRef = ref(null)
const canSend = computed(() => Boolean(props.modelValue && props.modelValue.trim()))

function submit(value) {
  const text = typeof value === 'string' ? value : props.modelValue
  if (!text || !text.trim()) return
  emit('send', text)
}

/** 发送后清空输入框：props 变化由父组件驱动，这里同步一次 DOM。 */
defineExpose({
  clear() {
    if (textareaRef.value) textareaRef.value.value = ''
  },
})
</script>
