<script setup lang="ts">
/**
 * 厂商与模型：上游接入、密钥、模型路由、默认兜底与接入健康。
 * 一屏内完成“看状态 -> 改配置 -> 测连通 -> 调路由”的闭环。
 */
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { message, Modal } from 'ant-design-vue'
import {
  ApiOutlined,
  LeftOutlined,
  PlusOutlined,
  ReloadOutlined,
  RightOutlined,
  SafetyCertificateOutlined,
} from '@ant-design/icons-vue'
import { formatChangeTime, formatContext } from '@deeptalk/shared'
import type { AdminModel, AuditItem, HealthInfo, ProviderInfo, Routing } from '@deeptalk/shared'
import {
  fetchAuditLogs,
  fetchHealth,
  fetchModels,
  fetchProviders,
  testProvider,
  toggleModel,
  toggleProvider,
  updateRouting,
} from '@/services/admin'
import { useAdminStore } from '@/stores/admin'
import PageHeader from '@/components/base/PageHeader.vue'
import ProviderCard from '@/components/business/ProviderCard.vue'
import ProviderFormModal from '@/components/business/ProviderFormModal.vue'
import ModelFormModal from '@/components/business/ModelFormModal.vue'

const route = useRoute()
const router = useRouter()
const admin = useAdminStore()

const loading = ref(true)
const providers = ref<ProviderInfo[]>([])
const counts = ref({ all: 0, builtin: 0, custom: 0, enabled: 0 })
const health = ref<HealthInfo | null>(null)
const models = ref<AdminModel[]>([])
const routing = ref<Routing>({ default_model_id: 0, fallback_model_id: 0 })
const audits = ref<AuditItem[]>([])
const allAudits = ref<AuditItem[]>([])

const activeFilter = ref<'all' | 'builtin' | 'custom'>('all')
const keyword = ref('')
const cardScroller = ref<HTMLElement | null>(null)
const testingId = ref(0)
const testingAll = ref(false)
const routingSaving = ref(false)

const formOpen = ref(false)
const editingProvider = ref<ProviderInfo | null>(null)
const modelFormOpen = ref(false)
const editingModel = ref<AdminModel | null>(null)
const auditOpen = ref(false)

let healthTimer: ReturnType<typeof setInterval> | null = null

const tabs = computed(() => [
  { key: 'all' as const, label: '全部', count: counts.value.all },
  { key: 'builtin' as const, label: '预置', count: counts.value.builtin },
  { key: 'custom' as const, label: '自定义', count: counts.value.custom },
])

const enabledModels = computed(() => models.value.filter((item) => item.enabled))
const enabledModelCount = computed(() => models.value.filter((item) => item.enabled).length)
const modelOptions = computed(() =>
  enabledModels.value.map((item) => ({
    label: `${item.display_name} · ${item.provider_name}`,
    value: item.id,
  })),
)
const fallbackOptions = computed(() => modelOptions.value.filter((item) => item.value !== routing.value.default_model_id))

/** 按标签页与关键词筛选卡片；服务端一次返回全部，厂商数量有限，本地筛选足够快。 */
const filteredProviders = computed(() => {
  const term = keyword.value.trim().toLowerCase()
  return providers.value.filter((provider) => {
    if (activeFilter.value === 'builtin' && !provider.is_builtin) return false
    if (activeFilter.value === 'custom' && provider.is_builtin) return false
    if (!term) return true
    return (
      provider.name.toLowerCase().includes(term) ||
      provider.base_url.toLowerCase().includes(term) ||
      provider.code.toLowerCase().includes(term) ||
      provider.models.some((model) => model.model_id.toLowerCase().includes(term))
    )
  })
})

const showCardArrows = computed(() => filteredProviders.value.length > 3)

