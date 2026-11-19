<script setup lang="ts">
/** 用量统计：按天查看 tokens 与消息量，支持 7 / 14 / 30 天切换。 */
import { computed, onMounted, ref } from 'vue'
import { message } from 'ant-design-vue'
import { BarChartOutlined, ReloadOutlined } from '@ant-design/icons-vue'
import { formatNumber, formatTokens } from '@deeptalk/shared'
import type { DailyUsage } from '@deeptalk/shared'
import { fetchUsageTimeline } from '@/services/admin'
import { useAdminStore } from '@/stores/admin'
import PageHeader from '@/components/base/PageHeader.vue'
import StatCard from '@/components/base/StatCard.vue'

const admin = useAdminStore()

const days = ref(14)
const loading = ref(true)
const items = ref<DailyUsage[]>([])
const totals = ref({ tokens: 0, messages: 0 })

const maxTokens = computed(() => Math.max(1, ...items.value.map((item) => item.tokens)))
const avgDaily = computed(() => (items.value.length ? Math.round(totals.value.tokens / items.value.length) : 0))
const avgPerMessage = computed(() =>
  totals.value.messages ? Math.round(totals.value.tokens / totals.value.messages) : 0,
)

/** 加载用量数据。 */
async function load(showError = true): Promise<void> {
  loading.value = true
  try {
    const data = await fetchUsageTimeline(days.value)
    items.value = data.days
    totals.value = data.totals
  } catch (error) {
    if (showError) message.error(error instanceof Error ? error.message : '用量加载失败')
    items.value = []
  } finally {
    loading.value = false
  }
}

/** 切换时间范围。 */
async function changeDays(value: number): Promise<void> {
  days.value = value
  await load()
}

/** 柱高。 */
function barHeight(tokens: number): string {
  if (!tokens) return '0%'
  return `${Math.max(2, Math.round((tokens / maxTokens.value) * 100))}%`
}

/** X 轴标签抽稀。 */
function showLabel(index: number): boolean {
  const step = days.value <= 7 ? 1 : days.value <= 14 ? 2 : 5
  return index % step === 0 || index === items.value.length - 1
}

onMounted(() => {
  if (!admin.overview) void admin.loadShell()
  void load()
})
</script>

<template>
  <div class="usage-page">
    <PageHeader title="用量统计" subtitle="按天聚合 tokens 与消息量，用于观察用量趋势和额度消耗。">
      <template #actions>
        <a-segmented
          :value="days"
          :options="[
            { label: '7 天', value: 7 },
            { label: '14 天', value: 14 },
            { label: '30 天', value: 30 },
          ]"
          @change="changeDays"
        />
        <a-button :loading="loading" @click="load()">
          <ReloadOutlined />
          刷新
        </a-button>
      </template>
    </PageHeader>

    <div class="stat-grid">
      <StatCard label="tokens 总量" :value="formatTokens(totals.tokens)" :sub="`最近 ${days} 天 · 含深度思考`" />
      <StatCard label="消息量" :value="formatNumber(totals.messages)" sub="助手回答条数" />
      <StatCard label="日均 tokens" :value="formatTokens(avgDaily)" sub="按自然日平均" />
      <StatCard label="单条均值" :value="formatTokens(avgPerMessage)" sub="tokens / 回答" />
    </div>

    <section class="panel chart-panel">
      <div class="panel-head">
        <div>
          <h2>用量趋势</h2>
          <p>最近 {{ days }} 天 · 鼠标悬停查看单日明细</p>
        </div>
        <span class="chart-total">{{ formatTokens(totals.tokens) }} tokens</span>
      </div>

      <div v-if="loading" class="chart-loading">
        <a-skeleton :paragraph="{ rows: 4 }" active />
      </div>
      <div v-else class="chart" role="img" aria-label="每日 tokens 柱状图">
        <div class="grid-lines" aria-hidden="true">
          <span /><span /><span /><span />
        </div>
        <a-tooltip
          v-for="(item, index) in items"
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
    </section>

    <section class="panel">
      <div class="panel-head">
        <div>
          <h2>按天明细</h2>
          <p>与上方柱状图同源，便于核对</p>
        </div>
        <BarChartOutlined class="panel-icon" />
      </div>
      <div class="table-wrap">
        <table class="usage-table">
          <thead>
            <tr>
              <th>日期</th>
              <th>消息数</th>
              <th>tokens</th>
              <th>单条均值</th>
              <th>占总量</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in [...items].reverse()" :key="item.date">
              <td>{{ item.date }}</td>
              <td>{{ formatNumber(item.messages) }}</td>
              <td>{{ formatTokens(item.tokens) }}</td>
              <td>{{ item.messages ? formatTokens(Math.round(item.tokens / item.messages)) : '—' }}</td>
              <td>
                <span class="share">
                  <i :style="{ width: `${totals.tokens ? Math.round((item.tokens / totals.tokens) * 100) : 0}%` }" />
                </span>
                {{ totals.tokens ? Math.round((item.tokens / totals.tokens) * 100) : 0 }}%
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <div v-if="!loading && !items.length" class="empty-state">暂无用量数据</div>
    </section>
  </div>
