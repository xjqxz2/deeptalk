/**
 * 双端共享的接口类型。
 * 说明：字段与 SPEC v1.0.0 的响应结构一一对应；新增字段先改契约再改类型。
 */

/** 统一响应信封。 */
export interface Envelope<T> {
  code: number
  msg: string
  data: T
}

/** 角色与套餐。 */
export type UserRole = 'admin' | 'user'
export type UserPlan = 'free' | 'pro' | 'team' | 'unlimited'

/** C / B 端共用的账号结构。 */
export interface User {
  id: number
  email: string
  nickname: string
  role: UserRole
  plan: UserPlan
  quota_tokens: number
  used_tokens: number
  used_percent: number
  disabled: boolean
  created_at: string
  last_login_at: string | null
  last_login_city: string
  last_login_ip: string
  avatar_text: string
  conversation_count: number
  message_count: number
  quota_reset_days: number
}

/** 用量卡片数据。 */
export interface UsageInfo {
  used_tokens: number
  quota_tokens: number
  used_percent: number
  reset_days: number
  plan: UserPlan
}

/** C 端模型选择项。 */
export interface ModelInfo {
  id: number
  model_id: string
  display_name: string
  description: string
  provider_name: string
  context_window: number
  is_default: boolean
}

/** 会话列表项。 */
export interface Conversation {
  id: number
  public_id: string
  title: string
  model_id: number
  updated_at: string
  message_count: number
}

/** 图片附件。 */
export interface Attachment {
  id: number
  name: string
  mime: string
  size: number
  width: number
  height: number
  url: string
}

/** 消息状态。 */
export type MessageStatus = 'complete' | 'streaming' | 'stopped' | 'error'

/** 联网检索来源。 */
export interface SearchSource {
  title: string
  url: string
  snippet: string
  /** 模型是否真正打开过该网页正文 */
  viewed?: boolean
}

/** 会话分享。 */
export interface ShareInfo {
  token: string
  created_at: string
}

/** 公开分享页的只读数据。 */
export interface SharedConversation {
  title: string
  owner: string
  shared_at: string
  messages: ChatMessage[]
}

/** A2UI v0.9 envelope，前端交给官方 Lit Renderer 处理。 */
export type A2UIMessage = Record<string, unknown>

/** 思考/检索过程中的实时状态，仅当前流式消息使用。 */
export type SearchActivityStatus = 'thinking' | 'searching' | 'opening' | 'reading' | 'found'

export interface SearchActivity {
  status: SearchActivityStatus
  label: string
  detail?: string
  url?: string
}

/** 消息结构（后端返回的持久化字段）。 */
export interface ChatMessage {
  id: number
  role: 'user' | 'assistant'
  content: string
  reasoning: string
  status: MessageStatus
  model_name: string
  input_tokens: number
  output_tokens: number
  duration_ms: number
  feedback: '' | 'up' | 'down'
  sources?: SearchSource[]
  searching?: boolean
  activity?: SearchActivity
  a2ui?: A2UIMessage[]
  attachments?: Attachment[]
  created_at: string
  created_at_label?: string
  error?: string
  notice?: string
  tokens_per_sec?: number
  character_count?: number
}

/** 注册开关等公开配置。 */
export interface AuthConfig {
  allow_registration: boolean
  has_users: boolean
  site_name: string
  web_search_enabled: boolean
}

/** 后台厂商卡片。 */
export interface ProviderInfo {
  id: number
  name: string
  code: string
  kind: 'deepseek' | 'openai' | 'aliyun' | 'moonshot' | 'custom'
  base_url: string
  api_key_masked: string
  has_api_key: boolean
  timeout_sec: number
  max_concurrency: number
  enabled: boolean
  verified: boolean
  status: 'active' | 'pending' | 'failed' | 'disabled'
  is_builtin: boolean
  is_default: boolean
  model_count: number
  last_latency_ms: number
  last_error: string
  models: AdminModel[]
}

/** 后台模型行。 */
export interface AdminModel {
  id: number
  provider_id: number
  provider_name: string
  model_id: string
  display_name: string
  description: string
  context_window: number
  max_concurrency: number
  enabled: boolean
  is_default: boolean
  is_fallback: boolean
}

/** 默认 / 兜底路由。 */
export interface Routing {
  default_model_id: number
  fallback_model_id: number
}

/** 厂商筛选计数。 */
export interface ProviderCounts {
  all: number
  builtin: number
  custom: number
  enabled: number
}

/** 接入健康。 */
export interface HealthItem {
  provider_id: number
  name: string
  ok: boolean
  latency_ms: number
}

export interface HealthInfo {
  upstreams_total: number
  upstreams_available: number
  errors_1h: number
  first_byte_p95_ms: number
  updated_at: string
  items: HealthItem[]
}

/** 最近变更。 */
export interface AuditItem {
  id: number
  actor_name: string
  action: string
  detail: string
  created_at: string
}

/** 后台用户统计。 */
export interface UserStats {
  total: number
  new_this_month: number
  active_7d: number
  active_rate: number
  monthly_tokens: number
  restricted: number
  quota_alert: number
  monthly_growth: number
}

/** 后台用户列表响应。 */
export interface UserListData {
  items: User[]
  total: number
  page: number
  page_size: number
  stats: UserStats
}

/** 后台概览。 */
export interface OverviewData {
  total_users: number
  new_this_month: number
  active_7d: number
  active_rate: number
  monthly_tokens: number
  monthly_growth: number
  restricted: number
  quota_alert: number
  providers_enabled: number
  models_enabled: number
  conversation_count: number
  monthly_messages: number
  health: HealthInfo
  recent_audits: AuditItem[]
}

/** 系统设置。 */
export interface SettingsInfo {
  allow_registration: boolean
  default_model_id: number
  fallback_model_id: number
  max_output_tokens: number
  site_name: string
  version: string
  storage_used_bytes: number
  storage_total_bytes: number
}

/** 用量时间序列。 */
export interface DailyUsage {
  date: string
  tokens: number
  messages: number
}

export interface UsageTimeline {
  days: DailyUsage[]
  totals: { tokens: number; messages: number }
}

/** 连接测试结果。 */
export interface TestResult {
  ok: boolean
  latency_ms: number
  models_found: number
  models: string[]
  message: string
}

/** 厂商保存请求。 */
export interface ProviderPayload {
  name: string
  kind: ProviderInfo['kind']
  base_url: string
  api_key?: string
  timeout_sec: number
  max_concurrency: number
  enabled: boolean
  is_default: boolean
  models?: Array<{
    model_id: string
    display_name: string
    description: string
    context_window: number
    max_concurrency: number
    enabled: boolean
  }>
}

/** SSE 流式事件。 */
export interface StreamMeta {
  conversation_id: number
  assistant_message_id: number
  user_message_id: number
  user_message: {
    id: number
    role: 'user'
    content: string
    status: MessageStatus
    created_at: string
    input_tokens: number
    attachments?: Attachment[]
  }
  model_name: string
  model_id: number
  title: string
  deep_thinking: boolean
  web_search: boolean
}

export interface StreamUsage {
  input_tokens: number
  output_tokens: number
  total_tokens: number
  duration_ms: number
  tokens_per_sec: number
  character_count: number
}
