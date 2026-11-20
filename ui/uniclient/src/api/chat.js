/** 会话与消息接口。 */
import { request } from './http'

/** 会话列表。 */
export function listConversations(keyword = '') {
  return request({ url: `/api/v1/c/conversations${keyword ? `?keyword=${encodeURIComponent(keyword)}` : ''}` })
}

/** 新建会话。 */
export function createConversation(title = '') {
  return request({ url: '/api/v1/c/conversations', method: 'POST', data: title ? { title } : {} })
}

/** 会话详情与消息。 */
export function getConversation(id) {
  return request({ url: `/api/v1/c/conversations/${encodeURIComponent(id)}` })
}

/** 重命名。 */
export function renameConversation(id, title) {
  return request({ url: `/api/v1/c/conversations/${encodeURIComponent(id)}`, method: 'PATCH', data: { title } })
}

/** 删除会话。 */
export function deleteConversation(id) {
  return request({ url: `/api/v1/c/conversations/${encodeURIComponent(id)}`, method: 'DELETE' })
}

/** 消息点赞/踩。 */
export function feedback(messageId, rating) {
  return request({ url: `/api/v1/c/messages/${messageId}/feedback`, method: 'POST', data: { rating } })
}

/** 可用模型。 */
export function listModels() {
  return request({ url: '/api/v1/c/models' })
}

/** 创建分享，返回 token。 */
export function createShare(id) {
  return request({ url: `/api/v1/c/conversations/${encodeURIComponent(id)}/share`, method: 'POST' })
}

/** 取消分享。 */
export function revokeShare(id) {
  return request({ url: `/api/v1/c/conversations/${encodeURIComponent(id)}/share`, method: 'DELETE' })
}