</template>

<style scoped>
.usage-page {
  max-width: 1400px;
}

.stat-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 16px;
  margin-bottom: 16px;
}

.panel {
  padding: 16px 18px;
  border: 1px solid var(--dt-line);
  border-radius: var(--dt-radius-card);
  background: #fff;
  box-shadow: var(--dt-shadow-card);
  margin-bottom: 16px;
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
}

.chart-total {
  color: var(--dt-brand);
  font-size: 11.4px;
  font-weight: 650;
}

.chart {
  position: relative;
  display: flex;
  align-items: flex-end;
  gap: 6px;
  height: 230px;
  padding-top: 8px;
}

.grid-lines {
  position: absolute;
  inset: 8px 0 22px;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  pointer-events: none;
}

.grid-lines span {
  display: block;
  border-top: 1px dashed #f0f2f6;
}

.bar-slot {
  position: relative;
  z-index: 1;
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
  height: 198px;
}

.bar {
  width: 72%;
  max-width: 28px;
  border-radius: 5px 5px 3px 3px;
  background: linear-gradient(180deg, #6d86ff 0%, #4d6bfe 100%);
  transition: height 0.3s ease;
}

.bar-slot:hover .bar {
  background: linear-gradient(180deg, #8b9dff 0%, #3b5bdb 100%);
}

.bar-label {
  margin-top: 6px;
  color: var(--dt-text-faint);
  font-size: 9px;
  white-space: nowrap;
}

.chart-loading {
  height: 230px;
}

.panel-icon {
  color: var(--dt-text-faint);
  font-size: 16px;
}

.table-wrap {
  overflow-x: auto;
}

.usage-table {
  width: 100%;
  border-collapse: collapse;
}

.usage-table th {
  padding: 8px 10px;
  border-bottom: 1px solid var(--dt-line-soft);
  color: var(--dt-text-faint);
  font-size: 9.8px;
  font-weight: 550;
  text-align: left;
}

.usage-table td {
  padding: 9px 10px;
  border-bottom: 1px solid var(--dt-line-soft);
  color: var(--dt-text-secondary);
  font-size: 10.8px;
}

.usage-table tr:last-child td {
  border-bottom: none;
}

.share {
  display: inline-block;
  width: 46px;
  height: 4px;
  margin-right: 6px;
  border-radius: 999px;
  background: #eef0f4;
  overflow: hidden;
  vertical-align: middle;
}

.share i {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: var(--dt-brand);
}

.empty-state {
  padding: 30px;
  color: var(--dt-text-faint);
  font-size: 11px;
  text-align: center;
}

@media (max-width: 1024px) {
  .stat-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 640px) {
  .stat-grid {
    grid-template-columns: 1fr;
  }

  .chart {
    gap: 2px;
    height: 190px;
  }

  .bar-track {
    height: 158px;
  }
}
</style>
