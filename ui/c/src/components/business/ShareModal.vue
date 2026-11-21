<script setup lang="ts">
/**
 * 分享对话弹窗：创建链接、复制链接、撤销分享。
 * 分享是只读快照，访客不能继续对话；撤销后旧链接立即失效。
 */
import { computed, ref, watch } from 'vue'
import { message } from 'ant-design-vue'
import { CopyOutlined, LinkOutlined } from '@ant-design/icons-vue'
import { copyText } from '@deeptalk/shared'
import type { ShareInfo } from '@deeptalk/shared'
import * as chatApi from '@/services/chat'

const open = defineModel<boolean>('open', { default: false })
const props = defineProps<{ conversationId: number | null; title: string }>()

const share = ref<ShareInfo | null>(null)
const loading = ref(false)
const creating = ref(false)
const revoking = ref(false)
const copied = ref(false)

/** 分享链接使用当前站点地址，避免硬编码域名。 */
const shareURL = computed(() => {
  if (!share.value) return ''
  if (typeof window === 'undefined') return `/s/${share.value.token}`
  return `${window.location.origin}/s/${share.value.token}`
})

/** 打开弹窗时读取现有分享。 */
async function load(): Promise<void> {
  if (props.conversationId === null) return
  loading.value = true
  try {
    const data = await chatApi.getShare(props.conversationId)
    share.value = data.share
  } catch (error) {
    message.error(error instanceof Error ? error.message : '读取分享失败')
  } finally {
    loading.value = false
  }
}

/** 创建分享并自动复制。 */
async function create(): Promise<void> {
  if (props.conversationId === null) return
  creating.value = true
  try {
    const data = await chatApi.createShare(props.conversationId)
    share.value = data.share
    await copy()
  } catch (error) {
    message.error(error instanceof Error ? error.message : '创建分享失败')
  } finally {
    creating.value = false
  }
}

/** 复制链接。 */
async function copy(): Promise<void> {
  if (!shareURL.value) return
  const ok = await copyText(shareURL.value)
  if (!ok) {
    message.warning('复制失败，请手动选择链接')
    return
  }
  copied.value = true
  message.success('链接已复制')
  window.setTimeout(() => {
    copied.value = false
  }, 1600)
}

/** 撤销分享。 */
async function revoke(): Promise<void> {
  if (props.conversationId === null) return
  revoking.value = true
  try {
    await chatApi.revokeShare(props.conversationId)
    share.value = null
    message.success('分享已取消，旧链接立即失效')
  } catch (error) {
    message.error(error instanceof Error ? error.message : '取消分享失败')
  } finally {
    revoking.value = false
  }
}

watch(open, (value) => {
  if (!value) return
  share.value = null
  copied.value = false
  void load()
})
</script>

<template>
  <a-modal v-model:open="open" :footer="null" :width="440" centered title="分享对话">
    <div class="share-body">
      <p class="share-lead">
        分享链接是只读的，获得链接的人可以查看这段对话，但不能继续提问或回复。
      </p>

      <a-spin :spinning="loading">
        <template v-if="share">
          <div class="share-link">
            <LinkOutlined class="share-link-icon" />
            <input class="share-input" :value="shareURL" readonly aria-label="分享链接" @focus="($event.target as HTMLInputElement).select()" />
            <a-button size="small" @click="copy">
              <template #icon><CopyOutlined /></template>
              {{ copied ? '已复制' : '复制' }}
            </a-button>
          </div>
          <div class="share-actions">
            <a-button type="text" size="small" danger :loading="revoking" @click="revoke">取消分享</a-button>
            <span class="share-tip">取消后该链接立即失效</span>
          </div>
        </template>

        <template v-else-if="!loading">
          <div class="share-empty">
            <p class="share-empty-title">{{ title }}</p>
            <a-button type="primary" :loading="creating" @click="create">创建分享链接</a-button>
            <p class="share-empty-tip">创建后，任何拿到链接的人都能只读查看这段对话。</p>
          </div>
        </template>
      </a-spin>
    </div>
  </a-modal>
</template>

<style scoped>
.share-body {
  padding-top: 2px;
  min-height: 96px;
}

.share-lead {
  margin: 0 0 14px;
  color: var(--dt-text-secondary);
  font-size: 12.4px;
  line-height: 20px;
}

.share-link {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 8px 6px 10px;
  border: 1px solid var(--dt-line);
  border-radius: 10px;
  background: var(--dt-bg);
}

.share-link-icon {
  color: var(--dt-text-faint);
  font-size: 13px;
}

.share-input {
  flex: 1;
  min-width: 0;
  border: none;
  background: transparent;
  color: var(--dt-text);
  font-family: var(--dt-mono);
  font-size: 11.6px;
  outline: none;
}

.share-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-top: 8px;
}

.share-tip {
  color: var(--dt-text-faint);
  font-size: 10.8px;
}

.share-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  padding: 14px 0 6px;
  text-align: center;
}

.share-empty-title {
  margin: 0;
  color: var(--dt-text);
  font-size: 13.2px;
  font-weight: 620;
  line-height: 20px;
}

.share-empty-tip {
  margin: 0;
  color: var(--dt-text-faint);
  font-size: 11px;
  line-height: 17px;
}
</style>
