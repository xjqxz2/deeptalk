<script setup lang="ts">
/**
 * 用户管理：列表做运营、右栏做处置。
 * 支持搜索、状态筛选、套餐/额度、限制登录、重置密码与 CSV 导出；
 * 后台不读取任何对话正文。
 */
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { message, Modal } from 'ant-design-vue'
import {
  DownloadOutlined,
  FilterOutlined,
  MoreOutlined,
  PlusOutlined,
  ReloadOutlined,
  SearchOutlined,
  WarningOutlined,
} from '@ant-design/icons-vue'
import { formatNumber, formatPercent, formatTokens } from '@deeptalk/shared'
import type { User, UserStats } from '@deeptalk/shared'
import {
  exportUsers,
  fetchUserDetail,
  fetchUsers,
  logoutAllSessions,
  setUserDisabled,
  updatePlan,
} from '@/services/admin'
import { useAdminStore } from '@/stores/admin'
import { useUserStore } from '@/stores/user'
import PageHeader from '@/components/base/PageHeader.vue'
import StatCard from '@/components/base/StatCard.vue'
import PlanTag from '@/components/base/PlanTag.vue'
import UserDetailPanel from '@/components/business/UserDetailPanel.vue'
import QuotaModal from '@/components/business/QuotaModal.vue'
import ResetPasswordModal from '@/components/business/ResetPasswordModal.vue'
import NewUserModal from '@/components/business/NewUserModal.vue'

type StatusFilter = 'all' | 'normal' | 'restricted' | 'quota'
type SortKey = 'recent' | 'created' | 'usage'

const route = useRoute()
const admin = useAdminStore()
const currentUser = useUserStore()

const loading = ref(true)
const exporting = ref(false)
const items = ref<User[]>([])
const total = ref(0)
const stats = ref<UserStats | null>(null)
const page = ref(1)
const pageSize = ref(8)
const status = ref<StatusFilter>('all')
const sort = ref<SortKey>('recent')
const searchInput = ref('')
const query = ref('')

const selectedId = ref(0)
const selectedUser = ref<User | null>(null)
const detailLoading = ref(false)
const detailBusy = ref(false)
const wide = ref(true)
const drawerOpen = ref(false)

const quotaOpen = ref(false)
const resetOpen = ref(false)
const newUserOpen = ref(false)

let searchTimer: ReturnType<typeof setTimeout> | null = null

const segments = computed(() => [
  { key: 'all' as const, label: '全部', count: stats.value?.total ?? 0 },
  {
    key: 'normal' as const,
    label: '正常',
    count: (stats.value?.total ?? 0) - (stats.value?.restricted ?? 0),
  },
  { key: 'restricted' as const, label: '已限制', count: stats.value?.restricted ?? 0 },
  { key: 'quota' as const, label: '额度将尽', count: stats.value?.quota_alert ?? 0 },
])

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize.value)))
const hasFilters = computed(() => Boolean(query.value) || status.value !== 'all')

/** 状态标签：限制优先，其次额度将尽。 */
function userStatus(user: User): { label: string; tone: 'normal' | 'warning' | 'danger' } {
  if (user.disabled) return { label: '已限制', tone: 'danger' }
  if (!isUnlimited(user) && user.used_percent >= 90) return { label: '额度将尽', tone: 'warning' }
  return { label: '正常', tone: 'normal' }
}

/** 无限版：不显示额度进度，避免出现 0/0 与 0%。 */
function isUnlimited(user: User): boolean {
  return user.plan === 'unlimited' || user.quota_tokens <= 0
}

/** 加载列表并保持选中项。 */
async function loadList(preserve = true): Promise<void> {
  loading.value = true
  try {
    const data = await fetchUsers({
      q: query.value,
      status: status.value,
      sort: sort.value,
      page: page.value,
      page_size: pageSize.value,
    })
    items.value = data.items
    total.value = data.total
    stats.value = data.stats
    if (!data.items.length) {
      selectedId.value = 0
      selectedUser.value = null
      return
    }
    const current = preserve ? data.items.find((item) => item.id === selectedId.value) : undefined
    const target = current ?? data.items[0]
    selectedId.value = target.id
    await loadDetail(target.id)
  } catch (error) {
    message.error(error instanceof Error ? error.message : '用户列表加载失败')
  } finally {
    loading.value = false
  }
}

