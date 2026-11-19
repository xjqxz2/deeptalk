/** B 端管理接口：概览、厂商、模型、用户、用量与系统设置。 */
import { del, download, get, patch, post, put } from '@deeptalk/shared'
import type {
  AdminModel,
  AuditItem,
  HealthInfo,
  OverviewData,
  ProviderCounts,
  ProviderInfo,
  ProviderPayload,
  Routing,
  SettingsInfo,
  TestResult,
  UsageTimeline,
  User,
  UserListData,
  UserStats,
} from '@deeptalk/shared'

/** 厂商列表响应。 */
export interface ProviderListData {
  items: ProviderInfo[]
  counts: ProviderCounts
  health: HealthInfo
}

/** 后台用户列表查询参数。 */
export interface AdminUserQuery {
  q?: string
  status?: 'all' | 'normal' | 'restricted' | 'quota'
  sort?: 'recent' | 'created' | 'usage'
  page?: number
  page_size?: number
}

/** 获取概览统计。 */
export function fetchOverview(): Promise<OverviewData> {
  return get<OverviewData>('/b/overview')
}

/** 获取厂商列表、计数与健康状态。 */
export function fetchProviders(filter: 'all' | 'builtin' | 'custom' = 'all'): Promise<ProviderListData> {
  return get<ProviderListData>('/b/providers', { filter })
}

/** 新建厂商。 */
export function createProvider(payload: ProviderPayload): Promise<{ provider: ProviderInfo }> {
  return post<{ provider: ProviderInfo }>('/b/providers', payload)
}

/** 编辑厂商。 */
export function updateProvider(id: number, payload: ProviderPayload): Promise<{ provider: ProviderInfo }> {
  return put<{ provider: ProviderInfo }>(`/b/providers/${id}`, payload)
}

/** 启用 / 停用厂商。 */
export function toggleProvider(id: number, enabled: boolean): Promise<{ provider: ProviderInfo }> {
  return patch<{ provider: ProviderInfo }>(`/b/providers/${id}/enabled`, { enabled })
}

/** 删除自定义厂商。 */
export function deleteProvider(id: number): Promise<null> {
  return del<null>(`/b/providers/${id}`)
}

/** 测试已保存厂商。 */
export function testProvider(id: number): Promise<TestResult> {
  return post<TestResult>(`/b/providers/${id}/test`)
}

/** 测试未保存配置。 */
export function testDraft(payload: ProviderPayload, id?: number): Promise<TestResult> {
  return post<TestResult>('/b/providers/test', { ...payload, id: id ?? 0 })
}

/** 模型路由表与默认 / 兜底路由。 */
export function fetchModels(): Promise<{ items: AdminModel[]; routing: Routing }> {
  return get<{ items: AdminModel[]; routing: Routing }>('/b/models')
}

/** 新增模型。 */
export function createModel(payload: {
  provider_id: number
  model_id: string
  display_name: string
  description: string
  context_window: number
  max_concurrency: number
  enabled: boolean
}): Promise<{ model: AdminModel }> {
  return post<{ model: AdminModel }>('/b/models', payload)
}

/** 编辑模型。 */
export function updateModel(
  id: number,
  payload: {
    provider_id: number
    model_id: string
    display_name: string
    description: string
    context_window: number
    max_concurrency: number
    enabled: boolean
  },
): Promise<{ model: AdminModel }> {
  return put<{ model: AdminModel }>(`/b/models/${id}`, payload)
}

/** 启用 / 停用模型。 */
export function toggleModel(id: number, enabled: boolean): Promise<{ model: AdminModel }> {
  return patch<{ model: AdminModel }>(`/b/models/${id}/enabled`, { enabled })
}

/** 更新默认 / 兜底路由。 */
export function updateRouting(payload: Routing): Promise<Routing> {
  return put<Routing>('/b/routing', payload)
}

/** 接入健康。 */
export function fetchHealth(): Promise<HealthInfo> {
  return get<HealthInfo>('/b/health')
}

/** 最近变更。 */
export function fetchAuditLogs(limit = 6): Promise<{ items: AuditItem[] }> {
  return get<{ items: AuditItem[] }>('/b/audit-logs', { limit })
}

/** 用户统计。 */
export function fetchUserStats(): Promise<UserStats> {
  return get<UserStats>('/b/users/stats')
}

/** 用户列表。 */
export function fetchUsers(query: AdminUserQuery): Promise<UserListData> {
  return get<UserListData>('/b/users', {
    q: query.q,
    status: query.status ?? 'all',
    sort: query.sort ?? 'recent',
    page: query.page ?? 1,
    page_size: query.page_size ?? 8,
  })
}

/** 用户详情。 */
export function fetchUserDetail(id: number): Promise<{ user: User }> {
  return get<{ user: User }>(`/b/users/${id}`)
}

/** 调整额度。 */
export function updateQuota(id: number, quotaTokens: number): Promise<{ user: User }> {
  return patch<{ user: User }>(`/b/users/${id}/quota`, { quota_tokens: quotaTokens })
}

/** 调整套餐。 */
export function updatePlan(id: number, plan: 'free' | 'pro' | 'team' | 'unlimited'): Promise<{ user: User }> {
  return patch<{ user: User }>(`/b/users/${id}/plan`, { plan })
}

/** 限制 / 恢复登录。 */
export function setUserDisabled(id: number, disabled: boolean): Promise<{ user: User }> {
  return patch<{ user: User }>(`/b/users/${id}/disabled`, { disabled })
}

/** 重置密码，返回一次性密码。 */
export function resetUserPassword(id: number): Promise<{ password: string; expires_hint: string }> {
  return post<{ password: string; expires_hint: string }>(`/b/users/${id}/reset-password`)
}

/** 强制退出全部设备。 */
export function logoutAllSessions(id: number): Promise<null> {
  return post<null>(`/b/users/${id}/logout-all`)
}

/** 管理员建号。 */
export function createUser(payload: {
  email: string
  password: string
  nickname: string
  plan: string
  quota_tokens: number
}): Promise<{ user: User }> {
  return post<{ user: User }>('/b/users', payload)
}

/** 导出账号 CSV。 */
export function exportUsers(): Promise<void> {
  return download('/b/users/export.csv', 'deeptalk-users.csv')
}

/** 系统设置。 */
export function fetchSettings(): Promise<SettingsInfo> {
  return get<SettingsInfo>('/b/settings')
}

/** 更新系统设置。 */
export function updateSettings(payload: {
  allow_registration?: boolean
  default_model_id?: number
  fallback_model_id?: number
  max_output_tokens?: number
}): Promise<SettingsInfo> {
  return put<SettingsInfo>('/b/settings', payload)
}

/** 用量时间序列。 */
export function fetchUsageTimeline(days: number): Promise<UsageTimeline> {
  return get<UsageTimeline>('/b/usage', { days })
}
