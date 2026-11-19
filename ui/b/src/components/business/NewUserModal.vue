<script setup lang="ts">
/** 管理员建号：注册关闭后仍可创建账号。 */
import { computed, reactive, ref, watch } from 'vue'
import { message } from 'ant-design-vue'
import { ApiError, PLAN_QUOTAS } from '@deeptalk/shared'
import { createUser } from '@/services/admin'

const open = defineModel<boolean>('open', { default: false })
const emit = defineEmits<{ (event: 'created'): void }>()

const saving = ref(false)
const error = ref('')

const form = reactive({
  email: '',
  password: '',
  nickname: '',
  plan: 'free' as 'free' | 'pro' | 'team' | 'unlimited',
  quotaM: 4,
})

const planOptions = [
  { label: '免费版 · 4M tokens / 月', value: 'free' },
  { label: '专业版 · 20M tokens / 月', value: 'pro' },
  { label: '团队版 · 50M tokens / 月', value: 'team' },
  { label: '无限版 · 不限 tokens', value: 'unlimited' },
]

const passwordValid = computed(
  () => form.password.length >= 8 && /[A-Za-z]/.test(form.password) && /\d/.test(form.password),
)

watch(open, (value) => {
  if (!value) return
  form.email = ''
  form.password = ''
  form.nickname = ''
  form.plan = 'free'
  form.quotaM = 4
  error.value = ''
})

/** 切换套餐时同步默认额度。 */
function onPlanChange(plan: 'free' | 'pro' | 'team' | 'unlimited'): void {
  form.plan = plan
  form.quotaM = (PLAN_QUOTAS[plan] ?? 4_000_000) / 1_000_000
}

/** 创建账号。 */
async function save(): Promise<void> {
  error.value = ''
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.email.trim())) {
    error.value = '请输入正确的邮箱地址'
    return
  }
  if (!passwordValid.value) {
    error.value = '密码至少 8 位，且需同时包含字母与数字'
    return
  }
  saving.value = true
  try {
    await createUser({
      email: form.email.trim(),
      password: form.password,
      nickname: form.nickname.trim(),
      plan: form.plan,
      quota_tokens: Math.round(form.quotaM * 1_000_000),
    })
    message.success('账号已创建')
    emit('created')
    open.value = false
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : '创建失败，请稍后重试'
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <a-modal v-model:open="open" title="新建账号" :width="460" :footer="null" destroy-on-close>
    <a-form layout="vertical" class="new-user">
      <a-form-item label="邮箱" required>
        <a-input v-model:value="form.email" placeholder="user@company.com" :maxlength="190" />
      </a-form-item>
      <a-form-item label="初始密码" required>
        <a-input-password v-model:value="form.password" placeholder="至少 8 位，包含字母与数字" autocomplete="new-password" />
        <p class="hint">密码使用 Argon2id 哈希，创建后不会以明文保存。</p>
      </a-form-item>
      <a-form-item label="昵称">
        <a-input v-model:value="form.nickname" placeholder="留空则取邮箱前缀" :maxlength="32" />
      </a-form-item>
      <div class="two-col">
        <a-form-item label="套餐">
          <a-select :value="form.plan" :options="planOptions" @change="onPlanChange" />
        </a-form-item>
        <a-form-item label="额度（M tokens）">
          <a-input-number v-if="form.plan !== 'unlimited'" v-model:value="form.quotaM" :min="1" :max="100" style="width: 100%" />
          <div v-else class="unlimited-hint">无限版不设月度 tokens 上限</div>
        </a-form-item>
      </div>

      <div v-if="error" class="form-error" role="alert">{{ error }}</div>

      <div class="actions">
        <a-button @click="open = false">取消</a-button>
        <a-button type="primary" :loading="saving" @click="save">创建账号</a-button>
      </div>
    </a-form>
  </a-modal>
</template>

<style scoped>
.new-user :deep(.ant-form-item) {
  margin-bottom: 13px;
}

.new-user :deep(.ant-form-item-label) {
  padding-bottom: 5px;
}

.new-user :deep(.ant-form-item-label > label) {
  color: var(--dt-text-secondary);
  font-size: 11.4px;
  font-weight: 600;
}

.two-col {
  display: grid;
  grid-template-columns: 1.4fr 1fr;
  gap: 12px;
}

.hint {
  margin: 4px 0 0;
  color: var(--dt-text-faint);
  font-size: 9.8px;
}

.unlimited-hint {
  display: flex;
  align-items: center;
  height: 32px;
  padding: 0 11px;
  border-radius: 8px;
  background: #fff7ed;
  color: #b93815;
  font-size: 11.5px;
}

.form-error {
  padding: 8px 10px;
  border-radius: 8px;
  background: var(--dt-danger-soft);
  color: var(--dt-danger-text);
  font-size: 11px;
}

.actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 6px;
}
</style>