/** 加载页面全部数据。 */
async function load(silent = false): Promise<void> {
  if (!silent) loading.value = true
  try {
    const [providerData, modelData, auditData] = await Promise.all([
      fetchProviders('all'),
      fetchModels(),
      fetchAuditLogs(6),
    ])
    providers.value = providerData.items
    counts.value = providerData.counts
    health.value = providerData.health
    models.value = modelData.items
    routing.value = modelData.routing
    audits.value = auditData.items
  } catch (error) {
    if (!silent) message.error(error instanceof Error ? error.message : '加载失败，请稍后重试')
  } finally {
    loading.value = false
  }
}

/** 30 秒刷新接入健康。 */
async function refreshHealth(silent = true): Promise<void> {
  try {
    health.value = await fetchHealth()
  } catch {
    if (!silent) message.error('健康状态刷新失败')
  }
}

/** 切换标签。 */
function setFilter(key: 'all' | 'builtin' | 'custom'): void {
  activeFilter.value = key
  cardScroller.value?.scrollTo({ left: 0, behavior: 'smooth' })
}

/** 横向翻页卡片。 */
function scrollCards(direction: 1 | -1): void {
  cardScroller.value?.scrollBy({ left: direction * 360, behavior: 'smooth' })
}

/** 清空搜索。 */
function clearSearch(): void {
  keyword.value = ''
  void router.replace({ path: '/providers' })
}

/** 打开新建厂商。 */
function openCreate(): void {
  editingProvider.value = null
  formOpen.value = true
}

/** 打开厂商配置。 */
function openEdit(provider: ProviderInfo): void {
  editingProvider.value = provider
  formOpen.value = true
}

/** 启停厂商；关闭需要二次确认。 */
function onToggle(provider: ProviderInfo, enabled: boolean): void {
  if (enabled) {
    void applyToggle(provider, true)
    return
  }
  Modal.confirm({
    title: `停用「${provider.name}」？`,
    content: '停用后该厂商的模型不再下发给 C 端，用户将无法再选择。',
    okText: '停用',
    okType: 'danger',
    cancelText: '取消',
    onOk: () => applyToggle(provider, false),
  })
}

/** 执行启停请求。 */
async function applyToggle(provider: ProviderInfo, enabled: boolean): Promise<void> {
  testingId.value = provider.id
  try {
    await toggleProvider(provider.id, enabled)
    message.success(enabled ? '厂商已启用' : '厂商已停用')
    await load(true)
    await admin.refreshOverview()
  } catch (error) {
    message.error(error instanceof Error ? error.message : '操作失败')
  } finally {
    testingId.value = 0
  }
}

/** 测试单个厂商。 */
async function onTest(provider: ProviderInfo): Promise<void> {
  testingId.value = provider.id
  try {
    const result = await testProvider(provider.id)
    if (result.ok) {
      message.success(`${provider.name}：${result.message}（${result.latency_ms} ms）`)
    } else {
      message.error(`${provider.name}：${result.message}`)
    }
    await load(true)
  } catch (error) {
    message.error(error instanceof Error ? error.message : '测试失败')
  } finally {
    testingId.value = 0
  }
}

/** 批量测试所有启用厂商。 */
async function testAll(): Promise<void> {
  const targets = providers.value.filter((item) => item.enabled)
  if (!targets.length) {
    message.warning('没有已启用的厂商')
    return
  }
  testingAll.value = true
  let ok = 0
  for (const provider of targets) {
    try {
      const result = await testProvider(provider.id)
      if (result.ok) ok += 1
    } catch {
      // 单个失败不阻断后续测试
    }
  }
  testingAll.value = false
  await load(true)
  message.success(`连通性测试完成：${ok} / ${targets.length} 个上游可用`)
}

/** 默认模型下拉变化。 */
function onDefaultModelChange(value: number): void {
  void saveRouting('default', value)
}

/** 兜底模型下拉变化；清空时传 0。 */
function onFallbackModelChange(value?: number): void {
  void saveRouting('fallback', value ?? 0)
}

