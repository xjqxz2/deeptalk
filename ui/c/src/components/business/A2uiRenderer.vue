<script setup lang="ts">
/**
 * A2UI 渲染器：把 agent 产出的 v0.9 A2UI 消息交给官方 Lit Renderer。
 * 只在消息确实包含 A2UI 时才动态加载渲染器，避免普通对话承担额外包体。
 */
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { message } from 'ant-design-vue'
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

/**
 * A2UI 组件库自带的表格样式只考虑了浅色主题：深色下表头是白底白字。
 * 这里在组件挂载后补一份主题样式（Web Component 用 shadowRoot，普通节点用全局类）。
 */
function applySurfaceTheme(element: HTMLElement): void {
  const css = `
    :host, a2ui-surface, * { color: var(--dt-text, #101828); }
    table { border-collapse: collapse; width: 100%; margin: 8px 0; font-size: 12.5px; }
    th, td { border: 1px solid var(--dt-line, #eaecf0); padding: 6px 8px; text-align: left; }
    th { background: var(--dt-hover, #f2f4f7); color: var(--dt-text-secondary, #475467); font-weight: 600; }
    tr:nth-child(even) td { background: color-mix(in srgb, var(--dt-hover, #f2f4f7) 45%, transparent); }
  `
  const root = (element as HTMLElement & { shadowRoot?: ShadowRoot | null }).shadowRoot
  if (root) {
    const style = document.createElement('style')
    style.textContent = css
    root.appendChild(style)
    return
  }
  element.setAttribute('data-dt-theme', 'on')
}

/**
 * A2UI 按钮点击：surface 内部由 @a2ui/lit 渲染，这里统一接管点击，
 * 至少给出可见反馈，避免「点了没反应」。
 */
function onHostClick(event: MouseEvent): void {
  const target = (event.target as HTMLElement | null)?.closest('button, a, [role="button"]')
  if (!target) return
  const label = (target.textContent || '').trim() || '操作'
  message.info(`已执行：${label}`)
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
      applySurfaceTheme(element)
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
    <div ref="host" class="a2ui-host" @click="onHostClick" />
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

/* A2UI 没有 shadowRoot 时的兜底：表格跟随主题 */
.a2ui-host[data-dt-theme] table,
.a2ui-surface table {
  border-collapse: collapse;
  width: 100%;
  margin: 8px 0;
  font-size: 12.5px;
}

.a2ui-host[data-dt-theme] th,
.a2ui-host[data-dt-theme] td,
.a2ui-surface th,
.a2ui-surface td {
  border: 1px solid var(--dt-line);
  padding: 6px 8px;
  text-align: left;
  color: var(--dt-text);
}

.a2ui-host[data-dt-theme] th,
.a2ui-surface th {
  background: var(--dt-hover);
  color: var(--dt-text-secondary);
  font-weight: 600;
}
</style>
