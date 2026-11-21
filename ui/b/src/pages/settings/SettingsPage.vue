<script setup lang="ts">
/** 系统设置：注册开关、默认路由、存储与安全说明。 */
import { computed, onMounted, ref } from 'vue'
import { message, Modal } from 'ant-design-vue'
import {
  CloudServerOutlined,
  KeyOutlined,
  LockOutlined,
  ReloadOutlined,
  SafetyCertificateOutlined,
  TeamOutlined,
} from '@ant-design/icons-vue'
import { formatBytes, formatNumber, formatTokens, roleLabel, put } from '@deeptalk/shared'
import type { AdminModel, SettingsInfo } from '@deeptalk/shared'
import { fetchModels, fetchSettings, updateRouting, updateSettings } from '@/services/admin'
import { useAdminStore } from '@/stores/admin'
import { useUserStore } from '@/stores/user'
import PageHeader from '@/components/base/PageHeader.vue'

const admin = useAdminStore()
const userStore = useUserStore()

const loading = ref(true)
const settings = ref<SettingsInfo | null>(null)
const models = ref<AdminModel[]>([])
const defaultModelId = ref(0)
const fallbackModelId = ref<number | undefined>(undefined)
const maxOutputTokens = ref(384000)
const siteName = ref('')
const savingSiteName = ref(false)
const savingRegistration = ref(false)
const savingRouting = ref(false)
const savingBudget = ref(false)

const user = computed(() => userStore.user)
const storagePercent = computed(() => {
  if (!settings.value) return 0
  return Math.min(100, Math.round((settings.value.storage_used_bytes / settings.value.storage_total_bytes) * 100))
})
const enabledModels = computed(() => models.value.filter((item) => item.enabled))
const modelOptions = computed(() =>
  enabledModels.value.map((item) => ({ label: `${item.display_name} · ${item.provider_name}`, value: item.id })),
)
const fallbackOptions = computed(() => modelOptions.value.filter((item) => item.value !== defaultModelId.value))

/** 加载设置与模型。 */
async function load(showError = true): Promise<void> {
  loading.value = true
  try {
    const [settingsData, modelData] = await Promise.all([fetchSettings(), fetchModels()])
    settings.value = settingsData
    models.value = modelData.items
    defaultModelId.value = modelData.routing.default_model_id
    fallbackModelId.value = modelData.routing.fallback_model_id || undefined
    maxOutputTokens.value = settingsData.max_output_tokens
    siteName.value = settingsData.site_name || '' 
  } catch (error) {
    if (showError) message.error(error instanceof Error ? error.message : '设置加载失败')
  } finally {
    loading.value = false
  }
}

/** 注册开关变更。 */
function onRegistrationToggle(checked: unknown): void {
  const next = Boolean(checked)
  if (!next) {
    Modal.confirm({
      title: '关闭开放注册？',
      content: '关闭后注册页会隐藏表单，只有管理员可以在后台创建账号。',
      okText: '关闭注册',
      okType: 'danger',
      cancelText: '取消',
      onOk: () => applyRegistration(false),
    })
    return
  }
  void applyRegistration(true)
}

/** 保存注册开关。 */
async function applyRegistration(value: boolean): Promise<void> {
  if (!settings.value) return
  savingRegistration.value = true
  try {
    settings.value = await updateSettings({ allow_registration: value })
    message.success(value ? '已开放注册' : '已关闭注册')
    await admin.loadShell()
  } catch (error) {
    message.error(error instanceof Error ? error.message : '保存失败')
    await load(false)
  } finally {
    savingRegistration.value = false
  }
}

/** 保存单次输出预算（thinking + 正文共用）。 */
/** 保存站点名称。 */
async function saveSiteName(): Promise<void> {
  const name = siteName.value.trim()
  if (!name) {
    message.warning('站点名称不能为空')
    return
  }
  savingSiteName.value = true
  try {
    settings.value = await put<SettingsInfo>('/b/settings', { site_name: name })
    message.success('站点名称已保存')
  } catch (error) {
    message.error(error instanceof Error ? error.message : '保存失败')
  } finally {
    savingSiteName.value = false
  }
}

async function saveBudget(): Promise<void> {
  if (maxOutputTokens.value < 1024 || maxOutputTokens.value > 1_000_000) {
    message.warning('输出预算需在 1024–1,000,000 tokens 之间')
    return
  }
  savingBudget.value = true
  try {
    settings.value = await updateSettings({ max_output_tokens: maxOutputTokens.value })
    message.success('输出预算已保存，下一条消息生效')
  } catch (error) {
    message.error(error instanceof Error ? error.message : '保存失败')
    await load(false)
  } finally {
    savingBudget.value = false
  }
}