/** 加载单个用户详情。 */
async function loadDetail(id: number): Promise<void> {
  detailLoading.value = true
  try {
    const data = await fetchUserDetail(id)
    selectedUser.value = data.user
  } catch (error) {
    message.error(error instanceof Error ? error.message : '用户详情加载失败')
  } finally {
    detailLoading.value = false
  }
}

/** 选择用户。 */
async function selectUser(user: User): Promise<void> {
  selectedId.value = user.id
  selectedUser.value = user
  if (!wide.value) drawerOpen.value = true
  await loadDetail(user.id)
}

/** 搜索防抖 300ms。 */
watch(searchInput, (value) => {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    query.value = value.trim()
    page.value = 1
    void loadList(false)
  }, 300)
})

/** 切换状态筛选。 */
function setStatus(key: StatusFilter): void {
  status.value = key
  page.value = 1
  void loadList(false)
}

/** 切换排序。 */
function onSortChange(value: SortKey): void {
  sort.value = value
  page.value = 1
  void loadList(false)
}

/** 每页条数变化。 */
function onPageSizeChanged(): void {
  page.value = 1
  void loadList(false)
}

/** 翻页。 */
function onPageChange(next: number): void {
  page.value = next
  void loadList(false)
}

/** 清空筛选。 */
function resetFilters(): void {
  searchInput.value = ''
  query.value = ''
  status.value = 'all'
  page.value = 1
  void loadList(false)
}

/** 快捷筛选。 */
function quickFilter(key: StatusFilter): void {
  setStatus(key)
}

/** 导出 CSV。 */
async function onExport(): Promise<void> {
  exporting.value = true
  try {
    await exportUsers()
    message.success('CSV 已开始下载（仅包含账号字段）')
  } catch (error) {
    message.error(error instanceof Error ? error.message : '导出失败')
  } finally {
    exporting.value = false
  }
}

/** 额度保存回调。 */
async function onQuotaSaved(user: User): Promise<void> {
  selectedUser.value = { ...(selectedUser.value ?? user), ...user }
  await loadList()
  await admin.refreshOverview()
}

/** 套餐调整（二次确认）。 */
function onPlanChange(plan: 'free' | 'pro' | 'team' | 'unlimited'): void {
  const user = selectedUser.value
  if (!user || user.plan === plan) return
  const label = plan === 'free' ? '免费版' : plan === 'pro' ? '专业版' : plan === 'team' ? '团队版' : '无限版'
  Modal.confirm({
    title: `将套餐调整为「${label}」？`,
    content: '套餐会同时重置该账号的额度；已用量不会被清空。',
    okText: '确认调整',
    cancelText: '取消',
    async onOk() {
      detailBusy.value = true
      try {
        const data = await updatePlan(user.id, plan)
        selectedUser.value = data.user
        message.success('套餐已调整')
        await loadList()
        await admin.refreshOverview()
      } finally {
        detailBusy.value = false
      }
    },
  })
}

/** 限制 / 恢复登录（二次确认）。 */
function onToggleDisabled(disabled: boolean): void {
  const user = selectedUser.value
  if (!user) return
  if (user.role === 'admin') {
    message.warning('管理员账号不可被限制')
    return
  }
  Modal.confirm({
    title: disabled ? `限制「${user.display_name || user.nickname || user.email}」登录？` : `恢复「${user.display_name || user.nickname || user.email}」登录？`,
    content: disabled
      ? '开启后该账号已签发的会话立即失效，用户将无法登录，可随时恢复。'
      : '恢复后用户需要重新登录，已失效的会话不会恢复。',
    okText: disabled ? '限制登录' : '恢复登录',
    okType: disabled ? 'danger' : 'primary',
    cancelText: '取消',
    async onOk() {
      detailBusy.value = true
      try {
        const data = await setUserDisabled(user.id, disabled)
        selectedUser.value = data.user
        message.success(disabled ? '已限制登录' : '已恢复登录')
        await loadList()
        await admin.refreshOverview()
      } finally {
        detailBusy.value = false
      }
    },
  })
}

/** 打开重置密码；禁止在列表里重置当前登录管理员。 */
function openReset(): void {
  const user = selectedUser.value
  if (!user) return
  if (user.id === currentUser.user?.id) {
    message.warning('当前登录管理员请使用右上角「账号与密码」修改密码')
    return
  }
  resetOpen.value = true
}

