<script setup lang="ts">
/**
 * 用户详情栏：账号信息、套餐额度与安全处置。
 * 隐私边界：这里只展示账号字段与用量，不读取任何对话正文。
 */
import { computed } from 'vue'
import { KeyOutlined, LockOutlined, LogoutOutlined, TeamOutlined, UserOutlined } from '@ant-design/icons-vue'
import { formatDate, formatLastLogin, formatPercent, formatTokens, planLabel, roleLabel } from '@deeptalk/shared'
import type { User } from '@deeptalk/shared'

const props = defineProps<{ user: User | null; loading: boolean; busy: boolean }>()
const emit = defineEmits<{
  (event: 'quota'): void
  (event: 'reset'): void
  (event: 'logout-all'): void
  (event: 'toggle-disabled', disabled: boolean): void
  (event: 'change-plan', plan: 'free' | 'pro' | 'team' | 'unlimited'): void
  (event: 'close'): void
}>()

const planOptions = [
  { label: '免费版 · 4M tokens / 月', value: 'free' },
  { label: '专业版 · 20M tokens / 月', value: 'pro' },
  { label: '团队版 · 50M tokens / 月', value: 'team' },
  { label: '无限版 · 不限 tokens', value: 'unlimited' },
]

const user = computed(() => props.user)
const isAdmin = computed(() => user.value?.role === 'admin')
const usedPercent = computed(() => user.value?.used_percent ?? 0)
const unlimited = computed(() => user.value?.plan === 'unlimited' || (user.value?.quota_tokens ?? 0) <= 0)
const status = computed(() => {
  if (!user.value) return { label: '—', tone: 'normal' }
  if (user.value.disabled) return { label: '已限制', tone: 'danger' }
  if (!unlimited.value && usedPercent.value >= 90) return { label: '额度将尽', tone: 'warning' }
  return { label: '正常', tone: 'normal' }
})

/** 套餐切换：父组件负责确认与请求。 */
function onPlanChange(value: 'free' | 'pro' | 'team' | 'unlimited'): void {
  emit('change-plan', value)
}

/** 限制登录开关：父组件负责二次确认。 */
function onDisabledChange(value: unknown): void {
  emit('toggle-disabled', Boolean(value))
}
</script>

<template>
  <aside class="user-detail">
    <div v-if="loading && !user" class="detail-loading">
      <a-skeleton avatar :paragraph="{ rows: 8 }" active />
    </div>

    <div v-else-if="!user" class="detail-empty">
      <span class="empty-icon"><UserOutlined /></span>
      <p>选择一位用户</p>
      <small>在左侧列表点击账号，查看额度与访问状态。</small>
    </div>

    <template v-else>
      <header class="detail-head">
        <span class="avatar">{{ user.avatar_text }}</span>
        <div class="identity">
          <strong>{{ user.nickname }}</strong>
          <span>{{ user.email }}</span>
        </div>
        <span class="status-pill" :class="status.tone">
          <i />
          {{ status.label }}
        </span>
      </header>

      <div class="detail-body">
        <section class="detail-section">
          <h3>账号</h3>
          <div class="row">
            <span>角色</span>
            <strong>{{ roleLabel(user.role) }}</strong>
          </div>
          <div class="row">
            <span>注册时间</span>
            <strong>{{ formatDate(user.created_at) }}</strong>
          </div>
          <div class="row">
            <span>最近登录</span>
            <strong>{{ formatLastLogin(user.last_login_at, user.last_login_city) }}</strong>
          </div>
          <div class="row">
            <span>登录 IP</span>
            <strong class="mono">{{ user.last_login_ip || '—' }}</strong>
          </div>
          <div class="row">
            <span>会话 / 消息</span>
            <strong>{{ user.conversation_count }} / {{ user.message_count }}</strong>
          </div>
        </section>

        <div class="divider" />

        <section class="detail-section">
          <div class="section-head">
            <h3>套餐与额度</h3>
            <span class="plan-current">{{ planLabel(user.plan) }}</span>
          </div>
          <a-select
            :value="user.plan"
            :options="planOptions"
            class="plan-select"
            :disabled="busy"
            aria-label="调整套餐"
            @change="onPlanChange"
          />
          <div v-if="!unlimited" class="quota-bar">
            <span :style="{ width: `${Math.min(100, usedPercent)}%` }" :class="{ warning: usedPercent >= 90 }" />
          </div>
          <div v-else class="quota-unlimited">不限月度 tokens</div>
          <div class="quota-meta">
            <span>已用 {{ formatTokens(user.used_tokens) }} / {{ unlimited ? '无限' : formatTokens(user.quota_tokens) }}</span>
            <span>{{ unlimited ? '不限' : formatPercent(usedPercent) }}</span>
          </div>
          <div class="quota-foot">
            <span>{{ unlimited ? '不设月度上限' : `重置 ${user.quota_reset_days || 0} 天` }}</span>
            <button v-if="!unlimited" type="button" class="link" :disabled="busy" @click="emit('quota')">调整额度 ›</button>
          </div>
        </section>

        <div class="divider" />

        <section class="detail-section">
          <h3>安全</h3>
          <div class="switch-row">
            <div class="switch-copy">
              <span>限制登录</span>
              <small>开启后已签发会话立即失效</small>
            </div>
            <a-switch
              :checked="user.disabled"
              :disabled="isAdmin || busy"
              :title="isAdmin ? '管理员账号不可被限制' : ''"
              @change="onDisabledChange"
            />
          </div>
          <a-button class="reset-btn" block :disabled="busy" @click="emit('reset')">
            <KeyOutlined />
            重置密码
          </a-button>
          <button type="button" class="logout-link" :disabled="busy" @click="emit('logout-all')">
            <LogoutOutlined />
            强制退出全部设备
          </button>
          <p class="security-hint">
            <LockOutlined />
            管理员操作会记录到审计日志
          </p>
        </section>

        <div class="detail-foot">
          <span><TeamOutlined /> 按账号隔离，不展示对话正文</span>
          <button type="button" class="done-btn" @click="emit('close')">完成</button>
        </div>
      </div>
    </template>
  </aside>
