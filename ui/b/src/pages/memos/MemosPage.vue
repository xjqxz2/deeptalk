<script setup lang="ts">
/**
 * 小橘灵感录：后台查看灵感总量、增长趋势，并可删除任意一条。
 * 数据来自 /b/memos/stats 与 /b/memos。
 */
import { computed, onMounted, ref } from 'vue'
import { Modal, message } from 'ant-design-vue'
import { BulbOutlined, DeleteOutlined, ReloadOutlined } from '@ant-design/icons-vue'
import { formatNumber } from '@deeptalk/shared'
import type { MemoItem, MemoStats } from '@deeptalk/shared'
import { deleteMemo, fetchMemoStats, fetchMemos } from '@/services/admin'
import PageHeader from '@/components/base/PageHeader.vue'
import StatCard from '@/components/base/StatCard.vue'

const loading = ref(true)
const deleting = ref(0)
const stats = ref<MemoStats>({ total: 0, today: 0, last7: 0, users: 0, daily: [] })
const items = ref<MemoItem[]>([])
const total = ref(0)
const page = ref(1)
const size = 20
const keyword = ref('')

/** 趋势条的最大值，用来算每根柱子的高度。 */
const maxDaily = computed(() => Math.max(1, ...stats.value.daily.map((day) => day.count)))

/** 近 7 天增长（今天 vs 前一天），用于展示上升/下降。 */
const deltaToday = computed(() => {
  const daily = stats.value.daily
  if (daily.length < 2) return 0
  return daily[daily.length - 1].count - daily[daily.length - 2].count
})

/** 加载统计与列表。 */
async function load(showError = true): Promise<void> {
  loading.value = true
  try {
    const [stat, list] = await Promise.all([
      fetchMemoStats(),
      fetchMemos({ q: keyword.value || undefined, page: page.value, size }),
    ])
    stats.value = stat
    items.value = list.items
    total.value = list.total
  } catch (error) {
    if (showError) message.error(error instanceof Error ? error.message : '灵感录加载失败')
    items.value = []
  } finally {
    loading.value = false
  }
}

/** 搜索：回到第一页。 */
function onSearch(): void {
  page.value = 1
  void load()
}

/** 翻页。 */
function onPage(next: number): void {
  page.value = next
  void load()
}

/** 删除一条灵感：先确认，再真正删除。 */
async function onDelete(item: MemoItem): Promise<void> {
  const ok = await new Promise<boolean>((resolve) => {
    Modal.confirm({
      title: '删除这条灵感？',
      content: `「${item.title || '无标题'}」删除后无法恢复。`,
      okText: '删除',
      cancelText: '取消',
      okButtonProps: { danger: true },
      onOk: () => resolve(true),
      onCancel: () => resolve(false),
    })
  })
  if (!ok) return
  deleting.value = item.id
  try {
    await deleteMemo(item.id)
    message.success('已删除')
    await load(false)
  } catch (error) {
    message.error(error instanceof Error ? error.message : '删除失败')
  } finally {
    deleting.value = 0
  }
}

/** 格式化时间。 */
function when(value: string): string {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '—'
  return `${date.getMonth() + 1}/${date.getDate()} ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`
}

onMounted(() => void load())
</script>