/** 保存默认路由。 */
async function saveRouting(): Promise<void> {
  if (!defaultModelId.value) {
    message.warning('请选择默认模型')
    return
  }
  if (fallbackModelId.value && fallbackModelId.value === defaultModelId.value) {
    message.warning('兜底模型不能与默认模型相同')
    return
  }
  savingRouting.value = true
  try {
    await updateRouting({
      default_model_id: defaultModelId.value,
      fallback_model_id: fallbackModelId.value || 0,
    })
    message.success('默认路由已保存')
    await admin.loadShell()
  } catch (error) {
    message.error(error instanceof Error ? error.message : '保存失败')
    await load(false)
  } finally {
    savingRouting.value = false
  }
}

onMounted(async () => {
  await load()
})
</script>

<template>
  <div class="settings-page">
    <PageHeader title="系统设置" subtitle="注册策略、默认模型路由与运行信息。">
      <template #actions>
        <a-button :loading="loading" @click="load()">
          <ReloadOutlined />
          刷新
        </a-button>
      </template>
    </PageHeader>

    <div v-if="loading && !settings" class="panel loading-panel">
      <a-skeleton :paragraph="{ rows: 8 }" active />
    </div>

    <div v-else class="settings-grid">
      <div class="main-col">
        <section class="panel">
          <div class="panel-head">
            <div>
              <h2>注册与账号</h2>
              <p>控制 C 端自助注册；关闭后仍可由管理员创建账号。</p>
            </div>
            <span class="status-pill" :class="{ off: !settings?.allow_registration }">
              <i />
              {{ settings?.allow_registration ? '开放注册' : '已关闭注册' }}
            </span>
          </div>
          <div class="setting-row">
            <div class="setting-copy">
              <strong>允许自助注册</strong>
              <p>注册即登录，无验证邮件；首个注册账号为平台管理员。</p>
            </div>
            <a-switch
              :checked="settings?.allow_registration"
              :loading="savingRegistration"
              aria-label="开放注册开关"
              @change="onRegistrationToggle"
            />
          </div>
          <div class="inline-note">
            <TeamOutlined />
            当前共 {{ formatNumber(admin.overview?.total_users ?? 0) }} 个账号，本月新增
            {{ admin.overview?.new_this_month ?? 0 }} 个。
          </div>
        </section>

        <section class="panel">
          <div class="panel-head">
            <div>
              <h2>默认路由</h2>
              <p>默认模型失败或超时时，仅触发一次兜底模型。</p>
            </div>
            <a-button type="primary" size="small" :loading="savingRouting" @click="saveRouting">保存路由</a-button>
          </div>
          <div class="routing-grid">
            <div>
              <label class="field-label">默认模型</label>
              <a-select
                v-model:value="defaultModelId"
                :options="modelOptions"
                placeholder="选择默认模型"
                class="routing-select"
              />
              <p class="field-hint">新用户与未指定模型的请求默认使用该模型。</p>
            </div>
            <div>
              <label class="field-label">兜底模型</label>
              <a-select
                v-model:value="fallbackModelId"
                :options="fallbackOptions"
                placeholder="不设置兜底"
                allow-clear
                class="routing-select"
              />
              <p class="field-hint">停用中的模型不可设为默认或兜底。</p>
            </div>
          </div>
          <div class="budget-row">
            <div class="budget-copy">
              <label class="field-label">单次输出预算（tokens）</label>
              <p class="field-hint">深度思考与正文共用这份预算；默认 384000（384K），最大 1,000,000。</p>
            </div>
            <div class="budget-control">
              <a-input-number
                v-model:value="maxOutputTokens"
                :min="1024"
                :max="1000000"
                :step="1024"
                style="width: 150px"
              />
              <a-button type="primary" ghost :loading="savingBudget" @click="saveBudget">保存预算</a-button>
            </div>
          </div>
        </section>

        <section class="panel">
          <div class="panel-head">
            <div>
              <h2>安全说明</h2>
              <p>MVP 的安全边界与数据策略。</p>
            </div>
            <SafetyCertificateOutlined class="panel-icon" />
          </div>
          <div class="security-list">
            <div class="security-item">
              <span class="security-ico"><KeyOutlined /></span>
              <div>
                <strong>上游密钥加密</strong>
                <p>API Key 使用 AES-GCM 加密入库，接口只返回掩码，前端永不回显。</p>
              </div>
            </div>
            <div class="security-item">
              <span class="security-ico"><LockOutlined /></span>
              <div>
                <strong>会话与密码</strong>
                <p>密码使用 Argon2id 哈希；会话 HttpOnly、SameSite=Lax，默认 7 天。</p>
              </div>
            </div>
            <div class="security-item">
              <span class="security-ico"><TeamOutlined /></span>
              <div>
                <strong>隐私边界</strong>
                <p>后台不读取对话正文；CSV 只导出账号字段，管理操作写入审计日志。</p>
              </div>
            </div>
          </div>
        </section>
      </div>

      <aside class="side-col">
        <section class="panel">
          <div class="panel-head">
            <div>
              <h2>存储与运行</h2>
              <p>单实例 SQLite 部署</p>
            </div>
            <CloudServerOutlined class="panel-icon" />
          </div>
          <div class="storage-bar">
            <span :style="{ width: `${Math.max(storagePercent, 1)}%` }" />
          </div>
          <div class="storage-meta">
            <strong>{{ storagePercent }}%</strong>
            <span>
              {{ formatBytes(settings?.storage_used_bytes ?? 0) }} /
              {{ formatBytes(settings?.storage_total_bytes ?? 0) }}
            </span>
          </div>
          <div class="site-name-row">
            <label class="field-label">站点名称</label>
            <a-input
              v-model:value="siteName"
              :maxlength="32"
              placeholder="例如：小橘树洞"
              class="site-name-input"
            />
            <a-button type="primary" size="small" :loading="savingSiteName" @click="saveSiteName">保存名称</a-button>
          </div>
          <div class="info-rows">
            <div><span>站点名称</span><strong>{{ settings?.site_name ?? '小橘树洞' }}</strong></div>
            <div><span>版本</span><strong>{{ settings?.version ?? 'v1.0.0' }}</strong></div>
            <div><span>数据库</span><strong>SQLite · WAL</strong></div>
            <div><span>监听</span><strong>0.0.0.0:9821</strong></div>
            <div><span>本月对话</span><strong>{{ formatNumber(admin.overview?.monthly_messages ?? 0) }} 条消息</strong></div>
            <div><span>平台 tokens</span><strong>{{ formatTokens(admin.overview?.monthly_tokens ?? 0) }}</strong></div>
          </div>
        </section>

        <section class="panel">
          <div class="panel-head">
            <div>
              <h2>当前管理员</h2>
              <p>唯一管理员账号</p>
            </div>
          </div>
          <div class="admin-card">
            <span class="avatar">{{ user?.avatar_text ?? 'A' }}</span>
            <div>
              <strong>{{ user?.nickname }}</strong>
              <span>{{ user?.email }}</span>
            </div>
          </div>
          <div class="info-rows">
            <div><span>角色</span><strong>{{ roleLabel(user?.role ?? 'admin') }}</strong></div>
            <div><span>最近登录</span><strong>{{ user?.last_login_city || '本地' }}</strong></div>
            <div><span>账号 ID</span><strong class="mono">usr_{{ user?.id?.toString().padStart(4, '0') }}</strong></div>
          </div>
        </section>

        <section class="panel note-panel">
          <p>
            <SafetyCertificateOutlined />
            修改注册开关、默认路由与危险操作都会写入审计日志，可在「厂商与模型」页查看最近变更。
          </p>
        </section>
      </aside>
    </div>
  </div>
