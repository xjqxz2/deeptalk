<script setup lang="ts">
/** 注册页：无验证邮件，密码实时校验，注册成功即登录；首个账号为管理员。 */
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import {
  CheckCircleFilled,
  InfoCircleOutlined,
  LockOutlined,
  MailOutlined,
  SafetyCertificateOutlined,
} from '@ant-design/icons-vue'
import { ApiError } from '@deeptalk/shared'
import type { AuthConfig } from '@deeptalk/shared'
import AuthLayout from '@/layouts/AuthLayout.vue'
import { fetchAuthConfig } from '@/services/auth'
import { useUserStore } from '@/stores/user'

const router = useRouter()
const userStore = useUserStore()

const email = ref('')
const password = ref('')
const confirm = ref('')
const error = ref('')
const loading = ref(false)
const confirmTouched = ref(false)
const config = ref<AuthConfig | null>(null)

/** 注册开关；未知时先按开放处理，避免闪烁。 */
const closed = computed(() => config.value !== null && !config.value.allow_registration)

const hasLength = computed(() => password.value.length >= 8)
const hasMixed = computed(() => /[A-Za-z]/.test(password.value) && /\d/.test(password.value))
const matched = computed(() => confirm.value.length > 0 && password.value === confirm.value)
const confirmError = computed(() =>
  confirmTouched.value && confirm.value.length > 0 && !matched.value ? '两次输入的密码不一致' : '',
)
const valid = computed(
  () =>
    /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.value.trim()) &&
    hasLength.value &&
    hasMixed.value &&
    matched.value,
)

/** 首个账号提示。 */
const adminHint = computed(() =>
  config.value && !config.value.has_users
    ? '当前还没有账号：本次注册将成为平台管理员。'
    : '注册成功即登录；首个账号为平台管理员。',
)

onMounted(async () => {
  try {
    config.value = await fetchAuthConfig()
  } catch {
    // 配置接口失败时按开放注册处理，提交阶段仍会得到服务端真实结果
    config.value = { allow_registration: true, has_users: true, site_name: 'DeepTalk', web_search_enabled: true }
  }
})

