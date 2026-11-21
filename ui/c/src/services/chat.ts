/** C 端会话与消息接口；流式请求在 chat store 中单独处理。 */
import { del, get, patch, post, upload } from '@deeptalk/shared'
import type { Attachment, ChatMessage, Conversation, ModelInfo, ShareInfo, SharedConversation } from '@deeptalk/shared'

/** 会话列表，keyword 由服务端做标题模糊匹配。 */
export function listConversations(keyword = ''): Promise<{ items: Conversation[] }> {
  return get<{ items: Conversation[] }>('/c/conversations', keyword ? { keyword } : undefined)
}

/** 新建会话。 */
export function createConversation(): Promise<{ conversation: Conversation }> {
  return post<{ conversation: Conversation }>('/c/conversations', {})
}

/** 会话详情与全部消息。 */
export function getConversation(id: number | string): Promise<{ conversation: Conversation; messages: ChatMessage[] }> {
  return get<{ conversation: Conversation; messages: ChatMessage[] }>(`/c/conversations/${encodeURIComponent(String(id))}`)
}

/** 重命名会话。 */
export function renameConversation(id: number, title: string): Promise<{ conversation: Conversation }> {
  return patch<{ conversation: Conversation }>(`/c/conversations/${id}`, { title })
}

/** 删除会话。 */
export function deleteConversation(id: number): Promise<null> {
  return del<null>(`/c/conversations/${id}`)
}

/** 消息赞 / 踩。 */
/** 生成过程中插话：入队，当前轮结束后继续。 */
export function interject(conversationID: number | string, content: string): Promise<{ queued: boolean }> {
  return post(`/c/conversations/${encodeURIComponent(String(conversationID))}/interjections`, { content })
}

export function sendFeedback(id: number, rating: 'up' | 'down' | ''): Promise<null> {
  return post<null>(`/c/messages/${id}/feedback`, { rating })
}

/** 上传一张图片，返回附件元数据。 */
export function uploadAttachment(file: File): Promise<{ attachment: Attachment }> {
  const form = new FormData()
  form.append('file', file, file.name)
  return upload<{ attachment: Attachment }>('/c/attachments', form)
}

/** 读取当前会话的分享；没有分享时 share 为 null。 */
export function getShare(id: number | string): Promise<{ share: ShareInfo | null }> {
  return get<{ share: ShareInfo | null }>(`/c/conversations/${encodeURIComponent(String(id))}/share`)
}

/** 创建（或复用）分享链接。 */
export function createShare(id: number | string): Promise<{ share: ShareInfo }> {
  return post<{ share: ShareInfo }>(`/c/conversations/${encodeURIComponent(String(id))}/share`)
}

/** 撤销分享，旧链接立即失效。 */
export function revokeShare(id: number | string): Promise<null> {
  return del<null>(`/c/conversations/${encodeURIComponent(String(id))}/share`)
}

/** 读取公开分享，不需要登录。 */
export function getShared(token: string): Promise<SharedConversation> {
  return get<SharedConversation>(`/c/public/shares/${encodeURIComponent(token)}`)
}

/** 当前可用模型。 */
export function listModels(): Promise<{ items: ModelInfo[] }> {
  return get<{ items: ModelInfo[] }>('/c/models')
}