/** 强制退出全部设备。 */
function onLogoutAll(): void {
  const user = selectedUser.value
  if (!user) return
  if (user.id === currentUser.user?.id) {
    message.warning('当前登录管理员请使用右上角账号菜单退出')
    return
  }
  Modal.confirm({
    title: '强制退出全部设备？',
    content: `「${user.display_name || user.nickname || user.email}」的所有已登录会话将立即失效，需要重新登录。`,
    okText: '全部退出',
    okType: 'danger',
    cancelText: '取消',
    async onOk() {
      detailBusy.value = true
      try {
        await logoutAllSessions(user.id)
        message.success('已强制退出全部设备')
      } finally {
        detailBusy.value = false
      }
    },
  })
}

/** 打开额度弹窗；无限版不设月度上限，直接提示。 */
function openQuota(): void {
  const user = selectedUser.value
  if (!user) return
  if (user.plan === 'unlimited' || user.quota_tokens <= 0) {
    message.warning('无限版不设月度额度上限，无需调整')
    return
  }
  quotaOpen.value = true
}

/** 行内操作。 */
function onRowMenu(info: { key: string }, user: User): void {
  onRowAction(info, user)
}

/** 行内操作。 */
function onRowAction(info: { key: string }, user: User): void {
  selectedId.value = user.id
  selectedUser.value = user
  if (info.key === 'detail') {
    void selectUser(user)
  }
  if (info.key === 'quota') openQuota()
  if (info.key === 'reset') openReset()
  if (info.key === 'toggle') onToggleDisabled(!user.disabled)
  if (info.key === 'logout') onLogoutAll()
}

/** 同步窗口断点，决定详情是内联还是抽屉。 */
function onResize(): void {
  const next = window.innerWidth > 1280
  wide.value = next
  if (next) drawerOpen.value = false
}

/** 新建账号回调。 */
async function onUserCreated(): Promise<void> {
  page.value = 1
  await loadList(false)
  await admin.refreshOverview()
}

onMounted(async () => {
  wide.value = window.innerWidth > 1280
  window.addEventListener('resize', onResize)
  const initialQuery = typeof route.query.q === 'string' ? route.query.q : ''
  if (initialQuery) {
    searchInput.value = initialQuery
    query.value = initialQuery
  }
  await loadList(false)
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', onResize)
  if (searchTimer) clearTimeout(searchTimer)
})
</script>

