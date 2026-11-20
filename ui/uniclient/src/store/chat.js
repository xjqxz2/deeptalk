/**
 * 聊天状态：会话列表、当前会话消息、流式生成状态。
 * 生成走 WebSocket，事件与后端 SSE 事件名保持一致。
 */
import { reactive, computed } from 'vue'
import { chatSocket } from '@/utils/ws'
import * as chatApi from '@/api/chat'

let currentKey = 0
let generationTimer = null

/** 生成开始：启动看门狗，超时自动解锁，避免输入框被永久禁用。 */
function startGenerationWatchdog() {
  clearTimeout(generationTimer)
  generationTimer = setTimeout(() => {
    if (!chat.generating) return
    const assistant = streamingMessage()
    if (assistant) {
      assistant.status = 'error'
      assistant.error = '生成超时，请重试'
    }
    chat.generating = false
    chat.activity = null
  }, 90000)
}

/** 生成结束：清掉看门狗。 */
function stopGenerationWatchdog() {
  clearTimeout(generationTimer)
  generationTimer = null
}

export const chat = reactive({
  conversations: [],
  current: null,
  messages: [],
  models: [],
  modelId: 0,
  loading: false,
  generating: false,
  activity: null, // { status, label, detail }
  usage: null, // 本月用量卡片数据
  connection: 'idle',
})

/** 当前会话的展示标题。 */
export const currentTitle = computed(() => chat.current?.title || '新对话')

/** 拉取会话列表。 */
export async function loadConversations(keyword = '') {
  const data = await chatApi.listConversations(keyword)
  chat.conversations = data.items || []
  return chat.conversations
}

/** 拉取用量（底部卡片）。 */
export async function loadUsage() {
  const { fetchUsage } = await import('@/api/auth')
  try {
    chat.usage = await fetchUsage()
  } catch {
    chat.usage = null
  }
  return chat.usage
}

/** 拉取模型列表，恢复上次选择。 */
export async function loadModels() {
  const data = await chatApi.listModels()
  chat.models = data.items || []
  const saved = Number(uni.getStorageSync('dt_model_id') || 0)
  const exists = chat.models.some((item) => item.id === saved)
  chat.modelId = exists ? saved : chat.models[0]?.id || 0
  return chat.models
}

/** 切换模型。 */
export function selectModel(modelId) {
  chat.modelId = modelId
  uni.setStorageSync('dt_model_id', modelId)
}

/** 打开会话，加载消息。 */
export async function openConversation(id) {
  chat.loading = true
  const key = ++currentKey
  try {
    const data = await chatApi.getConversation(id)
    if (key !== currentKey) return
    chat.current = data.conversation
    chat.messages = (data.messages || []).map(normalizeMessage)
  } finally {
    if (key === currentKey) chat.loading = false
  }
}

/** 新建会话（本地先占位，真正创建放到首次发送）。 */
export function startDraft() {
  currentKey += 1
  chat.current = null
  chat.messages = []
  chat.activity = null
}

/** 把服务端消息补上客户端渲染需要的字段。 */
function normalizeMessage(item) {
  return {
    ...item,
    reasoning: item.reasoning || '',
    sources: item.sources || [],
    attachments: item.attachments || [],
    a2ui: item.a2ui || [],
  }
}

/** 确保存在会话；没有就创建。 */
async function ensureConversation() {
  if (chat.current?.public_id) return chat.current
  const data = await chatApi.createConversation()
  chat.current = data.conversation
  await loadConversations()
  return chat.current
}

/** 追加一条本地消息。 */
function pushMessage(message) {
  chat.messages.push(message)
  return chat.messages[chat.messages.length - 1]
}

/** 找到正在生成的助手消息。 */
function streamingMessage() {
  const last = chat.messages[chat.messages.length - 1]
  return last && last.role === 'assistant' && last.status === 'streaming' ? last : null
}

// ---- WebSocket 事件 -------------------------------------------------------

chatSocket.on('meta', (payload) => {
  const assistant = streamingMessage()
  if (!assistant) return
  assistant.id = payload.assistant_message_id || assistant.id
  assistant.model_name = payload.model_name || assistant.model_name
  if (payload.title) applyTitle(payload.title)
})

chatSocket.on('reasoning', (payload) => {
  const assistant = streamingMessage()
  if (!assistant) return
  assistant.reasoning += payload.delta || ''
})

chatSocket.on('delta', (payload) => {
  const assistant = streamingMessage()
  if (!assistant) return
  assistant.content += payload.delta || ''
  chat.activity = null
})

chatSocket.on('activity', (payload) => {
  chat.activity = payload
})

chatSocket.on('sources', (payload) => {
  const assistant = streamingMessage()
  if (!assistant) return
  assistant.sources = payload.items || []
})

chatSocket.on('notice', (payload) => {
  const assistant = streamingMessage()
  if (!assistant) return
  assistant.notice = payload.message || ''
})