</template>

<style scoped>
.settings-page {
  max-width: 1180px;
}

.settings-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.5fr) minmax(300px, 1fr);
  gap: 16px;
  align-items: start;
}

.main-col,
.side-col {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
}

.panel {
  padding: 16px 18px;
  border: 1px solid var(--dt-line);
  border-radius: var(--dt-radius-card);
  background: var(--dt-surface);
  box-shadow: var(--dt-shadow-card);
}

.loading-panel {
  min-height: 260px;
}

.panel-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
}

.panel-head h2 {
  margin: 0 0 3px;
  color: var(--dt-text);
  font-size: 13px;
  font-weight: 650;
}

.panel-head p {
  margin: 0;
  color: var(--dt-text-faint);
  font-size: 10.2px;
}

.panel-icon {
  color: var(--dt-text-faint);
  font-size: 16px;
}

.status-pill {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  height: 21px;
  padding: 0 8px;
  border-radius: 999px;
  background: var(--dt-success-soft);
  color: var(--dt-success-text);
  font-size: 9.4px;
  font-weight: 600;
  white-space: nowrap;
}

.status-pill i {
  width: 4px;
  height: 4px;
  border-radius: 50%;
  background: currentColor;
}

.status-pill.off {
  background: var(--dt-hover);
  color: var(--dt-text-muted);
}