<template>
  <div class="users-page">
    <PageHeader
      title="用户管理"
      subtitle="账号、套餐与额度；后台不读取对话正文，导出的 CSV 仅含账号字段。"
    >
      <template #actions>
        <a-button :loading="exporting" @click="onExport">
          <DownloadOutlined />
          导出 CSV
        </a-button>
        <a-button type="primary" @click="newUserOpen = true">
          <PlusOutlined />
          新建账号
        </a-button>
      </template>
    </PageHeader>

    <div class="stat-grid">
      <StatCard
        label="总用户"
        :value="formatNumber(stats?.total ?? 0)"
        :sub="`本月新增 ${stats?.new_this_month ?? 0} · 全部邮箱注册`"
      />
      <StatCard
        label="7 日活跃"
        :value="formatNumber(stats?.active_7d ?? 0)"
        :sub="`${stats?.active_rate ?? 0}% 周活跃率`"
      />
      <StatCard
        label="本月用量"
        :value="formatTokens(stats?.monthly_tokens ?? 0)"
        :sub="`tokens · 较上月 +${stats?.monthly_growth ?? 0}%`"
      />
      <StatCard
        label="已限制"
        :value="formatNumber(stats?.restricted ?? 0)"
        sub="需关注 · 可随时解除"
        :tone="(stats?.restricted ?? 0) > 0 ? 'danger' : 'default'"
      />
    </div>

    <div class="users-layout">
      <section class="panel list-panel">
        <div class="toolbar">
          <a-input
            v-model:value="searchInput"
            class="search"
            placeholder="搜索邮箱或昵称"
            allow-clear
            aria-label="搜索用户"
          >
            <template #prefix><SearchOutlined class="search-icon" /></template>
          </a-input>

          <div class="segments" role="tablist" aria-label="用户状态筛选">
            <button
              v-for="segment in segments"
              :key="segment.key"
              type="button"
              class="segment"
              :class="{ active: status === segment.key }"
              role="tab"
              :aria-selected="status === segment.key"
              @click="setStatus(segment.key)"
            >
              {{ segment.label }} <span>{{ segment.count }}</span>
            </button>
          </div>

          <span class="toolbar-spacer" />

          <a-select
            :value="sort"
            :options="[
              { label: '最近活跃', value: 'recent' },
              { label: '注册时间', value: 'created' },
              { label: '用量优先', value: 'usage' },
            ]"
            class="sort-select"
            size="small"
            aria-label="排序方式"
            @change="onSortChange"
          />

          <a-popover trigger="click" placement="bottomRight">
            <a-button size="small" class="filter-btn">
              <FilterOutlined />
              筛选
            </a-button>
            <template #content>
              <div class="filter-panel">
                <p class="filter-title">每页显示</p>
                <a-radio-group v-model:value="pageSize" size="small" @change="onPageSizeChanged">
                  <a-radio-button :value="8">8</a-radio-button>
                  <a-radio-button :value="10">10</a-radio-button>
                  <a-radio-button :value="20">20</a-radio-button>
                </a-radio-group>
                <p class="filter-title">快捷筛选</p>
                <div class="filter-actions">
                  <a-button size="small" block @click="quickFilter('quota')">仅看额度将尽</a-button>
                  <a-button size="small" block @click="quickFilter('restricted')">仅看已限制</a-button>
                  <a-button size="small" type="text" block @click="resetFilters">清除全部筛选</a-button>
                </div>
              </div>
            </template>
          </a-popover>

          <a-button size="small" class="refresh-btn" aria-label="刷新列表" @click="loadList()">
            <ReloadOutlined />
          </a-button>
        </div>

        <div class="table-wrap">
          <table class="user-table">
            <colgroup>
              <col style="width: 34%" />
              <col style="width: 15%" />
              <col style="width: 26%" />
              <col style="width: 15%" />
              <col style="width: 10%" />
            </colgroup>
            <thead>
              <tr>
                <th>用户</th>
                <th>套餐</th>
                <th>本月用量</th>
                <th>状态</th>
                <th class="actions-col">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="loading">
                <td colspan="5" class="loading-cell">
                  <a-skeleton :paragraph="{ rows: 5 }" active />
                </td>
              </tr>
              <tr
                v-for="user in loading ? [] : items"
                :key="user.id"
                :class="{ selected: user.id === selectedId }"
                tabindex="0"
                @click="selectUser(user)"
                @keydown.enter="selectUser(user)"
              >
                <td>
                  <div class="user-cell">
                    <span class="avatar">{{ user.avatar_text }}</span>
                    <div class="user-copy">
                      <strong>
                        {{ user.display_name || user.nickname || user.email }}
                        <span v-if="user.role === 'admin'" class="admin-tag">管理员</span>
                      </strong>
                      <small>{{ user.email }}</small>
                    </div>
                  </div>
                </td>
                <td><PlanTag :plan="user.plan" /></td>
                <td>
                  <div class="usage-cell">
                    <div v-if="!isUnlimited(user)" class="usage-bar">
                      <span
                        :style="{ width: `${Math.min(100, user.used_percent)}%` }"
                        :class="{ warning: user.used_percent >= 90 }"
                      />
                    </div>
                    <div class="usage-meta">
                      <span>{{ formatTokens(user.used_tokens) }} / {{ isUnlimited(user) ? '无限' : formatTokens(user.quota_tokens) }}</span>
                      <small>{{ isUnlimited(user) ? '不限' : formatPercent(user.used_percent) }}</small>
                    </div>
                  </div>
                </td>
                <td>
                  <span class="status" :class="userStatus(user).tone">
                    <i />
                    {{ userStatus(user).label }}
                  </span>
                </td>
                <td class="actions-col" @click.stop>
                  <a-dropdown :trigger="['click']" placement="bottomRight">
                    <button type="button" class="more-btn" aria-label="用户操作">
                      <MoreOutlined />
                    </button>
                    <template #overlay>
                      <a-menu @click="onRowMenu($event, user)">
                        <a-menu-item key="detail">查看详情</a-menu-item>
                        <a-menu-item key="quota">调整额度</a-menu-item>
                        <a-menu-item key="reset">重置密码</a-menu-item>
                        <a-menu-divider />
                        <a-menu-item key="toggle" :danger="!user.disabled">
                          {{ user.disabled ? '恢复登录' : '限制登录' }}
                        </a-menu-item>
                        <a-menu-item key="logout">强制退出全部设备</a-menu-item>
                      </a-menu>
                    </template>
                  </a-dropdown>
                </td>
              </tr>
            </tbody>
          </table>

          <div v-if="!loading && !items.length" class="table-empty">
            <span class="empty-icon"><WarningOutlined /></span>
            <p>没有匹配的用户</p>
            <small>试试更换关键词，或清除状态筛选。</small>
            <a-button v-if="hasFilters" size="small" @click="resetFilters">清除筛选</a-button>
          </div>
        </div>

        <div class="pager">
          <span class="pager-text">
            共 {{ formatNumber(total) }} 位用户 · 第 {{ page }} / {{ totalPages }} 页
          </span>
          <a-pagination
            :current="page"
            :page-size="pageSize"
            :total="total"
            size="small"
            :show-size-changer="false"
            :show-quick-jumper="false"
            @change="onPageChange"
          />
        </div>
      </section>

      <div v-if="wide" class="detail-col">
        <UserDetailPanel
          :user="selectedUser"
          :loading="detailLoading"
          :busy="detailBusy"
          @quota="openQuota"
          @reset="openReset"
          @logout-all="onLogoutAll"
          @toggle-disabled="onToggleDisabled"
          @change-plan="onPlanChange"
          @close="selectedUser = null"
        />
      </div>
    </div>

    <a-drawer
      v-if="!wide"
      v-model:open="drawerOpen"
      placement="right"
      :width="340"
      :closable="false"
      root-class-name="user-detail-drawer"
    >
      <UserDetailPanel
        :user="selectedUser"
        :loading="detailLoading"
        :busy="detailBusy"
        @quota="openQuota"
        @reset="openReset"
        @logout-all="onLogoutAll"
        @toggle-disabled="onToggleDisabled"
        @change-plan="onPlanChange"
        @close="drawerOpen = false"
      />
    </a-drawer>

    <QuotaModal v-model:open="quotaOpen" :user="selectedUser" @saved="onQuotaSaved" />
    <ResetPasswordModal v-model:open="resetOpen" :user="selectedUser" @done="loadList" />
    <NewUserModal v-model:open="newUserOpen" @created="onUserCreated" />
  </div>