chatSocket.on('usage', (payload) => {
  const assistant = streamingMessage()
  if (!assistant) return
  assistant.output_tokens = payload.output_tokens || 0
  assistant.duration_ms = payload.duration_ms || 0
})

chatSocket.on('error', (payload) => {
  stopGenerationWatchdog()
  const assistant = streamingMessage() || chat.messages[chat.messages.length - 1]
  if (assistant && assistant.role === 'assistant') {
    assistant.status = 'error'
    assistant.error = payload.message || '生成失败，请重试'
  }
  chat.generating = false
  chat.activity = null
})

chatSocket.on('done', (payload) => {
  stopGenerationWatchdog()
  const assistant = streamingMessage()
  if (assistant) {
    assistant.status = payload.status || 'complete'
  }
  if (payload.title) applyTitle(payload.title)
  chat.generating = false
  chat.activity = null
  loadConversations().catch(() => {})
})

chatSocket.onStatus((status) => {
  chat.connection = status
  // 生成中断线时立刻告诉用户，避免界面一直停在「正在生成…」
  if (status !== 'open' && chat.generating) {
    stopGenerationWatchdog()
    const assistant = streamingMessage()
    if (assistant) {
      assistant.status = 'error'
      assistant.error = '连接已断开，请重试'
    }
    chat.generating = false
    chat.activity = null
  }
})

/** 更新会话标题（列表与当前会话）。 */
function applyTitle(title) {
  if (chat.current) chat.current.title = title
  const hit = chat.conversations.find((item) => item.id === chat.current?.id)
  if (hit) hit.title = title
}

// ---- 发送与操作 -----------------------------------------------------------

/**
 * 发送一条消息。
 * @param {string} content 用户输入
 * @param {number[]} attachmentIds 已上传的图片
 */
export async function send(content, attachmentIds = []) {
  if (chat.generating) return
  const text = String(content || '').trim()
  if (!text && !attachmentIds.length) return
  const conversation = await ensureConversation()
  pushMessage({
    id: -Date.now(),
    role: 'user',
    content: text,
    reasoning: '',
    status: 'complete',
    sources: [],
    attachments: [],
    created_at: new Date().toISOString(),
  })
  pushMessage({
    id: -Date.now() - 1,
    role: 'assistant',
    content: '',
    reasoning: '',
    status: 'streaming',
    model_name: chat.models.find((item) => item.id === chat.modelId)?.display_name || '',
    sources: [],
    attachments: [],
    created_at: new Date().toISOString(),
  })
  chat.generating = true
  chat.activity = null
  startGenerationWatchdog()
  chatSocket.send({
    type: 'stream',
    conversation_id: conversation.public_id,
    content: text,
    model_id: chat.modelId,
    attachment_ids: attachmentIds,
  })
}

/** 停止生成。 */
export function stop() {
  stopGenerationWatchdog()
  chatSocket.send({ type: 'stop' })
  const assistant = streamingMessage()
  if (assistant) assistant.status = 'stopped'
  chat.generating = false
  chat.activity = null
}

/** 重新生成上一条回答。 */
export function regenerate() {
  if (chat.generating || !chat.current) return
  // 去掉最后一条助手消息，重新占位
  const last = chat.messages[chat.messages.length - 1]
  if (last?.role === 'assistant') chat.messages.pop()
  pushMessage({
    id: -Date.now(),
    role: 'assistant',
    content: '',
    reasoning: '',
    status: 'streaming',
    model_name: chat.models.find((item) => item.id === chat.modelId)?.display_name || '',
    sources: [],
    attachments: [],
    created_at: new Date().toISOString(),
  })
  chat.generating = true
  startGenerationWatchdog()
  chatSocket.send({
    type: 'stream',
    conversation_id: chat.current.public_id,
    model_id: chat.modelId,
    regenerate: true,
  })
}

/** 点赞 / 踩。 */
export async function rate(message, rating) {
  if (!message.id || message.id < 0) return
  const next = message.feedback === rating ? '' : rating
  await chatApi.feedback(message.id, next)
  message.feedback = next
}

/** 删除会话。 */
export async function removeConversation(id) {
  await chatApi.deleteConversation(id)
  chat.conversations = chat.conversations.filter((item) => item.id !== id)
  if (chat.current?.id === id) startDraft()
}

/** 重命名会话。 */
export async function rename(id, title) {
  const data = await chatApi.renameConversation(id, title)
  const hit = chat.conversations.find((item) => item.id === id)
  if (hit) hit.title = data.conversation.title
  if (chat.current?.id === id) chat.current.title = data.conversation.title
}

/** 生成分享链接并复制。 */
export async function shareCurrent() {
  if (!chat.current?.public_id) return ''
  const data = await chatApi.createShare(chat.current.public_id)
  const { WEB_BASE } = await import('@/config/env')
  const url = `${WEB_BASE}/s/${data.share.token}`
  uni.setClipboardData({ data: url })
  return url
}
