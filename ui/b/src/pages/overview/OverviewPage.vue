<script setup lang="ts">
/** 后台概览：关键指标、用量趋势、接入健康与最近变更。 */
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import {
  ApiOutlined,
  ArrowRightOutlined,
  CheckCircleFilled,
  CloudServerOutlined,
  ReloadOutlined,
  SettingOutlined,
  TeamOutlined,
  ThunderboltOutlined,
  WarningOutlined,
} from '@ant-design/icons-vue'
import { formatNumber, formatChangeTime, formatTokens } from '@deeptalk/shared'
import type { DailyUsage } from '@deeptalk/shared'
import { fetchUsageTimeline } from '@/services/admin'
import { useAdminStore } from '@/stores/admin'
import PageHeader from '@/components/base/PageHeader.vue'
import StatCard from '@/components/base/StatCard.vue'

const router = useRouter()
const admin = useAdminStore()

const timeline = ref<DailyUsage[]>([])
const timelineLoading = ref(false)
const refreshing = ref(false)
const loadFailed = ref(false)

const overview = computed(() => admin.overview)
const maxTokens = computed(() => Math.max(1, ...timeline.value.map((item) => item.tokens)))
const chartTotal = computed(() => timeline.value.reduce((sum, item) => sum + item.tokens, 0))
const chartMessages = computed(() => timeline.value.reduce((sum, item) => sum + item.messages, 0))

/** 加载趋势数据。 */
async function loadTimeline(): Promise<void> {
  timelineLoading.value = true
  try {
    const data = await fetchUsageTimeline(14)
    timeline.value = data.days
  } catch {
    timeline.value = []
  } finally {
    timelineLoading.value = false
  }
}

/** 刷新整页。 */
async function refresh(): Promise<void> {
  refreshing.value = true
  try {
    await Promise.all([admin.loadShell(), loadTimeline()])
    loadFailed.value = !admin.overview
    if (admin.overview) message.success('已刷新')
  } finally {
    refreshing.value = false
  }
}

/** 柱状图高度百分比。 */
function barHeight(tokens: number): string {
  if (!tokens) return '0%'
  return `${Math.max(3, Math.round((tokens / maxTokens.value) * 100))}%`
}

/** 只显示部分日期标签，避免拥挤。 */
function showLabel(index: number): boolean {
  return index % 3 === 0 || index === timeline.value.length - 1
}

onMounted(async () => {
  if (!admin.overview) await admin.loadShell()
  loadFailed.value = !admin.overview
  void loadTimeline()
})
</script>