/** 更新默认 / 兜底路由。 */
async function saveRouting(which: 'default' | 'fallback', value: number | undefined): Promise<void> {
  const next: Routing = {
    default_model_id: which === 'default' ? Number(value ?? 0) : routing.value.default_model_id,
    fallback_model_id: which === 'fallback' ? Number(value ?? 0) : routing.value.fallback_model_id,
  }
  if (!next.default_model_id) {
    message.warning('默认模型不能为空')
    return
  }
  if (next.fallback_model_id && next.fallback_model_id === next.default_model_id) {
    message.warning('兜底模型不能与默认模型相同')
    return
  }
  routingSaving.value = true
  try {
    routing.value = await updateRouting(next)
    message.success('默认路由已更新')
    await load(true)
  } catch (error) {
    message.error(error instanceof Error ? error.message : '路由更新失败')
    await load(true)
  } finally {
    routingSaving.value = false
  }
}

/** 启停模型；停用默认 / 兜底模型时提示会自动修复。 */
function onModelToggle(model: AdminModel, enabled: boolean): void {
  if (enabled) {
    void applyModelToggle(model, true)
    return
  }
  Modal.confirm({
    title: `停用模型「${model.model_id}」？`,
    content:
      model.is_default || model.is_fallback
        ? '该模型正在承担默认或兜底路由，停用后系统会自动切换到其他启用模型。'
        : '停用后用户侧将不再出现该模型。',
    okText: '停用',
    okType: 'danger',
    cancelText: '取消',
    onOk: () => applyModelToggle(model, false),
  })
}

/** 执行模型启停。 */
async function applyModelToggle(model: AdminModel, enabled: boolean): Promise<void> {
  try {
    await toggleModel(model.id, enabled)
    message.success(enabled ? '模型已启用' : '模型已停用')
    await load(true)
    await admin.refreshOverview()
  } catch (error) {
    message.error(error instanceof Error ? error.message : '操作失败')
  }
}

/** 模型开关变化。 */
function onModelSwitch(model: AdminModel, checked: unknown): void {
  onModelToggle(model, Boolean(checked))
}

/** 打开模型编辑。 */
function openModelEdit(model: AdminModel): void {
  editingModel.value = model
  modelFormOpen.value = true
}

/** 打开新增模型。 */
function openModelCreate(): void {
  editingModel.value = null
  modelFormOpen.value = true
}

/** 打开全部审计记录。 */
async function openAudits(): Promise<void> {
  auditOpen.value = true
  try {
    const data = await fetchAuditLogs(30)
    allAudits.value = data.items
  } catch {
    allAudits.value = audits.value
  }
}

/** 厂商保存后刷新。 */
async function onProviderSaved(): Promise<void> {
  await load(true)
  await admin.refreshOverview()
}

/** 模型保存后刷新。 */
async function onModelSaved(): Promise<void> {
  await load(true)
  await admin.refreshOverview()
}

watch(
  () => route.query.q,
  (value) => {
    keyword.value = typeof value === 'string' ? value : ''
  },
  { immediate: true },
)

onMounted(async () => {
  await load()
  healthTimer = setInterval(() => {
    if (document.visibilityState === 'visible') void refreshHealth()
  }, 30_000)
})

onBeforeUnmount(() => {
  if (healthTimer) clearInterval(healthTimer)
})
</script>

