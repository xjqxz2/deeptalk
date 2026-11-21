/**
 * 对话 Store：会话列表、消息、模型选择与 SSE 流式生成。
 * 设计要点：
 * - 流式增量直接写入消息对象，Vue 的批处理天然合帧；不要每个 token 触发一次滚动计算；
 * - 客户端中断使用 AbortController，服务端会保留已输出内容并标记 stopped；
 * - 重新生成覆盖最后一条回答，不重复插入用户消息。
 */
import { computed, reactive, ref } from 'vue'
import { defineStore } from 'pinia'
import { ApiError, conversationGroup, signatureHeadersFor } from '@deeptalk/shared'
import type {
  A2UIMessage,
  Attachment,
  ChatMessage,
  Conversation,
  ModelInfo,
  SearchActivity,
  SearchSource,
  StreamMeta,
  StreamUsage,
} from '@deeptalk/shared'
import * as chatApi from '@/services/chat'
import { useUserStore } from './user'

/** 会话分组顺序。 */
const GROUP_ORDER = ['今天', '7 天内', '30 天内', '更早'] as const

/** SSE 事件块。 */
interface SSEBlock {
  event: string
  data: string
}

/** 单个会话的运行态；流式生成可以同时在多个会话中进行。 */
interface ConversationRun {
  conversation: Conversation
  messages: ChatMessage[]
  generating: boolean
  loaded: boolean
  abortController: AbortController | null
}

/** 解析一个 SSE 事件块，忽略注释行与空块。 */
function parseSSEBlock(block: string): SSEBlock | null {
  let event = 'message'
  const dataLines: string[] = []
  for (const rawLine of block.split('\n')) {
    const line = rawLine.replace(/\r$/, '')
    if (line.startsWith('event:')) {
      event = line.slice(6).trim()
    } else if (line.startsWith('data:')) {
      dataLines.push(line.slice(5).replace(/^ /, ''))
    }
  }
  if (dataLines.length === 0) return null
  return { event, data: dataLines.join('\n') }
}