</template>

<style scoped>
.users-page {
  max-width: 1440px;
}

.stat-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 16px;
  margin-bottom: 16px;
}

.users-layout {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 312px;
  gap: 16px;
  align-items: start;
}

.list-panel {
  padding: 0;
  border: 1px solid var(--dt-line);
  border-radius: var(--dt-radius-card);
  background: var(--dt-surface);
  box-shadow: var(--dt-shadow-card);
  overflow: hidden;
}

.toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 14px;
  border-bottom: 1px solid var(--dt-line-soft);
  flex-wrap: wrap;
}

.search {
  width: 280px;
}

.search :deep(.ant-input-affix-wrapper) {
  height: 32px;
  border-radius: 8px;
  background: var(--dt-sidebar);
  border-color: var(--dt-line);
}

.search :deep(input) {
  font-size: 11.4px;
}

.search-icon {
  color: var(--dt-text-faint);
  font-size: 11px;
}

.segments {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  padding: 2px;
  border-radius: 8px;
  background: var(--dt-hover);
}

.segment {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: 26px;
  padding: 0 9px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--dt-text-muted);
  font-size: 10px;
  font-weight: 550;
  cursor: pointer;
}

.segment span {
  color: var(--dt-text-faint);
  font-size: 9px;
}

.segment.active {
  background: var(--dt-surface);
  color: var(--dt-text);
  box-shadow: 0 1px 2px rgba(16, 24, 40, 0.08);
}

.toolbar-spacer {
  flex: 1;
}

.sort-select {
  width: 108px;
}

.sort-select :deep(.ant-select-selector) {
  height: 30px !important;
  border-radius: 7px !important;
  font-size: 10.2px;
}

.filter-btn,
.refresh-btn {
  height: 30px;
  border-radius: 7px;
  font-size: 10.2px;
}

.filter-btn :deep(.anticon) {
  font-size: 10px;
}

.filter-panel {
  width: 200px;
}

.filter-title {
  margin: 8px 0 6px;
  color: var(--dt-text-faint);
  font-size: 9.8px;
  font-weight: 600;
}

.filter-title:first-child {
  margin-top: 0;
}

