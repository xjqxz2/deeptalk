<script setup lang="ts">
/**
 * 套餐管理：三档套餐的限额随时可调。
 * 两类锁止条件：自然月消费上限（元）与 5 小时 / 7 天 / 30 天的请求次数上限。
 * 限额填 0 表示该条不限制。
 */
import { onMounted, ref } from 'vue'
import { message } from 'ant-design-vue'
import { get, put } from '@deeptalk/shared'

interface Plan {
  id: number
  code: string
  name: string
  description: string
  spend_limit_cents: number
  limit_short: number
  limit_week: number
  limit_month: number
  enabled: boolean
}

const plans = ref<Plan[]>([])
const loading = ref(false)
const saving = ref<number | null>(null)

/** 元与分的换算。 */
const toYuan = (cents: number) => (cents / 100).toFixed(2)
const toCents = (yuan: number | string) => Math.round(Number(yuan || 0) * 100)

async function load(): Promise<void> {
  loading.value = true
  try {
    const data = await get<{ items: Plan[] }>('/b/plans')
    plans.value = data.items
  } catch (error) {
    message.error(error instanceof Error ? error.message : '加载套餐失败')
  } finally {
    loading.value = false
  }
}

async function save(plan: Plan, patch: Record<string, number | boolean>): Promise<void> {
  saving.value = plan.id
  try {
    const data = await put<{ plan: Plan }>(`/b/plans/${plan.id}`, patch)
    Object.assign(plan, data.plan)
    message.success('已保存')
  } catch (error) {
    message.error(error instanceof Error ? error.message : '保存失败')
  } finally {
    saving.value = null
  }
}

onMounted(load)
</script>

<template>
  <div class="plans-page">
    <div class="page-head">
      <div>
        <h1 class="page-title">套餐与限额</h1>
        <p class="page-desc">
          每个套餐都有两类锁止条件：自然月消费上限，以及 5 小时 / 7 天 / 30 天内的请求次数上限。任一条触发即停止新一轮请求。
        </p>
      </div>
    </div>

    <a-spin :spinning="loading">
      <div class="plan-grid">
        <a-card v-for="plan in plans" :key="plan.id" class="plan-card" :title="`${plan.name} · ${plan.code}`">
          <template #extra>
            <a-switch
              :checked="plan.enabled"
              size="small"
              @change="(value: boolean) => save(plan, { enabled: value })"
            />
          </template>

          <div class="field">
            <span class="label">月消费上限（元）</span>
            <a-input-number
              :value="Number(toYuan(plan.spend_limit_cents))"
              :min="0"
              :step="1"
              :precision="2"
              @change="(value: number) => save(plan, { spend_limit_cents: toCents(value) })"
            />
          </div>
          <div class="field">
            <span class="label">5 小时请求上限</span>
            <a-input-number
              :value="plan.limit_short"
              :min="0"
              :precision="0"
              @change="(value: number) => save(plan, { limit_short: Number(value) || 0 })"
            />
          </div>
          <div class="field">
            <span class="label">7 天请求上限</span>
            <a-input-number
              :value="plan.limit_week"
              :min="0"
              :precision="0"
              @change="(value: number) => save(plan, { limit_week: Number(value) || 0 })"
            />
          </div>
          <div class="field">
            <span class="label">30 天请求上限</span>
            <a-input-number
              :value="plan.limit_month"
              :min="0"
              :precision="0"
              @change="(value: number) => save(plan, { limit_month: Number(value) || 0 })"
            />
          </div>

          <p class="hint">0 表示该条不限制；保存后立即对该套餐的所有用户生效。</p>
        </a-card>
      </div>
    </a-spin>
  </div>
</template>

<style scoped>
.plans-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.page-title {
  margin: 0;
  color: var(--dt-text);
  font-size: 20px;
  font-weight: 700;
}

.page-desc {
  margin: 6px 0 0;
  color: var(--dt-text-muted);
  font-size: 13px;
}

.plan-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 16px;
}

.plan-card {
  border-radius: 14px;
}

.field {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
}

.label {
  color: var(--dt-text-secondary);
  font-size: 13px;
}

.hint {
  margin: 4px 0 0;
  color: var(--dt-text-faint);
  font-size: 12px;
}
</style>
