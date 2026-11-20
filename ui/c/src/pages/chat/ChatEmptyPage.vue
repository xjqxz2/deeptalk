<script setup lang="ts">
/**
 * 空态工作台：品牌问候 + 输入区。
 * 首条消息发送时创建会话，再跳到会话页由同一条 SSE 流继续渲染。
 */
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { BrandMark } from '@deeptalk/shared'
import { useChatStore } from '@/stores/chat'
import { useAppStore } from '@/stores/app'
import Composer from '@/components/business/Composer.vue'

const app = useAppStore()
const chat = useChatStore()
const router = useRouter()

/** 输入内容。 */
const prompt = ref('')

/** 选择/粘贴图片后上传；失败时保留已上传的图片并提示。 */
async function handleAddFiles(files: File[]): Promise<void> {
  try {
    await chat.uploadFiles(files)
  } catch (error) {
    message.error(error instanceof Error ? error.message : '图片上传失败')
  }
}

/** 发送首条消息：先创建会话，再跳转并把内容交给会话页发送。 */
async function handleSend(): Promise<void> {
  const text = prompt.value.trim()
  if (!text && chat.attachments.length === 0) return
  if (!chat.models.length) {
    message.warning('当前没有可用模型，请联系管理员')
    return
  }
  if (chat.generating) return
  try {
    const conversation = await chat.createConversation()
    chat.queuePrompt(text)
    prompt.value = ''
    await router.push({ name: 'chat-session', params: { id: conversation.public_id || conversation.id } })
  } catch (error) {
    message.error(error instanceof Error ? error.message : '创建对话失败，请稍后重试')
  }
}

onMounted(() => {
  // 回到空态时清掉上一条草稿，避免顶部标题与内容错位
  chat.startNewDraft()
})
</script>

<template>
  <div class="chat-empty">
    <div class="empty-scroll">
      <div class="empty-hero dt-fade-up">
        <div class="hero-mark">
          <BrandMark :size="52" :radius="15" />
        </div>
        <h1>我是 DeepTalk，很高兴见到你！</h1>
      </div>

      <div class="empty-composer">
        <Composer
          v-model:text="prompt"
          :attachments="chat.attachments"
          :uploading="chat.uploading"
          :generating="chat.generating"
          :autofocus="!app.isMobile"
          @add-files="handleAddFiles"
          @remove-attachment="chat.removeAttachment"
          @send="handleSend"
        />
      </div>
    </div>
  </div>
</template>

<style scoped>
.chat-empty {
  height: 100%;
  overflow-y: auto;
}

.empty-scroll {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 42px;
  min-height: 100%;
  padding: 48px 40px calc(112px + env(safe-area-inset-bottom));
}

.empty-hero {
  display: flex;
  flex-direction: column;
  align-items: center;
  width: 100%;
  max-width: var(--dt-chat-max, 752px);
}

.hero-mark {
  display: inline-flex;
  margin-bottom: 24px;
}

.empty-hero h1 {
  margin: 0;
  color: var(--dt-text);
  font-size: 30px;
  line-height: 1.35;
  font-weight: 730;
  letter-spacing: -0.5px;
  text-align: center;
}

.empty-composer {
  width: 100%;
  max-width: var(--dt-composer-max, 752px);
}

@media (max-width: 900px) {
  .empty-scroll {
    justify-content: flex-start;
    padding: 40px 24px calc(24px + env(safe-area-inset-bottom));
    gap: 30px;
  }
}

@media (max-width: 600px) {
  .empty-scroll {
    padding: 28px 16px calc(16px + env(safe-area-inset-bottom));
  }

  .empty-hero h1 {
    font-size: 24px;
  }
}
</style>