<template>
  <div class="memo-page">
    <PageHeader title="小橘灵感录" description="用户在对话里记下的灵感，可查看总量、增长趋势，并清理不合适的内容。">
      <template #actions>
        <a-button :loading="loading" @click="load()">
          <template #icon><ReloadOutlined /></template>
          刷新
        </a-button>
      </template>
    </PageHeader>

    <div class="stat-row">
      <StatCard label="灵感总数" :value="formatNumber(stats.total)" sub="全部用户累计" />
      <StatCard
        label="今日新增"
        :value="formatNumber(stats.today)"
        :sub="deltaToday >= 0 ? `较昨天 +${deltaToday}` : `较昨天 ${deltaToday}`"
      />
      <StatCard label="近 7 天" :value="formatNumber(stats.last7)" sub="滚动一周新增" />
      <StatCard label="记录人数" :value="formatNumber(stats.users)" sub="至少记过一条" />
    </div>

    <section class="card trend-card">
      <header class="card-head">
        <h3>近 7 天趋势</h3>
        <span class="muted">每天新增的灵感条数</span>
      </header>
      <div class="trend">
        <div v-for="day in stats.daily" :key="day.date" class="trend-col">
          <span class="trend-value">{{ day.count }}</span>
          <div class="trend-bar" :style="{ height: `${Math.max(4, (day.count / maxDaily) * 100)}%` }" />
          <span class="trend-date">{{ day.date.slice(5) }}</span>
        </div>
      </div>
      <p v-if="!stats.daily.length" class="empty-hint">还没有数据，等用户记下第一条灵感。</p>
    </section>

    <section class="card list-card">
      <header class="card-head">
        <h3>全部灵感</h3>
        <a-input-search
          v-model:value="keyword"
          placeholder="搜索标题、正文、标签或用户邮箱"
          allow-clear
          style="max-width: 320px"
          @search="onSearch"
        />
      </header>

      <table class="memo-table">
        <thead>
          <tr>
            <th class="col-title">标题</th>
            <th class="col-content">内容</th>
            <th class="col-tags">标签</th>
            <th class="col-user">记录人</th>
            <th class="col-time">时间</th>
            <th class="col-actions">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in items" :key="item.id">
            <td class="col-title"><span class="title-text">{{ item.title || '无标题' }}</span></td>
            <td class="col-content"><span class="content-text">{{ item.content }}</span></td>
            <td class="col-tags">
              <span v-for="tag in item.tags.split(',').filter(Boolean)" :key="tag" class="tag">{{ tag }}</span>
              <span v-if="!item.tags" class="muted">—</span>
            </td>
            <td class="col-user">{{ item.user_email || '—' }}</td>
            <td class="col-time">{{ when(item.created_at) }}</td>
            <td class="col-actions">
              <a-button type="text" danger size="small" :loading="deleting === item.id" @click="onDelete(item)">
                <template #icon><DeleteOutlined /></template>
                删除
              </a-button>
            </td>
          </tr>
        </tbody>
      </table>

      <div v-if="!loading && !items.length" class="table-empty">
        <span class="empty-icon"><BulbOutlined /></span>
        <p>还没有灵感</p>
        <small>用户在对话里说「记一下…」就会出现在这里。</small>
      </div>

      <footer v-if="total > size" class="pager">
        <a-pagination
          :current="page"
          :total="total"
          :page-size="size"
          :show-size-changer="false"
          @change="onPage"
        />
      </footer>
    </section>
  </div>
</template>

<style scoped>
.memo-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.stat-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 12px;
}

.card {
  background: var(--dt-surface, #fff);
  border: 1px solid var(--dt-line, #eaecf0);
  border-radius: 12px;
  padding: 16px 18px;
}

.card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 14px;
}

.card-head h3 {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
  color: var(--dt-text, #101828);
}

.muted {
  color: var(--dt-text-tertiary, #98a2b3);
  font-size: 12px;
}

.trend {
  display: flex;
  align-items: flex-end;
  gap: 10px;
  height: 132px;
}

.trend-col {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: flex-end;
  gap: 6px;
  height: 100%;
}

.trend-bar {
  width: 100%;
  max-width: 38px;
  border-radius: 6px 6px 2px 2px;
  background: linear-gradient(180deg, #ffb066, #ff8a3d);
  transition: height 0.3s ease;
}

.trend-value {
  font-size: 12px;
  color: var(--dt-text-secondary, #475467);
}

.trend-date {
  font-size: 11px;
  color: var(--dt-text-tertiary, #98a2b3);
}

.memo-table {
  width: 100%;
  table-layout: fixed;
  border-collapse: collapse;
  font-size: 13px;
}

.memo-table th {
  text-align: left;
  font-weight: 500;
  color: var(--dt-text-tertiary, #98a2b3);
  padding: 8px 10px;
  border-bottom: 1px solid var(--dt-line, #eaecf0);
}

.memo-table td {
  padding: 10px;
  border-bottom: 1px solid var(--dt-line, #eaecf0);
  color: var(--dt-text, #101828);
  vertical-align: top;
}

.col-title { width: 26%; }
.col-content { width: 34%; }
.col-tags { width: 14%; }
.col-user { width: 14%; }
.col-time { width: 9%; }
.col-actions { width: 80px; text-align: right; }

.title-text {
  font-weight: 550;
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.content-text {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  color: var(--dt-text-secondary, #475467);
}

.tag {
  display: inline-block;
  margin: 0 4px 4px 0;
  padding: 1px 7px;
  border-radius: 999px;
  background: var(--dt-hover, #f2f4f7);
  color: var(--dt-text-secondary, #475467);
  font-size: 11px;
}

.table-empty {
  padding: 36px 0;
  text-align: center;
  color: var(--dt-text-tertiary, #98a2b3);
}

.empty-icon {
  font-size: 22px;
  color: #ff8a3d;
}

.table-empty p {
  margin: 8px 0 2px;
  color: var(--dt-text-secondary, #475467);
}

.pager {
  display: flex;
  justify-content: flex-end;
  padding-top: 12px;
}

.empty-hint {
  color: var(--dt-text-tertiary, #98a2b3);
  font-size: 12px;
  margin: 10px 0 0;
}
</style>