<template>
  <div class="providers-page">
    <PageHeader
      title="厂商与模型"
      subtitle="统一管理上游接入、密钥与默认路由；Key 只存服务端，前端永不回显。"
    >
      <template #actions>
        <a-button :loading="testingAll" @click="testAll">
          <ApiOutlined />
          测试连通性
        </a-button>
        <a-button type="primary" @click="openCreate">
          <PlusOutlined />
          新建厂商
        </a-button>
      </template>
    </PageHeader>

    <div class="tabs-row">
      <div class="filter-tabs" role="tablist" aria-label="厂商筛选">
        <button
          v-for="tab in tabs"
          :key="tab.key"
          type="button"
          class="filter-tab"
          :class="{ active: activeFilter === tab.key }"
          role="tab"
          :aria-selected="activeFilter === tab.key"
          @click="setFilter(tab.key)"
        >
          {{ tab.label }} <span>{{ tab.count }}</span>
        </button>
      </div>
      <div v-if="keyword" class="search-chip">
        搜索：{{ keyword }}
        <button type="button" aria-label="清除搜索" @click="clearSearch">×</button>
      </div>
      <div v-if="showCardArrows" class="card-arrows">
        <button type="button" aria-label="上一页厂商" @click="scrollCards(-1)"><LeftOutlined /></button>
        <button type="button" aria-label="下一页厂商" @click="scrollCards(1)"><RightOutlined /></button>
      </div>
    </div>

    <div v-if="loading" class="loading-grid">
      <a-skeleton v-for="index in 3" :key="index" avatar :paragraph="{ rows: 2 }" active />
    </div>

    <template v-else>
      <div v-if="filteredProviders.length" ref="cardScroller" class="provider-row">
        <ProviderCard
          v-for="provider in filteredProviders"
          :key="provider.id"
          :provider="provider"
          :testing="testingId === provider.id"
          @edit="openEdit"
          @toggle="onToggle"
          @test="onTest"
        />
      </div>

      <section v-else class="panel empty-panel">
        <span class="empty-icon"><ApiOutlined /></span>
        <h3>没有匹配的厂商</h3>
        <p>调整筛选条件，或接入一个新的 OpenAI 兼容服务。</p>
        <div class="empty-actions">
          <a-button v-if="keyword" @click="clearSearch">清除搜索</a-button>
          <a-button type="primary" @click="openCreate">
            <PlusOutlined />
            新建厂商
          </a-button>
        </div>
      </section>

      <div class="content-grid">
        <div class="left-col">
          <section class="panel model-panel">
            <div class="panel-head">
              <div>
                <h2>模型路由</h2>
                <p>共 {{ models.length }} 个模型 · {{ enabledModelCount }} 个已启用</p>
              </div>
              <a-button size="small" @click="openModelCreate">
                <PlusOutlined />
                添加模型
              </a-button>
            </div>

            <div class="model-table-wrap">
              <table class="model-table">
                <thead>
                  <tr>
                    <th class="col-model">模型</th>
                    <th class="col-provider">厂商</th>
                    <th class="col-context">上下文</th>
                    <th class="col-concurrency">并发</th>
                    <th class="col-status">状态</th>
                  </tr>
                </thead>
                <tbody>
                  <tr
                    v-for="model in models"
                    :key="model.id"
                    :class="{ disabled: !model.enabled }"
                    tabindex="0"
                    @click="openModelEdit(model)"
                    @keydown.enter="openModelEdit(model)"
                  >
                    <td class="col-model">
                      <div class="model-cell">
                        <i class="model-dot" :class="{ off: !model.enabled }" />
                        <div class="model-copy">
                          <strong>{{ model.model_id }}</strong>
                          <small>{{ model.description || model.display_name }}</small>
                        </div>
                        <span v-if="model.is_default" class="tag default">默认</span>
                        <span v-if="model.is_fallback" class="tag fallback">兜底</span>
                      </div>
                    </td>
                    <td class="col-provider">
                      <span class="provider-pill">{{ model.provider_name }}</span>
                    </td>
                    <td class="col-context">{{ formatContext(model.context_window) }}</td>
                    <td class="col-concurrency">{{ model.max_concurrency }}</td>
                    <td class="col-status" @click.stop>
                      <a-switch
                        :checked="model.enabled"
                        size="small"
                        :aria-label="`${model.model_id} 启用开关`"
                        @change="onModelSwitch(model, $event)"
                      />
                    </td>
                  </tr>
                </tbody>
              </table>
              <div v-if="!models.length" class="table-empty">还没有模型，先添加一个吧</div>
            </div>

            <div class="table-foot">
              <span>每 30 秒同步上游模型列表</span>
              <span>共 {{ models.length }} 个模型</span>
            </div>
          </section>
        </div>

        <aside class="right-col">
          <section class="panel">
            <div class="panel-head">
              <div>
                <h2>默认路由</h2>
                <p>失败或超时仅触发一次兜底</p>
              </div>
              <a-spin v-if="routingSaving" size="small" />
            </div>
            <label class="field-label">默认模型</label>
            <a-select
              :value="routing.default_model_id || undefined"
              :options="modelOptions"
              placeholder="选择默认模型"
              class="routing-select"
              :disabled="routingSaving"
              @change="onDefaultModelChange"
            />
            <label class="field-label">兜底模型</label>
            <a-select
              :value="routing.fallback_model_id || undefined"
              :options="fallbackOptions"
              placeholder="不设置兜底"
              allow-clear
              class="routing-select"
              :disabled="routingSaving"
              @change="onFallbackModelChange"
            />
            <p class="routing-hint">
              <SafetyCertificateOutlined />
              停用中的模型不可设为默认或兜底
            </p>
          </section>

          <section class="panel">
            <div class="panel-head">
              <div>
                <h2>接入健康</h2>
                <p>30 秒自动刷新</p>
              </div>
              <span class="live-pill"><i /> 实时</span>
            </div>
            <div class="health-list">
              <div class="health-row">
                <span>上游可用</span>
                <strong class="success">{{ health?.upstreams_available ?? 0 }} / {{ health?.upstreams_total ?? 0 }}</strong>
              </div>
              <div class="health-row">
                <span>近 1 小时错误</span>
                <strong :class="{ danger: (health?.errors_1h ?? 0) > 0 }">{{ health?.errors_1h ?? 0 }} 次</strong>
              </div>
              <div class="health-row">
                <span>首字节 P95</span>
                <strong>{{ health?.first_byte_p95_ms ?? 0 }} ms</strong>
              </div>
            </div>
            <button type="button" class="refresh-link" @click="refreshHealth(false)">
              <ReloadOutlined />
              立即刷新
            </button>
          </section>

          <section class="panel">
            <div class="panel-head">
              <div>
                <h2>最近变更</h2>
                <p>配置操作审计</p>
              </div>
              <button type="button" class="link" @click="openAudits">全部 ›</button>
            </div>
            <div class="audit-list">
              <div v-for="item in audits.slice(0, 3)" :key="item.id" class="audit-item">
                <span class="audit-time">{{ formatChangeTime(item.created_at) }}</span>
                <span class="audit-text">{{ item.detail }}</span>
              </div>
              <p v-if="!audits.length" class="empty-hint">暂无变更记录</p>
            </div>
          </section>

          <section class="panel security-panel">
            <div class="security-head">
              <span class="security-icon"><SafetyCertificateOutlined /></span>
              <div>
                <h2>密钥安全</h2>
                <p>Key 只写不读，前端永不回显；测试连接由服务端发起。</p>
              </div>
            </div>
          </section>
        </aside>
      </div>
    </template>

    <ProviderFormModal
      v-model:open="formOpen"
      :provider="editingProvider"
      @saved="onProviderSaved"
      @deleted="onProviderSaved"
    />
    <ModelFormModal
      v-model:open="modelFormOpen"
      :model="editingModel"
      :providers="providers"
      @saved="onModelSaved"
    />

    <a-modal v-model:open="auditOpen" title="最近变更" :footer="null" :width="520">
      <div class="audit-modal">
        <div v-for="item in allAudits" :key="item.id" class="audit-modal-row">
          <span class="audit-modal-dot" />
          <div>
            <p>{{ item.detail }}</p>
            <small>{{ formatChangeTime(item.created_at) }} · {{ item.actor_name }}</small>
          </div>
        </div>
        <a-empty v-if="!allAudits.length" description="暂无审计记录" />
      </div>
    </a-modal>
  </div>