</template>

<style scoped>
.user-detail {
  min-height: 420px;
  border: 1px solid var(--dt-line);
  border-radius: var(--dt-radius-card);
  background: #fff;
  box-shadow: var(--dt-shadow-card);
  overflow: hidden;
}

.detail-loading {
  padding: 18px;
}

.detail-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-height: 420px;
  padding: 24px;
  text-align: center;
}

.empty-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  margin-bottom: 4px;
  border-radius: 11px;
  background: var(--dt-brand-soft);
  color: var(--dt-brand);
  font-size: 17px;
}

.detail-empty p {
  margin: 0;
  color: var(--dt-text-secondary);
  font-size: 12.4px;
  font-weight: 600;
}

.detail-empty small {
  color: var(--dt-text-faint);
  font-size: 9.8px;
  max-width: 190px;
  line-height: 15px;
}

.detail-head {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 14px 16px;
  border-bottom: 1px solid var(--dt-line-soft);
}

.avatar {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 38px;
  height: 38px;
  border-radius: 50%;
  background: var(--dt-brand-soft);
  color: var(--dt-brand-hover);
  font-size: 14px;
  font-weight: 700;
  flex: 0 0 auto;
}

.identity {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
  flex: 1;
}

.identity strong {
  color: var(--dt-text);
  font-size: 13px;
  font-weight: 650;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.identity span {
  color: var(--dt-text-faint);
  font-size: 9.8px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.status-pill {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: 20px;
  padding: 0 7px;
  border-radius: 999px;
  font-size: 9.2px;
  font-weight: 600;
  white-space: nowrap;
}

.status-pill i {
  width: 4px;
  height: 4px;
  border-radius: 50%;
  background: currentColor;
}

.status-pill.normal {
  background: var(--dt-success-soft);
  color: var(--dt-success-text);
}

.status-pill.warning {
  background: var(--dt-warning-soft);
  color: var(--dt-warning-text);
}

.status-pill.danger {
  background: var(--dt-danger-soft);
  color: var(--dt-danger-text);
}

.detail-body {
  padding: 4px 16px 12px;
}

.detail-section {
  padding: 12px 0;
}

.detail-section h3 {
  margin: 0 0 9px;
  color: var(--dt-text-muted);
  font-size: 10.2px;
  font-weight: 650;
  letter-spacing: 0.2px;
}

.section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.plan-current {
  color: var(--dt-text-faint);
  font-size: 9.4px;
}

.row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 5px 0;
  font-size: 10.4px;
}

.row span {
  color: var(--dt-text-faint);
  flex: 0 0 auto;
}

.row strong {
  color: var(--dt-text-secondary);
  font-size: 10.4px;
  font-weight: 550;
  text-align: right;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.row strong.mono {
  font-family: var(--dt-mono);
  font-size: 9.4px;
}

.divider {
  height: 1px;
  background: var(--dt-line-soft);
}

.plan-select {
  width: 100%;
  margin: 2px 0 9px;
}

.plan-select :deep(.ant-select-selector) {
  height: 32px !important;
  border-radius: 8px !important;
  font-size: 10.8px;
}

.quota-unlimited {
  margin-top: 10px;
  padding: 7px 9px;
  border-radius: 7px;
  background: #fff7ed;
  color: #b93815;
  font-size: 10.5px;
  text-align: center;
}

.quota-bar {
  height: 5px;
  border-radius: 999px;
  background: #eef0f4;
  overflow: hidden;
}

.quota-bar span {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: var(--dt-brand);
}

.quota-bar span.warning {
  background: var(--dt-warning);
}

.quota-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 5px;
  color: var(--dt-text-muted);
  font-size: 9.4px;
}

.quota-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 7px;
  color: var(--dt-text-faint);
  font-size: 9.2px;
}

.link {
  padding: 0;
  border: none;
  background: transparent;
  color: var(--dt-brand);
  font-size: 9.8px;
  cursor: pointer;
}

.link:hover {
  color: var(--dt-brand-hover);
}

.switch-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 8px 0 9px;
}

.switch-copy {
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.switch-copy span {
  color: var(--dt-text-secondary);
  font-size: 10.6px;
  font-weight: 550;
}

.switch-copy small {
  color: var(--dt-text-faint);
  font-size: 9.2px;
}

.reset-btn {
  height: 32px;
  margin: 3px 0 8px;
  border-radius: 8px;
  font-size: 10.8px;
}

.logout-link {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 0;
  border: none;
  background: transparent;
  color: var(--dt-text-muted);
  font-size: 9.8px;
  cursor: pointer;
}

.logout-link:hover {
  color: var(--dt-danger-text);
}

.security-hint {
  display: flex;
  align-items: center;
  gap: 5px;
  margin: 10px 0 0;
  padding-top: 9px;
  border-top: 1px solid var(--dt-line-soft);
  color: var(--dt-text-faint);
  font-size: 9px;
}

.detail-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding-top: 10px;
  border-top: 1px solid var(--dt-line-soft);
  color: var(--dt-text-faint);
  font-size: 8.8px;
}

.detail-foot span {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.done-btn {
  border: none;
  background: transparent;
  color: var(--dt-brand);
  font-size: 10.2px;
  cursor: pointer;
  white-space: nowrap;
}
</style>