<template>
  <div class="overview-page">
    <PageHeader title="概览" subtitle="平台运行状态、模型接入与用户增长一屏掌握。">
      <template #actions>
        <a-button :loading="refreshing" @click="refresh">
          <ReloadOutlined />
          刷新
        </a-button>
        <a-button type="primary" @click="router.push('/providers')">
          <ApiOutlined />
          厂商与模型
        </a-button>
      </template>
    </PageHeader>

    <div v-if="!overview && !loadFailed" class="loading-state">
      <a-skeleton :paragraph="{ rows: 6 }" active />
    </div>

    <template v-if="overview">
      <div class="stat-grid">
        <StatCard
          label="总用户"
          :value="formatNumber(overview.total_users)"
          :sub="`本月新增 ${overview.new_this_month} · 活跃率 ${overview.active_rate}%`"
        />
        <StatCard
          label="本月用量"
          :value="`${formatTokens(overview.monthly_tokens)}`"
          :sub="`${formatNumber(overview.monthly_messages)} 条消息 · 环比 +${overview.monthly_growth}%`"
        />
        <StatCard
          label="已限制"
          :value="formatNumber(overview.restricted)"
          sub="需关注 · 可随时解除"
          :tone="overview.restricted > 0 ? 'danger' : 'default'"
        />
        <StatCard
          label="已启用模型"
          :value="formatNumber(overview.models_enabled)"
          :sub="`${overview.providers_enabled} 个上游厂商 · 共 ${formatNumber(overview.conversation_count)} 段会话`"
        />
      </div>

      <div class="overview-grid">
        <div class="grid-left">
          <section class="panel usage-panel">
            <div class="panel-head">
              <div>
                <h2>用量趋势</h2>
                <p>最近 14 天 tokens 与消息量</p>
              </div>
              <button type="button" class="link" @click="router.push('/usage')">
                查看详情
                <ArrowRightOutlined />
              </button>
            </div>

            <div v-if="timelineLoading" class="chart-skeleton">
              <a-skeleton :paragraph="{ rows: 3 }" active />
            </div>
            <div v-else class="chart">
              <a-tooltip
                v-for="(item, index) in timeline"
                :key="item.date"
                :title="`${item.date} · ${formatTokens(item.tokens)} tokens · ${item.messages} 条`"
                placement="top"
              >
                <div class="bar-slot">
                  <div class="bar-track">
                    <span class="bar" :style="{ height: barHeight(item.tokens) }" />
                  </div>
                  <small v-if="showLabel(index)" class="bar-label">{{ item.date.slice(5) }}</small>
                  <small v-else class="bar-label">&nbsp;</small>
                </div>
              </a-tooltip>
            </div>
            <div class="chart-foot">
              <span>合计 {{ formatTokens(chartTotal) }} tokens</span>
              <span>{{ formatNumber(chartMessages) }} 条消息</span>
            </div>
          </section>

          <section class="panel">
            <div class="panel-head">
              <div>
                <h2>快捷入口</h2>
                <p>常用后台操作</p>
              </div>
            </div>
            <div class="shortcuts">
              <button type="button" class="shortcut" @click="router.push('/providers')">
                <span class="shortcut-icon"><ThunderboltOutlined /></span>
                <span class="shortcut-copy">
                  <strong>配置模型路由</strong>
                  <small>厂商密钥、模型启用与默认兜底</small>
                </span>
                <ArrowRightOutlined class="shortcut-arrow" />
              </button>
              <button type="button" class="shortcut" @click="router.push('/users')">
                <span class="shortcut-icon"><TeamOutlined /></span>
                <span class="shortcut-copy">
                  <strong>管理用户额度</strong>
                  <small>限制登录、重置密码与额度调整</small>
                </span>
                <ArrowRightOutlined class="shortcut-arrow" />
              </button>
              <button type="button" class="shortcut" @click="router.push('/settings')">
                <span class="shortcut-icon"><SettingOutlined /></span>
                <span class="shortcut-copy">
                  <strong>系统设置</strong>
                  <small>注册开关、默认路由与存储信息</small>
                </span>
                <ArrowRightOutlined class="shortcut-arrow" />
              </button>
            </div>
          </section>
        </div>

        <div class="grid-right">
          <section class="panel">
            <div class="panel-head">
              <div>
                <h2>接入健康</h2>
                <p>实时 · 30 秒自动刷新</p>
              </div>
              <span class="live-dot"><i /> 实时</span>
            </div>
            <div class="health-metrics">
              <div class="metric">
                <span>上游可用</span>
                <strong class="success">{{ overview.health.upstreams_available }} / {{ overview.health.upstreams_total }}</strong>
              </div>
              <div class="metric">
                <span>近 1 小时错误</span>
                <strong :class="{ danger: overview.health.errors_1h > 0 }">{{ overview.health.errors_1h }} 次</strong>
              </div>
              <div class="metric">
                <span>首字节 P95</span>
                <strong>{{ overview.health.first_byte_p95_ms }} ms</strong>
              </div>
            </div>
            <div class="provider-health">
              <div v-for="item in overview.health.items" :key="item.provider_id" class="health-row">
                <span class="health-dot" :class="{ off: !item.ok }" />
                <span class="health-name">{{ item.name }}</span>
                <span class="health-latency">{{ item.latency_ms ? `${item.latency_ms} ms` : '—' }}</span>
              </div>
              <p v-if="!overview.health.items.length" class="empty-hint">暂无启用的上游厂商</p>
            </div>
          </section>

          <section class="panel">
            <div class="panel-head">
              <div>
                <h2>最近变更</h2>
                <p>管理员操作审计</p>
              </div>
              <button type="button" class="link" @click="router.push('/providers')">全部</button>
            </div>
            <div class="audit-list">
              <div v-for="item in overview.recent_audits" :key="item.id" class="audit-item">
                <span class="audit-dot" />
                <div class="audit-copy">
                  <p>{{ item.detail }}</p>
                  <small>{{ formatChangeTime(item.created_at) }} · {{ item.actor_name }}</small>
                </div>
              </div>
              <p v-if="!overview.recent_audits.length" class="empty-hint">暂无审计记录</p>
            </div>
          </section>

          <section class="panel storage-panel">
            <div class="panel-head">
              <div>
                <h2>运行状态</h2>
                <p>单实例 SQLite 部署</p>
              </div>
              <CloudServerOutlined class="panel-icon" />
            </div>
            <div class="status-rows">
              <div><span>数据存储</span><strong>SQLite · WAL</strong></div>
              <div><span>服务监听</span><strong>0.0.0.0:9821</strong></div>
              <div><span>会话策略</span><strong>HttpOnly · 7 天</strong></div>
              <div><span>版本</span><strong>v1.0.0</strong></div>
            </div>
            <p class="status-note">
              <CheckCircleFilled />
              服务运行正常
            </p>
          </section>
        </div>
      </div>
    </template>

    <div v-if="!overview && loadFailed" class="error-state">
      <WarningOutlined />
      <p>概览数据加载失败</p>
      <a-button @click="refresh">重新加载</a-button>
    </div>
  </div>