export const useChatStore = defineStore('chat', () => {
  const userStore = useUserStore()

  /** 会话列表。 */
  const conversations = ref<Conversation[]>([])
  /** 可用模型。 */
  const models = ref<ModelInfo[]>([])
  /** 当前会话。 */
  const current = ref<Conversation | null>(null)
  /** 当前会话消息；切换会话时指向对应 ConversationRun.messages。 */
  const messages = ref<ChatMessage[]>([])
  /** 每个会话独立的运行态，支持后台并行生成。 */
  const runs = reactive(new Map<number, ConversationRun>())
  /** 侧栏搜索关键词。 */
  const searchKeyword = ref('')
  /** 当前选中的模型主键。 */
  const selectedModelId = ref(0)
  /** 待发送的图片附件。 */
  const attachments = ref<Attachment[]>([])
  /** 图片上传中。 */
  const uploading = ref(false)
  /** 列表加载中。 */
  const loadingConversations = ref(false)
  /** 消息加载中。 */
  const loadingMessages = ref(false)
  /** 列表级错误。 */
  const listError = ref('')
  /** 新建会话后待发送的首条消息。 */
  const pendingPrompt = ref<string | null>(null)
  /** 当前会话对应的运行态。 */
  const currentRun = computed(() => (current.value ? runs.get(current.value.id) ?? null : null))
  /** 当前会话是否正在生成；其他会话的生成不会锁住当前输入框。 */
  const generating = computed(() => currentRun.value?.generating ?? false)
  /** 会话加载序号，用于丢弃切换页面后返回的旧请求。 */
  let loadSeq = 0

  /** 当前选中的模型对象。 */
  const selectedModel = computed(() => models.value.find((item) => item.id === selectedModelId.value) ?? null)

  /** 确保会话有一个运行态，并更新其中的会话对象。 */
  function ensureRun(conversation: Conversation): ConversationRun {
    let run = runs.get(conversation.id)
    if (!run) {
      run = reactive<ConversationRun>({
        conversation,
        messages: [],
        generating: false,
        loaded: false,
        abortController: null,
      })
      runs.set(conversation.id, run)
    } else {
      run.conversation = conversation
    }
    return run
  }

  /** 按数字 ID 或 public_id 查找运行态。 */
  function findRun(id: number | string): ConversationRun | null {
    const text = String(id)
    for (const run of runs.values()) {
      if (String(run.conversation.id) === text || run.conversation.public_id === text) return run
    }
    return null
  }

  /** 切换当前会话到指定运行态，并同步模型选择。 */
  function activateRun(run: ConversationRun): void {
    current.value = run.conversation
    messages.value = run.messages
    if (
      run.conversation.model_id &&
      models.value.some((item) => item.id === run.conversation.model_id)
    ) {
      selectedModelId.value = run.conversation.model_id
    }
  }

  /** 侧栏用：仍有后台生成的会话 ID。 */
  const generatingIds = computed(() => {
    const ids: number[] = []
    for (const run of runs.values()) {
      if (run.generating) ids.push(run.conversation.id)
    }
    return ids
  })

  /** 同步会话标题到运行态、当前会话和侧栏。 */
  function updateTitle(id: number, title: string): void {
    const run = runs.get(id)
    if (run) run.conversation.title = title
    if (current.value?.id === id) current.value.title = title
    const hit = conversations.value.find((item) => item.id === id)
    if (hit) hit.title = title
  }

  /** 按设计稿分组后的会话列表。 */
  const groupedConversations = computed(() => {
    const buckets = new Map<string, Conversation[]>()
    for (const item of conversations.value) {
      const group = conversationGroup(item.updated_at)
      const list = buckets.get(group) ?? []
      list.push(item)
      buckets.set(group, list)
    }
    return GROUP_ORDER.filter((group) => buckets.has(group)).map((group) => ({
      label: group,
      items: buckets.get(group) ?? [],
    }))
  })

  /** 加载模型列表，并恢复用户上次选择的模型。 */
  async function loadModels(): Promise<void> {
    const data = await chatApi.listModels()
    models.value = data.items
    const stored = Number(localStorage.getItem('deeptalk.model'))
    const valid = data.items.some((item) => item.id === selectedModelId.value)
    if (!valid && stored && data.items.some((item) => item.id === stored)) {
      selectedModelId.value = stored
    }
    if (!data.items.some((item) => item.id === selectedModelId.value)) {
      selectedModelId.value = data.items.find((item) => item.is_default)?.id ?? data.items[0]?.id ?? 0
      try {
        localStorage.setItem('deeptalk.model', String(selectedModelId.value))
      } catch {
        // 忽略存储失败
      }
    }
  }

  /** 切换模型；只影响下一条消息，不追溯历史。 */
  function setModel(id: number): void {
    selectedModelId.value = id
    localStorage.setItem('deeptalk.model', String(id))
  }

  /** 加载会话列表；搜索关键字由侧栏防抖后传入。 */
  async function loadConversations(keyword = searchKeyword.value): Promise<void> {
    loadingConversations.value = true
    listError.value = ''
    try {
      const data = await chatApi.listConversations(keyword)
      conversations.value = data.items
      // 后台流结束也会刷新标题；同步所有运行态，而不只是当前会话
      for (const item of data.items) {
        const run = runs.get(item.id)
        if (run) run.conversation.title = item.title
      }
      if (current.value) {
        const hit = data.items.find((item) => item.id === current.value?.id)
        if (hit) current.value.title = hit.title
      }
    } catch (error) {
      listError.value = error instanceof ApiError ? error.message : '会话列表加载失败'
    } finally {
      loadingConversations.value = false
    }
  }

  /** 搜索会话。 */
  async function search(keyword: string): Promise<void> {
    searchKeyword.value = keyword
    await loadConversations(keyword)
  }

  /** 新建空会话，不改变当前消息（由页面决定是否跳转）。 */
  async function createConversation(): Promise<Conversation> {
    const data = await chatApi.createConversation()
    conversations.value = [data.conversation, ...conversations.value]
    loadSeq += 1
    loadingMessages.value = false
    const run = ensureRun(data.conversation)
    run.messages = []
    run.loaded = true
    activateRun(run)
    return data.conversation
  }

  /**
   * 打开会话。若该会话仍在后台生成，直接切回活动消息数组，不打断流；
   * 其他会话的生成也不受影响。
   */
  async function loadConversation(id: number | string): Promise<void> {
    const existing = findRun(id)
    if (existing?.generating) {
      loadSeq += 1
      loadingMessages.value = false
      activateRun(existing)
      return
    }
    const token = ++loadSeq
    loadingMessages.value = true
    try {
      const data = await chatApi.getConversation(id)
      if (token !== loadSeq) return
      const run = ensureRun(data.conversation)
      if (!run.generating) {
        run.messages = data.messages
        run.loaded = true
      }
      activateRun(run)
      // 空会话保留用户在空态选择的模型；已有消息的会话恢复其最近模型
      if (!run.generating && data.messages.length > 0) {
        const modelID = data.conversation.model_id
        if (modelID && models.value.some((item) => item.id === modelID)) {
          selectedModelId.value = modelID
        }
      }
    } finally {
      if (token === loadSeq) loadingMessages.value = false
    }
  }

  /** 重命名当前或指定会话。 */
  async function renameConversation(id: number, title: string): Promise<void> {
    const data = await chatApi.renameConversation(id, title)
    updateTitle(id, data.conversation.title)
  }

  /** 删除会话并从列表移除。 */
  async function deleteConversation(id: number): Promise<void> {
    const run = runs.get(id)
    run?.abortController?.abort()
    await chatApi.deleteConversation(id)
    runs.delete(id)
    conversations.value = conversations.value.filter((item) => item.id !== id)
    if (current.value?.id === id) {
      loadSeq += 1
      loadingMessages.value = false
      current.value = null
      messages.value = []
    }
  }

  /** 记录待发送的新会话首条消息。 */
  function queuePrompt(text: string): void {
    pendingPrompt.value = text
  }

  /** 取出并清空待发送消息。 */
  function consumePendingPrompt(): string | null {
    const text = pendingPrompt.value
    pendingPrompt.value = null
    return text
  }

  /** 回到空态草稿：清空当前会话但保留历史列表。 */
  /** 上传图片；只保留最近 4 张，单张上限与后端保持一致（8MB）。 */
  async function uploadFiles(files: File[]): Promise<void> {
    const images = files.filter((file) => file.type.startsWith('image/'))
    if (!images.length) {
      throw new Error('仅支持图片文件')
    }
    const remaining = 4 - attachments.value.length
    if (remaining <= 0) {
      throw new Error('单条消息最多 4 张图片')
    }
    uploading.value = true
    try {
      for (const file of images.slice(0, remaining)) {
        if (file.size > 8 * 1024 * 1024) {
          throw new Error(`图片 ${file.name} 超过 8MB`)
        }
        const data = await chatApi.uploadAttachment(file)
        attachments.value.push(data.attachment)
      }
    } finally {
      uploading.value = false
    }
  }

  /** 移除一张待发送图片。 */
  function removeAttachment(id: number): void {
    attachments.value = attachments.value.filter((item) => item.id !== id)
  }

  /** 清空待发送图片。 */
  function clearAttachments(): void {
    attachments.value = []
  }

  function startNewDraft(): void {
    // 只重置当前草稿，不打断其他会话的后台生成
    loadSeq += 1
    loadingMessages.value = false
    current.value = null
    messages.value = []
    pendingPrompt.value = null
    searchKeyword.value = ''
    attachments.value = []
  }

  /** 停止生成：只中断当前会话，其他会话继续。 */
  function stop(): void {
    currentRun.value?.abortController?.abort()
  }

  /** 更新本地消息的反馈状态。 */
  async function feedback(message: ChatMessage, rating: 'up' | 'down'): Promise<void> {
    const next = message.feedback === rating ? '' : rating
    const previous = message.feedback
    message.feedback = next
    if (message.id <= 0) return
    try {
      await chatApi.sendFeedback(message.id, next)
    } catch {
      message.feedback = previous
      throw new Error('反馈失败，请稍后重试')
    }
  }

  /**
   * 发送消息或重新生成。
   * 调用前必须已有 current 会话；同一会话生成中禁止重复提交。
   */
  /**
   * 插话：生成过程中补一句话。
   * 本地先挂一条"排队中"的消息，服务端在当前轮结束后把它并入对话并继续生成。
   */
  async function interject(conversationID: number, content: string): Promise<void> {
    const run = currentRun.value
    if (!run) return
    const optimistic: ChatMessage = reactive({
      id: -Date.now(),
      role: 'user',
      content,
      reasoning: '',
      status: 'queued',
      model_name: '',
      input_tokens: 0,
      output_tokens: 0,
      duration_ms: 0,
      feedback: '',
      interjection: true,
      created_at: new Date().toISOString(),
    })
    run.messages.push(optimistic)
    try {
      await chatApi.interject(conversationID, content)
      // 排队超时兜底：60 秒还没被处理就提示，避免永久「排队中」
      window.setTimeout(() => {
        if (optimistic.status === 'queued') {
          optimistic.status = 'error'
          optimistic.error = '插话未被处理，请重新发送'
        }
      }, 60_000)
    } catch (error) {
      // 当前没有正在生成的回答：撤掉排队消息，改为普通发送
      run.messages = run.messages.filter((item) => item !== optimistic)
      throw error
    }
  }

  async function send(content: string, options: { regenerate?: boolean } = {}): Promise<void> {
    const conversation = current.value
    const run = currentRun.value
    if (!conversation || !run) return
    const regenerate = options.regenerate === true
    const text = content.trim()
    // 生成中不再拒绝输入：改为插话，服务端在当前轮结束后接着处理
    if (run.generating && !regenerate && text) {
      await interject(conversation.id, text)
      return
    }
    if (run.generating) return
    const pendingAttachments = regenerate ? [] : [...attachments.value]
    if (!regenerate && !text && pendingAttachments.length === 0) return
    if (!selectedModelId.value) {
      throw new ApiError('当前没有可用模型，请联系管理员', 3001, 400)
    }
    const targetMessages = run.messages
    if (regenerate) {
      // 服务端会删除旧回答；前端先移除本地最后一条助手消息，保持界面不重复
      while (targetMessages.length > 0 && targetMessages[targetMessages.length - 1].role === 'assistant') {
        targetMessages.pop()
      }
    } else {
      targetMessages.push({
        id: -Date.now(),
        role: 'user',
        content: text,
        reasoning: '',
        status: 'complete',
        model_name: '',
        input_tokens: 0,
        output_tokens: 0,
        duration_ms: 0,
        feedback: '',
        attachments: pendingAttachments,
        created_at: new Date().toISOString(),
      })
    }
    // 必须使用 reactive 包装后再 push，否则后续直接改原始对象不会触发视图更新
    const assistant = reactive<ChatMessage>({
      id: -(Date.now() + 1),
      role: 'assistant',
      content: '',
      reasoning: '',
      status: 'streaming',
      model_name: selectedModel.value?.display_name ?? '',
      input_tokens: 0,
      output_tokens: 0,
      duration_ms: 0,
      feedback: '',
      created_at: new Date().toISOString(),
    })
    targetMessages.push(assistant)
    // 插话轮会复用一个连接继续推流：meta 到达时把写入目标切到新的助手消息
    let activeAssistant = assistant
    run.generating = true
    const controller = new AbortController()
    run.abortController = controller
    const conversationID = conversation.id
    let finished = false

    // 图片随消息一起发出后立即从草稿移除，避免切到另一个会话时串图
    if (!regenerate && pendingAttachments.length > 0) {
      const sentIDs = new Set(pendingAttachments.map((item) => item.id))
      attachments.value = attachments.value.filter((item) => !sentIDs.has(item.id))
    }

    /** 处理单个 SSE 事件。 */
    const applyEvent = (block: SSEBlock): void => {
      let payload: unknown
      try {
        payload = JSON.parse(block.data)
      } catch {
        return
      }
      switch (block.event) {
        case 'meta': {
          const meta = payload as StreamMeta
          // 插话轮：把本地"排队中"的消息换成服务端落库后的真实消息
          if (meta.user_message?.interjection) {
            for (let index = targetMessages.length - 1; index >= 0; index -= 1) {
              const candidate = targetMessages[index]
              if (candidate.role === 'user' && candidate.status === 'queued') {
                candidate.id = meta.user_message.id
                candidate.created_at = meta.user_message.created_at
                candidate.input_tokens = meta.user_message.input_tokens
                candidate.status = 'complete'
                break
              }
            }
          }
          if (meta.user_message?.interjection) {
            const next = reactive<ChatMessage>({
              id: meta.assistant_message_id,
              role: 'assistant',
              content: '',
              reasoning: '',
              status: 'streaming',
              model_name: meta.model_name ?? '',
              input_tokens: 0,
              output_tokens: 0,
              duration_ms: 0,
              feedback: '',
              created_at: new Date().toISOString(),
            })
            targetMessages.push(next)
            activeAssistant = next
            return
          }
          assistant.id = meta.assistant_message_id
          assistant.model_name = meta.model_name
          assistant.created_at = meta.user_message?.created_at ?? assistant.created_at
          if (!regenerate) {
            // 用服务端返回的真实 ID 替换本地乐观消息
            for (let index = targetMessages.length - 2; index >= 0; index -= 1) {
              const candidate = targetMessages[index]
              if (candidate.role === 'user' && candidate.id < 0) {
                candidate.id = meta.user_message.id
                candidate.created_at = meta.user_message.created_at
                candidate.input_tokens = meta.user_message.input_tokens
                if (meta.user_message.attachments?.length) {
                  candidate.attachments = meta.user_message.attachments
                }
                break
              }
            }
          }
          if (meta.title) updateTitle(conversationID, meta.title)
          break
        }
        case 'reasoning':
          activeAssistant.reasoning += (payload as { delta: string }).delta ?? ''
          break
        case 'delta': {
          const delta = (payload as { delta: string }).delta ?? ''
          if (delta) activeAssistant.activity = undefined
          activeAssistant.content += delta
          break
        }
        case 'search':
          activeAssistant.searching = true
          break
        case 'activity':
          activeAssistant.activity = payload as SearchActivity
          break
        case 'a2ui': {
          const messages = (payload as { messages?: A2UIMessage[] }).messages ?? []
          activeAssistant.a2ui = [...(activeAssistant.a2ui ?? []), ...messages]
          break
        }
        case 'sources':
          activeAssistant.sources = (payload as { items: SearchSource[] }).items ?? []
          activeAssistant.searching = false
          break
        case 'notice': {
          const notice = payload as { message: string; interjection?: string }
          if (notice.interjection) {
            const stuck = run.messages.find((item) => item.role === 'user' && item.status === 'queued')
            if (stuck) {
              stuck.status = 'error'
              stuck.error = notice.message || '插话未处理，请重新发送'
            }
          }
          activeAssistant.notice = notice.message
          activeAssistant.searching = false
          activeAssistant.activity = undefined
          break
        }
        case 'usage': {
          const usage = payload as StreamUsage
          activeAssistant.input_tokens = usage.input_tokens
          activeAssistant.output_tokens = usage.output_tokens
          activeAssistant.duration_ms = usage.duration_ms
          activeAssistant.tokens_per_sec = usage.tokens_per_sec
          activeAssistant.character_count = usage.character_count
          break
        }
        case 'error':
          activeAssistant.status = 'error'
          activeAssistant.searching = false
          activeAssistant.activity = undefined
          activeAssistant.error = (payload as { message: string }).message || '生成失败，请重试'
          break
        case 'done': {
          const done = payload as { status: ChatMessage['status']; title?: string }
          activeAssistant.status = done.status ?? 'complete'
          activeAssistant.searching = false
          activeAssistant.activity = undefined
          finished = true
          if (done.title) updateTitle(conversationID, done.title)
          break
        }
        default:
          // ping 等保活事件无需处理
          break
      }
    }

    try {
      const streamBody = JSON.stringify({
        content: text,
        // 模型可能刚被管理员删除；selectedModel 只会在当前列表中存在
        model_id: selectedModel.value?.id ?? 0,
        attachment_ids: pendingAttachments.map((item) => item.id),
        regenerate,
      })
      const streamPath = `/api/v1/c/conversations/${conversationID}/stream`
      const signHeaders = await signatureHeadersFor('POST', streamPath, streamBody)
      const response = await fetch(streamPath, {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json', ...signHeaders },
        body: streamBody,
        signal: controller.signal,
      })
      if (!response.ok) {
        let message = '生成失败，请稍后重试'
        let code = 2001
        try {
          const envelope = (await response.json()) as { code: number; msg: string }
          message = envelope.msg || message
          code = envelope.code ?? code
        } catch {
          // 非 JSON 错误体时保留默认文案
        }
        throw new ApiError(message, code, response.status)
      }
      if (!response.body) {
        throw new ApiError('浏览器不支持流式响应', 2001, 0)
      }
      const reader = response.body.getReader()
      const decoder = new TextDecoder()
      let buffer = ''
      for (;;) {
        const { done, value } = await reader.read()
        if (done) break
        buffer += decoder.decode(value, { stream: true })
        let separator = buffer.indexOf('\n\n')
        while (separator >= 0) {
          const block = buffer.slice(0, separator)
          buffer = buffer.slice(separator + 2)
          const parsed = parseSSEBlock(block)
          if (parsed) applyEvent(parsed)
          separator = buffer.indexOf('\n\n')
        }
      }
      const tail = parseSSEBlock(buffer)
      if (tail) applyEvent(tail)
      if (!finished && assistant.status === 'streaming') {
        assistant.status = assistant.content || assistant.reasoning ? 'stopped' : 'error'
        if (assistant.status === 'stopped') assistant.notice = '连接已中断，已保留当前内容'
        else assistant.error = '未收到回答，请重试'
      }
    } catch (error) {
      if (error instanceof DOMException && error.name === 'AbortError') {
        assistant.status = 'stopped'
      } else {
        assistant.status = 'error'
        assistant.error = error instanceof ApiError ? error.message : '生成失败，请重试'
      }
    } finally {
      run.generating = false
      if (run.abortController === controller) run.abortController = null
      // 刷新侧栏标题与用量，失败不打断对话
      await Promise.all([loadConversations(), userStore.refreshUsage()])
    }
  }

  /** 重新生成最后一条回答。 */
  async function regenerate(): Promise<void> {
    await send('', { regenerate: true })
  }

  /** 退出登录时中止所有后台生成并清空对话状态。 */
  function reset(): void {
    for (const run of runs.values()) {
      run.abortController?.abort()
      run.generating = false
      run.abortController = null
    }
    runs.clear()
    loadSeq += 1
    loadingMessages.value = false
    conversations.value = []
    messages.value = []
    current.value = null
    searchKeyword.value = ''
    pendingPrompt.value = null
    attachments.value = []
  }

  return {
    conversations,
    groupedConversations,
    models,
    current,
    messages,
    searchKeyword,
    selectedModel,
    selectedModelId,
    generating,
    generatingIds,
    attachments,
    uploading,
    loadingConversations,
    loadingMessages,
    listError,
    loadModels,
    setModel,
    uploadFiles,
    removeAttachment,
    clearAttachments,
    loadConversations,
    search,
    createConversation,
    loadConversation,
    renameConversation,
    deleteConversation,
    queuePrompt,
    consumePendingPrompt,
    startNewDraft,
    send,
    stop,
    regenerate,
    feedback,
    reset,
  }
})