</template>

<style scoped>
.providers-page {
  max-width: 1440px;
}

.tabs-row {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}

.filter-tabs {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  padding: 3px;
  border-radius: 9px;
  background: #f2f4f7;
}

.filter-tab {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  height: 27px;
  padding: 0 11px;
  border: none;
  border-radius: 7px;
  background: transparent;
  color: var(--dt-text-muted);
  font-size: 10.6px;
  font-weight: 550;
  cursor: pointer;
}

.filter-tab span {
  color: var(--dt-text-faint);
  font-size: 9.4px;
}

.filter-tab:hover {
  color: var(--dt-text);
}

.filter-tab.active {
  background: #fff;
  color: var(--dt-text);
  box-shadow: 0 1px 2px rgba(16, 24, 40, 0.08);
}

.search-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 26px;
  padding: 0 9px;
  border-radius: 999px;
  background: var(--dt-brand-soft);
  color: var(--dt-brand-hover);
  font-size: 10.2px;
}

.search-chip button {
  border: none;
  background: transparent;
  color: var(--dt-brand-hover);
  font-size: 13px;
  line-height: 1;
  cursor: pointer;
}

.card-arrows {
  display: inline-flex;
  gap: 5px;
  margin-left: auto;
}

.card-arrows button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border: 1px solid var(--dt-line);
  border-radius: 7px;
  background: #fff;
  color: var(--dt-text-muted);
  font-size: 9px;
  cursor: pointer;
}