.setting-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  padding: 12px 14px;
  border: 1px solid var(--dt-line);
  border-radius: 10px;
  background: var(--dt-sidebar);
}

.setting-copy strong {
  color: var(--dt-text-secondary);
  font-size: 11.8px;
  font-weight: 620;
}

.setting-copy p {
  margin: 2px 0 0;
  color: var(--dt-text-faint);
  font-size: 9.8px;
  line-height: 15px;
}

.inline-note {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 10px;
  color: var(--dt-text-faint);
  font-size: 9.6px;
}

.budget-row {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 16px;
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid var(--dt-line-soft);
}

.budget-copy {
  min-width: 0;
}

.budget-control {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 0 0 auto;
}

.routing-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}

.field-label {
  display: block;
  margin-bottom: 6px;
  color: var(--dt-text-secondary);
  font-size: 10.6px;
  font-weight: 600;
}

.routing-select {
  width: 100%;
}

.routing-select :deep(.ant-select-selector) {
  height: 36px !important;
  border-radius: 9px !important;
  font-size: 11.4px;
}

.field-hint {
  margin: 5px 0 0;
  color: var(--dt-text-faint);
  font-size: 9.4px;
  line-height: 14px;
}

.security-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.security-item {
  display: flex;
  align-items: flex-start;
  gap: 10px;
}

.security-ico {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: 8px;
  background: var(--dt-brand-softer);
  border: 1px solid var(--dt-line);
  color: var(--dt-brand);
  font-size: 12px;
  flex: 0 0 auto;
}

.security-item strong {
  color: var(--dt-text-secondary);
  font-size: 11.2px;
  font-weight: 620;
}

.security-item p {
  margin: 2px 0 0;
  color: var(--dt-text-faint);
  font-size: 9.6px;
  line-height: 15px;
}

.storage-bar {
  height: 6px;
  border-radius: 999px;
  background: var(--dt-line);
  overflow: hidden;
}

.storage-bar span {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: var(--dt-brand);
}

.storage-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 6px;
  color: var(--dt-text-faint);
  font-size: 9.6px;
}

.storage-meta strong {
  color: var(--dt-brand);
  font-size: 11px;
}

.info-rows {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: 12px;
  padding-top: 11px;
  border-top: 1px solid var(--dt-line-soft);
}

.info-rows div {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  font-size: 10px;
}

.info-rows span {
  color: var(--dt-text-faint);
}

.info-rows strong {
  color: var(--dt-text-secondary);
  font-size: 10px;
  font-weight: 550;
  text-align: right;
}

.info-rows strong.mono {
  font-family: var(--dt-mono);
  font-size: 9.4px;
}

.admin-card {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border: 1px solid var(--dt-line-soft);
  border-radius: 10px;
  background: var(--dt-sidebar);
}

.avatar {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  border-radius: 50%;
  background: var(--dt-brand-soft);
  color: var(--dt-brand-hover);
  font-size: 13px;
  font-weight: 700;
}

.admin-card div {
  display: flex;
  flex-direction: column;
  gap: 1px;
  min-width: 0;
}

.admin-card strong {
  color: var(--dt-text);
  font-size: 12px;
  font-weight: 620;
}

.admin-card span {
  color: var(--dt-text-faint);
  font-size: 9.6px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.note-panel {
  background: var(--dt-brand-softer);
  border-color: var(--dt-line);
}

.note-panel p {
  display: flex;
  align-items: flex-start;
  gap: 7px;
  margin: 0;
  color: var(--dt-text-muted);
  font-size: 9.8px;
  line-height: 16px;
}

.note-panel p :deep(svg) {
  color: var(--dt-brand);
  margin-top: 1px;
}

@media (max-width: 1024px) {
  .settings-grid {
    grid-template-columns: 1fr;
  }

  .routing-grid {
    grid-template-columns: 1fr;
    gap: 12px;
  }

  .budget-row {
    align-items: flex-start;
    flex-direction: column;
  }
}

.site-name-row {
  display: flex;
  align-items: center;
  gap: 10px;
  margin: 4px 0 16px;
}

.site-name-input {
  max-width: 240px;
}
</style>