/** 提交注册。 */
async function onSubmit(): Promise<void> {
  error.value = ''
  confirmTouched.value = true
  if (!valid.value) {
    error.value = '请检查邮箱、密码与确认密码'
    return
  }
  loading.value = true
  try {
    await userStore.register(email.value.trim(), password.value)
    message.success('账号创建成功，已进入工作台')
    await router.push('/')
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : '注册失败，请稍后重试'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <AuthLayout
    eyebrow="CREATE ACCOUNT"
    title="创建账号"
    subtitle="使用邮箱注册，注册后直接进入工作台。"
    switch-text="已有账号？"
    switch-label="去登录"
    switch-to="/login"
  >
    <div v-if="closed" class="closed-state">
      <span class="closed-icon"><LockOutlined /></span>
      <h2>注册已关闭</h2>
      <p>当前仅支持管理员创建账号，请联系平台管理员开通。</p>
      <a-button type="primary" block class="submit-btn" @click="router.push('/login')">返回登录</a-button>
    </div>

    <form v-else class="auth-form" novalidate @submit.prevent="onSubmit">
      <div class="field">
        <label class="field-label" for="register-email">邮箱</label>
        <a-input
          id="register-email"
          v-model:value="email"
          class="auth-input"
          size="large"
          placeholder="you@company.com"
          autocomplete="email"
        >
          <template #prefix><MailOutlined class="field-icon" /></template>
        </a-input>
      </div>

      <div class="field">
        <label class="field-label" for="register-password">密码</label>
        <a-input-password
          id="register-password"
          v-model:value="password"
          class="auth-input"
          size="large"
          placeholder="至少 8 位，包含字母与数字"
          autocomplete="new-password"
        >
          <template #prefix><LockOutlined class="field-icon" /></template>
        </a-input-password>
      </div>

      <div class="field">
        <label class="field-label" for="register-confirm">确认密码</label>
        <a-input-password
          id="register-confirm"
          v-model:value="confirm"
          class="auth-input"
          :class="{ invalid: Boolean(confirmError) }"
          size="large"
          placeholder="再次输入密码"
          autocomplete="new-password"
          @blur="confirmTouched = true"
          @press-enter="onSubmit"
        >
          <template #prefix><LockOutlined class="field-icon" /></template>
        </a-input-password>
        <p v-if="confirmError" class="field-error" role="alert">{{ confirmError }}</p>
      </div>

      <ul class="rules">
        <li :class="{ passed: hasLength }">
          <CheckCircleFilled />
          至少 8 位
        </li>
        <li :class="{ passed: hasMixed }">
          <CheckCircleFilled />
          字母 + 数字
        </li>
        <li class="passed">
          <CheckCircleFilled />
          无需邮箱验证
        </li>
      </ul>

      <div v-if="error" class="form-error" role="alert" aria-live="polite">{{ error }}</div>

      <a-button type="primary" html-type="submit" block class="submit-btn" :loading="loading" :disabled="!valid || loading">
        创建账号并进入
      </a-button>

      <p class="admin-hint">
        <InfoCircleOutlined />
        {{ adminHint }}
      </p>

      <p class="note">
        <LockOutlined />
        不发送验证邮件 · 忘记密码由管理员重置
      </p>

      <div class="security">
        <SafetyCertificateOutlined />
        密码使用 Argon2id 哈希，前端不保存明文
      </div>
    </form>
  </AuthLayout>
</template>

<style scoped>
.auth-form {
  display: flex;
  flex-direction: column;
}

.field {
  margin-bottom: 15px;
}

.field-label {
  display: block;
  margin-bottom: 8px;
  color: var(--dt-text-secondary);
  font-size: 12px;
  font-weight: 620;
}

.field-icon {
  color: var(--dt-text-faint);
  font-size: 13px;
}

.auth-input :deep(.ant-input-affix-wrapper),
.auth-input :deep(.ant-input) {
  min-height: 46px;
  border-radius: 10px;
  font-size: 13.2px;
}

.auth-input :deep(.ant-input-affix-wrapper) {
  padding-inline: 13px;
}

.auth-input :deep(.ant-input-prefix) {
  margin-inline-end: 9px;
}

.auth-input.invalid :deep(.ant-input-affix-wrapper) {
  border-color: var(--dt-danger);
}

.field-error {
  margin: 6px 0 0;
  color: var(--dt-danger-text);
  font-size: 10.8px;
}

.rules {
  display: flex;
  align-items: center;
  gap: 16px;
  margin: 0 0 18px;
  padding: 0;
  list-style: none;
  flex-wrap: wrap;
}

.rules li {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  color: var(--dt-text-faint);
  font-size: 10.2px;
}

.rules li :deep(svg) {
  color: #d0d5dd;
  font-size: 10px;
}

.rules li.passed {
  color: var(--dt-text-muted);
}

.rules li.passed :deep(svg) {
  color: var(--dt-success);
}

.form-error {
  margin: -4px 0 12px;
  padding: 9px 12px;
  border: 1px solid #fee4e2;
  border-radius: 9px;
  background: var(--dt-danger-soft);
  color: var(--dt-danger-text);
  font-size: 11.8px;
  line-height: 18px;
}

.submit-btn {
  height: 46px;
  border-radius: 10px;
  font-size: 13.6px;
  font-weight: 620;
}

.admin-hint {
  display: flex;
  align-items: flex-start;
  gap: 6px;
  margin: 14px 0 0;
  padding: 9px 11px;
  border-radius: 8px;
  background: var(--dt-brand-softer);
  color: var(--dt-text-muted);
  font-size: 10.4px;
  line-height: 16px;
}

.admin-hint :deep(svg) {
  color: var(--dt-brand);
  margin-top: 1px;
}

.note {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 18px 0 0;
  padding-top: 15px;
  border-top: 1px solid var(--dt-line-soft);
  color: var(--dt-text-muted);
  font-size: 10.6px;
}

.security {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 12px;
  color: var(--dt-text-faint);
  font-size: 10.2px;
}

.closed-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  padding: 28px 0 12px;
}

.closed-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 52px;
  height: 52px;
  margin-bottom: 16px;
  border-radius: 14px;
  background: var(--dt-warning-soft);
  color: var(--dt-warning-text);
  font-size: 20px;
}

.closed-state h2 {
  margin: 0 0 8px;
  color: var(--dt-text);
  font-size: 20px;
  font-weight: 700;
}

.closed-state p {
  margin: 0 0 22px;
  color: var(--dt-text-muted);
  font-size: 12.6px;
  line-height: 20px;
}

@media (max-width: 1024px) {
  /* iOS Safari 在小于 16px 的输入框聚焦时会自动放大页面 */
  .auth-input :deep(.ant-input-affix-wrapper),
  .auth-input :deep(.ant-input) {
    font-size: 16px;
  }
}

@media (max-width: 480px) {
  .auth-input :deep(.ant-input-affix-wrapper),
  .auth-input :deep(.ant-input) {
    min-height: 44px;
  }

  .submit-btn {
    height: 44px;
  }
}
</style>