.card-arrows button:hover {
  border-color: #cfd9ff;
  color: var(--dt-brand);
}

.loading-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
  margin-bottom: 16px;
}

.loading-grid :deep(.ant-skeleton) {
  padding: 18px;
  border: 1px solid var(--dt-line);
  border-radius: 14px;
  background: #fff;
}

.provider-row {
  display: grid;
  grid-auto-flow: column;
  grid-auto-columns: 284px;
  gap: 16px;
  overflow-x: auto;
  scroll-snap-type: x mandatory;
  padding-bottom: 2px;
  margin-bottom: 16px;
  max-width: 884px;
}

.provider-row::-webkit-scrollbar {
  height: 0;
}

.empty-panel {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 220px;
  margin-bottom: 16px;
  text-align: center;
}

.empty-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 46px;
  height: 46px;
  margin-bottom: 12px;
  border-radius: 13px;
  background: var(--dt-brand-soft);
  color: var(--dt-brand);
  font-size: 20px;
}

.empty-panel h3 {
  margin: 0 0 6px;
  color: var(--dt-text);
  font-size: 15px;
  font-weight: 650;
}

.empty-panel p {
  margin: 0 0 16px;
  color: var(--dt-text-muted);
  font-size: 11.6px;
}

.empty-actions {
  display: flex;
  gap: 8px;
}

.content-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 312px;
  gap: 16px;
  align-items: start;
}

.left-col,
.right-col {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
}

.right-col {
  gap: 12px;
}

.right-col .panel {
  padding: 12px 14px;
}

.right-col .panel-head {
  margin-bottom: 8px;
}

.right-col .field-label {
  margin: 7px 0 4px;
}

.right-col .routing-select :deep(.ant-select-selector) {
  height: 30px !important;
}

.right-col .routing-hint {
  margin-top: 7px;
}

.right-col .health-row {
  padding: 5px 0;
}

.right-col .refresh-link {
  margin-top: 7px;
}

.right-col .audit-list {
  gap: 6px;
}

.panel {
  padding: 14px 16px;
  border: 1px solid var(--dt-line);
  border-radius: var(--dt-radius-card);
  background: #fff;
  box-shadow: var(--dt-shadow-card);
}

.panel-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
}

.panel-head h2 {
  margin: 0 0 2px;
  color: var(--dt-text);
  font-size: 12.8px;
  font-weight: 650;
}