.filter-actions {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.table-wrap {
  overflow-x: auto;
}

.user-table {
  width: 100%;
  border-collapse: collapse;
  table-layout: fixed;
}

.user-table th {
  padding: 9px 12px;
  border-bottom: 1px solid var(--dt-line-soft);
  color: var(--dt-text-faint);
  font-size: 9.8px;
  font-weight: 550;
  text-align: left;
  white-space: nowrap;
}

.user-table td {
  padding: 8px 12px;
  border-bottom: 1px solid var(--dt-line-soft);
  vertical-align: middle;
}

.user-table tbody tr {
  height: 56px;
  cursor: pointer;
  outline: none;
  transition: background 0.12s ease;
}

.user-table tbody tr:hover {
  background: var(--dt-brand-softer);
}

.user-table tbody tr.selected {
  background: var(--dt-brand-softer);
  box-shadow: inset 2px 0 0 var(--dt-brand);
}

.user-table tbody tr:last-child td {
  border-bottom: none;
}

.user-cell {
  display: flex;
  align-items: center;
  gap: 9px;
  min-width: 0;
}

.avatar {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border-radius: 50%;
  background: var(--dt-brand-soft);
  color: var(--dt-brand-hover);
  font-size: 11.8px;
  font-weight: 700;
  flex: 0 0 auto;
}

.user-copy {
  display: flex;
  flex-direction: column;
  gap: 1px;
  min-width: 0;
}

.user-copy strong {
  display: flex;
  align-items: center;
  gap: 5px;
  color: var(--dt-text);
  font-size: 11.8px;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.admin-tag {
  padding: 1px 5px;
  border-radius: 4px;
  background: var(--dt-brand-soft);
  color: var(--dt-brand-hover);
  font-size: 8.2px;
  font-weight: 600;
}

.user-copy small {
  color: var(--dt-text-faint);
  font-size: 9.6px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.usage-cell {
  display: flex;
  align-items: center;
  gap: 9px;
  min-width: 0;
}

.usage-bar {
  width: 62px;
  height: 4px;
  border-radius: 999px;
  background: var(--dt-line);
  overflow: hidden;
  flex: 0 0 auto;
}

.usage-bar span {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: var(--dt-brand);
}

.usage-bar span.warning {
  background: var(--dt-warning);
}

.usage-meta {
  display: flex;
  flex-direction: column;
  gap: 1px;
  min-width: 0;
}

.usage-meta span {
  color: var(--dt-text-secondary);
  font-size: 10px;
  white-space: nowrap;
}

.usage-meta small {
  color: var(--dt-text-faint);
  font-size: 8.8px;
}

.status {
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

.status i {
  width: 4px;
  height: 4px;
  border-radius: 50%;
  background: currentColor;
}

.status.normal {
  background: var(--dt-success-soft);
  color: var(--dt-success-text);
}

.status.warning {
  background: var(--dt-warning-soft);
  color: var(--dt-warning-text);
}

.status.danger {
  background: var(--dt-danger-soft);
  color: var(--dt-danger-text);
}

.actions-col {
  text-align: right;
}

.more-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--dt-text-faint);
  font-size: 13px;
  cursor: pointer;
}

.more-btn:hover {
  background: var(--dt-hover);
  color: var(--dt-text-secondary);
}

.loading-cell {
  padding: 20px !important;
}

.table-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 5px;
  padding: 40px 20px;
  text-align: center;
}

.empty-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 38px;
  height: 38px;
  margin-bottom: 3px;
  border-radius: 10px;
  background: var(--dt-warning-soft);
  color: var(--dt-warning-text);
  font-size: 16px;
}

.table-empty p {
  margin: 0;
  color: var(--dt-text-secondary);
  font-size: 12.2px;
  font-weight: 600;
}

.table-empty small {
  margin-bottom: 4px;
  color: var(--dt-text-faint);
  font-size: 9.8px;
}

.pager {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 14px;
  border-top: 1px solid var(--dt-line-soft);
}

.pager-text {
  color: var(--dt-text-faint);
  font-size: 9.8px;
}

.detail-col {
  position: sticky;
  top: 0;
}

.detail-col :deep(.user-detail) {
  max-height: calc(100vh - 120px);
  overflow-y: auto;
}

@media (max-width: 1280px) {
  .stat-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .users-layout {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 900px) {
  .search {
    width: 100%;
  }

  .segments {
    width: 100%;
    overflow-x: auto;
  }

  .toolbar-spacer {
    display: none;
  }
}

@media (max-width: 640px) {
  .stat-grid {
    grid-template-columns: 1fr;
  }

  .user-table colgroup col:nth-child(2),
  .user-table colgroup col:nth-child(4) {
    width: 0;
  }

  .user-table th:nth-child(2),
  .user-table td:nth-child(2),
  .user-table th:nth-child(4),
  .user-table td:nth-child(4) {
    display: none;
  }

  .pager {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>
