<script setup lang="ts">
/**
 * A2UI 渲染器：把 agent 产出的 v0.9 A2UI 消息交给官方 Lit Renderer。
 * 只在消息确实包含 A2UI 时才动态加载渲染器，避免普通对话承担额外包体。
 */
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import type { MessageProcessor, SurfaceModel } from '@a2ui/web_core/v0_9'
import type { A2UIMessage } from '@deeptalk/shared'

const props = defineProps<{ messages: A2UIMessage[] }>()

const host = ref<HTMLElement | null>(null)
const error = ref('')
let processor: MessageProcessor | null = null
let processed = 0
let disposed = false

interface SurfaceElement extends HTMLElement {
  surface: SurfaceModel
}

/** 将新增的 A2UI envelope 交给 MessageProcessor。 */
function processPending(): void {
  if (!processor || processed >= props.messages.length) return
  const pending = props.messages.slice(processed)
  try {
    processor.processMessages(pending)
    processed = props.messages.length
    error.value = ''
  } catch (reason) {
    error.value = reason instanceof Error ? reason.message : 'A2UI 渲染失败'
  }
}

onMounted(async () => {
  if (!props.messages.length || !host.value) return
  try {
    const [{ MessageProcessor }, { basicCatalog }] = await Promise.all([
      import('@a2ui/web_core/v0_9'),
      import('@a2ui/lit/v0_9'),
    ])
    if (disposed || !host.value) return
    processor = new MessageProcessor([basicCatalog])
    processor.onSurfaceCreated((surface) => {
      const element = document.createElement('a2ui-surface') as SurfaceElement
      element.className = 'a2ui-surface'
      element.dataset.surfaceId = surface.id
      element.surface = surface
      host.value?.appendChild(element)
    })
    processor.onSurfaceDeleted((surfaceId) => {
      const element = host.value?.querySelector<SurfaceElement>(`[data-surface-id="${surfaceId}"]`)
      element?.remove()
    })
    processPending()
  } catch (reason) {
    error.value = reason instanceof Error ? reason.message : 'A2UI 渲染器加载失败'
  }
})

watch(
  () => props.messages,
  () => processPending(),
  { deep: true },
)

onBeforeUnmount(() => {
  disposed = true
  processor?.dispose()
  processor = null
  host.value?.replaceChildren()
})
</script>

<template>
  <div class="a2ui-wrap">
    <div ref="host" class="a2ui-host" />
    <p v-if="error" class="a2ui-error">界面暂时无法渲染：{{ error }}</p>
  </div>
</template>

<style scoped>
.a2ui-wrap {
  width: 100%;
  margin: 10px 0 14px;
}

.a2ui-host {
  display: block;
  width: 100%;
  overflow-x: auto;
  border: 1px solid var(--dt-line);
  border-radius: 12px;
  background: var(--dt-surface);
  padding: 16px 18px;
}

.a2ui-host :deep(a2ui-surface) {
  display: block;
  width: 100%;
}

.a2ui-error {
  margin: 8px 0 0;
  color: var(--dt-danger-text);
  font-size: 11.4px;
}
</style>
