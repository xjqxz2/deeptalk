/**
 * 展示层格式化工具：数字、tokens、时间与套餐文案。
 * 所有页面共用，避免同一套格式出现多个版本。
 */
import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'
import 'dayjs/locale/zh-cn'

dayjs.extend(relativeTime)
dayjs.locale('zh-cn')

/** 套餐展示文案。 */
export const PLAN_LABELS: Record<string, string> = {
  free: '免费版',
  pro: '专业版',
  team: '团队版',
  unlimited: '无限版',
}

/** 角色展示文案。 */
export const ROLE_LABELS: Record<string, string> = {
  admin: '平台管理员',
  user: '普通用户',
}

/** 套餐额度文案，与后端默认额度一致。 */
export const PLAN_QUOTAS: Record<string, number> = {
  free: 4_000_000,
  pro: 20_000_000,
  team: 50_000_000,
  unlimited: 0,
}

/** 千分位数字，例如 12,480。 */
export function formatNumber(value: number): string {
  return new Intl.NumberFormat('zh-CN').format(Math.round(value))
}

/**
 * tokens 紧凑格式：1.28M / 38.6M / 500K。
 * 说明：去掉多余的 .0，保持与设计稿一致的观感。
 */
export function formatTokens(value: number): string {
  if (!Number.isFinite(value) || value <= 0) return '0'
  if (value >= 1_000_000) {
    const millions = value / 1_000_000
    return `${trimZero(millions.toFixed(millions >= 10 ? 1 : 2))}M`
  }
  if (value >= 1_000) {
    const thousands = value / 1_000
    return `${trimZero(thousands.toFixed(thousands >= 10 ? 0 : 1))}K`
  }
  return String(Math.round(value))
}

/** 去掉小数末尾的 0，例如 1.20 → 1.2、4.0 → 4。 */
function trimZero(text: string): string {
  return text.includes('.') ? text.replace(/\.?0+$/, '') : text
}

/** 百分比，保留整数。 */
export function formatPercent(value: number): string {
  return `${Math.min(100, Math.max(0, Math.round(value)))}%`
}

/** 相对时间：刚刚 / 3 分钟前 / 1 小时前 / 昨天 / 3 天前。 */
export function formatRelativeTime(iso: string | null | undefined): string {
  if (!iso) return '从未登录'
  const target = dayjs(iso)
  if (!target.isValid()) return '—'
  const diffMinutes = dayjs().diff(target, 'minute')
  if (diffMinutes < 1) return '刚刚'
  if (diffMinutes < 60) return `${diffMinutes} 分钟前`
  const diffHours = Math.floor(diffMinutes / 60)
  if (diffHours < 24) return `${diffHours} 小时前`
  const diffDays = Math.floor(diffHours / 24)
  if (diffDays === 1) return '昨天'
  if (diffDays < 30) return `${diffDays} 天前`
  return target.format('YYYY-MM-DD')
}

/** 变更时间：今天显示 HH:mm，昨天显示“昨日”，其余显示日期。 */
export function formatChangeTime(iso: string | null | undefined): string {
  if (!iso) return '—'
  const target = dayjs(iso)
  if (!target.isValid()) return '—'
  if (target.isSame(dayjs(), 'day')) return target.format('HH:mm')
  if (target.isSame(dayjs().subtract(1, 'day'), 'day')) return '昨日'
  return target.format('MM-DD')
}

/** 日期：2026-10-02。 */
export function formatDate(iso: string | null | undefined): string {
  if (!iso) return '—'
  return dayjs(iso).format('YYYY-MM-DD')
}

/** 时间：14:32。 */
export function formatTime(iso: string | null | undefined): string {
  if (!iso) return '—'
  return dayjs(iso).format('HH:mm')
}

/** 日期时间：2026-10-02 20:14。 */
export function formatDateTime(iso: string | null | undefined): string {
  if (!iso) return '—'
  return dayjs(iso).format('YYYY-MM-DD HH:mm')
}

/** 最近登录描述：3 分钟前 · 上海。 */
export function formatLastLogin(iso: string | null | undefined, city: string): string {
  if (!iso) return '从未登录'
  return `${formatRelativeTime(iso)}${city ? ` · ${city}` : ''}`
}

/** 字节数格式化为 GB / MB，用于存储用量。 */
export function formatBytes(bytes: number): string {
  if (!bytes) return '0 MB'
  const gb = bytes / 1024 / 1024 / 1024
  if (gb >= 1) return `${gb.toFixed(1)} GB`
  const mb = bytes / 1024 / 1024
  return `${mb.toFixed(mb >= 10 ? 0 : 1)} MB`
}

/** 套餐名称。 */
export function planLabel(plan: string): string {
  return PLAN_LABELS[plan] ?? plan
}

/** 角色名称。 */
export function roleLabel(role: string): string {
  return ROLE_LABELS[role] ?? role
}

/** 模型上下文窗口：128K / 8K。 */
export function formatContext(tokens: number): string {
  if (!tokens) return '—'
  if (tokens >= 1_000_000) {
    return `${trimZero((tokens / 1_000_000).toFixed(1))}M`
  }
  if (tokens >= 1000) return `${Math.floor(tokens / 1000)}K`
  return String(tokens)
}

/** 根据更新时间把会话分为今天 / 7 天内 / 30 天内 / 更早。 */
export function conversationGroup(iso: string): '今天' | '7 天内' | '30 天内' | '更早' {
  const target = dayjs(iso)
  const now = dayjs()
  if (target.isSame(now, 'day')) return '今天'
  const diffDays = now.diff(target, 'day')
  if (diffDays < 7) return '7 天内'
  if (diffDays < 30) return '30 天内'
  return '更早'
}