</template>

<style scoped>
.overview-page {
  max-width: 1400px;
}

.stat-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 16px;
  margin-bottom: 16px;
}

.overview-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.62fr) minmax(300px, 1fr);
  gap: 16px;
  align-items: start;
}

.grid-left,
.grid-right {
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

.panel-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 14px;
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
  line-height: 15px;
}

.link {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 0;
  border: none;
  background: transparent;
  color: var(--dt-brand);
  font-size: 10.4px;
  cursor: pointer;
  white-space: nowrap;
}

.link:hover {
  color: var(--dt-brand-hover);
}

.chart {
  display: flex;
  align-items: flex-end;
  gap: 6px;
  height: 148px;
  padding: 6px 0 0;
}

.bar-slot {
  display: flex;
  flex: 1;
  flex-direction: column;
  align-items: center;
  height: 100%;
  min-width: 0;
}

.bar-track {
  display: flex;
  align-items: flex-end;
  justify-content: center;
  width: 100%;
  height: 118px;
}

.bar {
  width: 78%;
  max-width: 22px;
  border-radius: 5px 5px 3px 3px;
  background: linear-gradient(180deg, #6d86ff 0%, #4d6bfe 100%);
  transition: height 0.3s ease;
}

.bar-slot:hover .bar {
  background: linear-gradient(180deg, var(--dt-brand) 0%, var(--dt-brand-hover) 100%);
}

.bar-label {
  margin-top: 6px;
  color: var(--dt-text-faint);
  font-size: 9px;
  white-space: nowrap;
}

.chart-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px solid var(--dt-line-soft);
  color: var(--dt-text-muted);
  font-size: 10px;
}

.chart-skeleton {
  height: 148px;
}

.shortcuts {
  display: flex;
  flex-direction: column;
}

.shortcut {
  display: flex;
  align-items: center;
  gap: 11px;
  width: 100%;
  padding: 10px 6px;
  border: none;
  border-bottom: 1px solid var(--dt-line-soft);
  background: transparent;
  text-align: left;
  cursor: pointer;
}

.shortcut:last-child {
  border-bottom: none;
}

.shortcut:hover {
  background: var(--dt-brand-softer);
}