.panel-head p {
  margin: 0;
  color: var(--dt-text-faint);
  font-size: 10px;
  line-height: 15px;
}

.link {
  padding: 0;
  border: none;
  background: transparent;
  color: var(--dt-brand);
  font-size: 10.2px;
  cursor: pointer;
  white-space: nowrap;
}

.model-panel {
  min-width: 0;
}

.model-table-wrap {
  overflow-x: auto;
}

.model-table {
  width: 100%;
  border-collapse: collapse;
  table-layout: fixed;
}

.model-table th {
  padding: 8px 8px;
  border-bottom: 1px solid var(--dt-line-soft);
  color: var(--dt-text-faint);
  font-size: 9.8px;
  font-weight: 550;
  text-align: left;
  white-space: nowrap;
}

.model-table td {
  padding: 8px 8px;
  border-bottom: 1px solid var(--dt-line-soft);
  color: var(--dt-text-secondary);
  font-size: 10.8px;
  vertical-align: middle;
}

.model-table tbody tr {
  height: 50px;
  cursor: pointer;
  outline: none;
}

.model-table tbody tr:hover {
  background: #fafbff;
}

.model-table tbody tr.disabled {
  opacity: 0.55;
}

.model-table tbody tr:last-child td {
  border-bottom: none;
}

.col-model {
  width: 43%;
}

.col-provider {
  width: 22%;
}

.col-context {
  width: 13%;
}

.col-concurrency {
  width: 10%;
}

.col-status {
  width: 12%;
  text-align: right;
}

.model-cell {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.model-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--dt-brand);
  flex: 0 0 auto;
}

.model-dot.off {
  background: #d0d5dd;
}

.model-copy {
  display: flex;
  flex-direction: column;
  gap: 1px;
  min-width: 0;
  flex: 1;
}

.model-copy strong {
  color: var(--dt-text);
  font-size: 11.4px;
  font-weight: 620;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.model-copy small {
  color: var(--dt-text-faint);
  font-size: 9.4px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tag {
  height: 17px;
  padding: 0 5px;
  border-radius: 4px;
  font-size: 8.2px;
  font-weight: 600;
  display: inline-flex;
  align-items: center;
  flex: 0 0 auto;
}

.tag.default {
  background: var(--dt-brand-soft);
  color: var(--dt-brand-hover);
}

.tag.fallback {
  background: #f2f4f7;
  color: var(--dt-text-muted);
}

.provider-pill {
  display: inline-flex;
  align-items: center;
  height: 20px;
  padding: 0 7px;
  border-radius: 5px;
  background: #f2f4f7;
  color: var(--dt-text-secondary);
  font-size: 9.4px;
  white-space: nowrap;
}

.table-empty {
  padding: 30px 10px;
  color: var(--dt-text-faint);
  font-size: 11px;
  text-align: center;
}

.table-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 8px;
  padding-top: 9px;
  border-top: 1px solid var(--dt-line-soft);
  color: var(--dt-text-faint);
  font-size: 9.4px;
}

.field-label {
  display: block;
  margin: 11px 0 6px;
  color: var(--dt-text-muted);
  font-size: 10.2px;
  font-weight: 550;
}

.routing-select {
  width: 100%;
}

.routing-select :deep(.ant-select-selector) {
  height: 34px !important;
  border-radius: 8px !important;
  font-size: 11.4px;
}

.routing-hint {
  display: flex;
  align-items: center;
  gap: 5px;
  margin: 10px 0 0;
  color: var(--dt-text-faint);
  font-size: 9.2px;
}

.live-pill {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  color: var(--dt-success-text);
  font-size: 9.2px;
  background: var(--dt-success-soft);
  height: 20px;
  padding: 0 7px;
  border-radius: 999px;
}

.live-pill i {
  width: 4px;
  height: 4px;
  border-radius: 50%;
  background: currentColor;
}

.health-list {
  display: flex;
  flex-direction: column;
}

.health-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 7px 0;
  border-bottom: 1px dashed var(--dt-line-soft);
  font-size: 10.2px;
}