.shortcut-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border-radius: 8px;
  background: var(--dt-brand-soft);
  color: var(--dt-brand);
  font-size: 13px;
  flex: 0 0 auto;
}

.shortcut-copy {
  display: flex;
  flex-direction: column;
  min-width: 0;
  flex: 1;
}

.shortcut-copy strong {
  color: var(--dt-text-secondary);
  font-size: 11.8px;
  font-weight: 600;
}

.shortcut-copy small {
  color: var(--dt-text-faint);
  font-size: 9.8px;
}

.shortcut-arrow {
  color: var(--dt-text-faint);
  font-size: 10px;
}

.live-dot {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  color: var(--dt-text-faint);
  font-size: 9.6px;
}

.live-dot i {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: var(--dt-success);
  box-shadow: 0 0 0 2px var(--dt-success-soft);
}

.health-metrics {
  display: flex;
  flex-direction: column;
  gap: 0;
}

.metric {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 0;
  border-bottom: 1px dashed var(--dt-line-soft);
}

.metric span {
  color: var(--dt-text-muted);
  font-size: 10.8px;
}

.metric strong {
  color: var(--dt-text);
  font-size: 11.8px;
  font-weight: 650;
}

.metric strong.success {
  color: var(--dt-success-text);
}

.metric strong.danger {
  color: var(--dt-danger-text);
}

.provider-health {
  display: flex;
  flex-direction: column;
  gap: 7px;
  margin-top: 10px;
}

.health-row {
  display: flex;
  align-items: center;
  gap: 7px;
  font-size: 10.4px;
}

.health-dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: var(--dt-success);
  flex: 0 0 auto;
}

.health-dot.off {
  background: var(--dt-warning);
}

.health-name {
  flex: 1;
  min-width: 0;
  color: var(--dt-text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.health-latency {
  color: var(--dt-text-faint);
  font-family: var(--dt-mono);
  font-size: 9.4px;
}

.audit-list {
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.audit-item {
  display: flex;
  gap: 8px;
  padding: 8px 0;
  border-bottom: 1px solid var(--dt-line-soft);
}

.audit-item:last-child {
  border-bottom: none;
}

.audit-dot {
  width: 5px;
  height: 5px;
  margin-top: 6px;
  border-radius: 50%;
  background: var(--dt-brand);
  flex: 0 0 auto;
}

.audit-copy p {
  margin: 0 0 2px;
  color: var(--dt-text-secondary);
  font-size: 10.8px;
  line-height: 16px;
}

.audit-copy small {
  color: var(--dt-text-faint);
  font-size: 9.2px;
}

.empty-hint {
  margin: 10px 0 2px;
  color: var(--dt-text-faint);
  font-size: 10.4px;
  text-align: center;
}

.panel-icon {
  color: var(--dt-text-faint);
  font-size: 16px;
}

.status-rows {
  display: flex;
  flex-direction: column;
  gap: 7px;
}

.status-rows div {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 10.4px;
}

.status-rows span {
  color: var(--dt-text-muted);
}

.status-rows strong {
  color: var(--dt-text-secondary);
  font-weight: 550;
  font-family: var(--dt-mono);
  font-size: 9.6px;
}

.status-note {
  display: flex;
  align-items: center;
  gap: 5px;
  margin: 12px 0 0;
  padding-top: 10px;
  border-top: 1px solid var(--dt-line-soft);
  color: var(--dt-success-text);
  font-size: 9.8px;
}

.loading-state {
  padding: 40px;
  border: 1px solid var(--dt-line);
  border-radius: 14px;
  background: var(--dt-surface);
}

.error-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 40px;
  color: var(--dt-text-muted);
}

@media (max-width: 1200px) {
  .stat-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .overview-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 640px) {
  .stat-grid {
    grid-template-columns: 1fr;
  }

  .chart {
    gap: 3px;
  }

  .bar-label {
    font-size: 8px;
  }
}
</style>