.health-row span {
  color: var(--dt-text-muted);
}

.health-row strong {
  color: var(--dt-text);
  font-size: 11.2px;
  font-weight: 650;
}

.health-row strong.success {
  color: var(--dt-success-text);
}

.health-row strong.danger {
  color: var(--dt-danger-text);
}

.upstream-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-top: 9px;
}

.upstream-row {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 9.6px;
}

.upstream-dot {
  width: 4px;
  height: 4px;
  border-radius: 50%;
  background: var(--dt-success);
  flex: 0 0 auto;
}

.upstream-dot.off {
  background: var(--dt-warning);
}

.upstream-name {
  flex: 1;
  min-width: 0;
  color: var(--dt-text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.upstream-latency {
  color: var(--dt-text-faint);
  font-family: var(--dt-mono);
  font-size: 8.8px;
}

.refresh-link {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin-top: 9px;
  padding: 0;
  border: none;
  background: transparent;
  color: var(--dt-text-faint);
  font-size: 9.2px;
  cursor: pointer;
}

.refresh-link:hover {
  color: var(--dt-brand);
}

.audit-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.audit-item {
  display: flex;
  gap: 8px;
  font-size: 9.6px;
  line-height: 15px;
}

.audit-time {
  color: var(--dt-text-faint);
  font-family: var(--dt-mono);
  font-size: 8.8px;
  white-space: nowrap;
  flex: 0 0 auto;
}

.audit-text {
  color: var(--dt-text-secondary);
}

.empty-hint {
  margin: 8px 0;
  color: var(--dt-text-faint);
  font-size: 9.8px;
  text-align: center;
}

.security-panel {
  padding: 12px 14px;
}

.security-head {
  display: flex;
  align-items: flex-start;
  gap: 9px;
}

.security-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border-radius: 8px;
  background: var(--dt-success-soft);
  color: var(--dt-success-text);
  font-size: 12px;
  flex: 0 0 auto;
}

.security-head h2 {
  margin: 0 0 2px;
  color: var(--dt-text-secondary);
  font-size: 11.2px;
  font-weight: 650;
}

.security-head p {
  margin: 0;
  color: var(--dt-text-faint);
  font-size: 9.4px;
  line-height: 14px;
}

.audit-modal {
  display: flex;
  flex-direction: column;
  gap: 10px;
  max-height: 420px;
  overflow-y: auto;
}

.audit-modal-row {
  display: flex;
  gap: 9px;
  padding-bottom: 9px;
  border-bottom: 1px solid var(--dt-line-soft);
}

.audit-modal-dot {
  width: 5px;
  height: 5px;
  margin-top: 6px;
  border-radius: 50%;
  background: var(--dt-brand);
  flex: 0 0 auto;
}

.audit-modal-row p {
  margin: 0 0 2px;
  color: var(--dt-text-secondary);
  font-size: 11.4px;
  line-height: 17px;
}

.audit-modal-row small {
  color: var(--dt-text-faint);
  font-size: 9.6px;
}

@media (max-width: 1200px) {
  .content-grid {
    grid-template-columns: 1fr;
  }

  .right-col {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    align-items: start;
  }

  .security-panel {
    display: none;
  }
}

@media (max-width: 1024px) {
  .provider-row {
    grid-auto-columns: 284px;
  }

  .loading-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 768px) {
  .provider-row {
    grid-auto-columns: 86%;
  }

  .right-col {
    grid-template-columns: 1fr;
  }

  .loading-grid {
    grid-template-columns: 1fr;
  }

  .col-provider,
  .col-context {
    display: none;
  }

  .col-model {
    width: 62%;
  }

  .col-concurrency {
    width: 18%;
  }

  .col-status {
    width: 20%;
  }
}
</style>
